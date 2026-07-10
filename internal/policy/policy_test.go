package policy

import (
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestBuildPolicy(t *testing.T) {
	now := time.Date(2026, 7, 9, 16, 0, 0, 0, time.UTC)
	repo := config.Repository{Name: "example", GitHub: "owner/example", Base: "main", Remote: "origin"}
	clean := gitscan.LocalLane{
		ID: "git:owner/example@feature", Branch: "feature", Publication: model.PublicationPublished,
		Worktree: model.Worktree{Path: "/tmp/feature", HeadOID: "abc", Upstream: "origin/feature", AheadBase: 1, UpdatedAt: now},
	}
	open := model.PullRequest{
		Number: 1, Title: "Feature", URL: "https://example.test/1", HeadRefName: "feature", HeadRefOID: "abc",
		BaseRefName: "main", UpdatedAt: now, CI: model.CISuccess, MergeState: "CLEAN", Mergeable: "MERGEABLE",
	}

	tests := []struct {
		name        string
		local       *gitscan.LocalLane
		pullRequest *model.PullRequest
		ready       bool
		action      model.Action
	}{
		{"ready", &clean, &open, true, model.ActionManualTestMerge},
		{"remote only", nil, &open, true, model.ActionManualTestMerge},
		{"pending allowed", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.CI = model.CIPending }), true, model.ActionWaitForCI},
		{"approved merge", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.ReviewDecision = "APPROVED" }), true, model.ActionMergePR},
		{"review required", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.ReviewDecision = "REVIEW_REQUIRED" }), true, model.ActionReviewPR},
		{"required review supersedes historical approval", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.ReviewDecision = "REVIEW_REQUIRED"; pr.Feedback.Approvals = 1 }), true, model.ActionReviewPR},
		{"merge blocked needs review", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.MergeState = "BLOCKED" }), true, model.ActionReviewPR},
		{"failure", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.CI = model.CIFailure }), false, model.ActionFixCI},
		{"draft", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.IsDraft = true }), false, model.ActionMarkReady},
		{"changes requested", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.ReviewDecision = "CHANGES_REQUESTED" }), false, model.ActionAddressReview},
		{"changes requested without protection decision", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.Feedback.ChangesRequested = 1 }), false, model.ActionAddressReview},
		{"unresolved thread", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.Feedback.UnresolvedThreads = 1 }), false, model.ActionAddressReview},
		{"merge conflict", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.Mergeable = "CONFLICTING" }), false, model.ActionResolveConflict},
		{"dirty local", mutateLocal(clean, func(lane *gitscan.LocalLane) { lane.Worktree.Unstaged = 1 }), &open, false, model.ActionInspectLocal},
		{"unpushed", mutateLocal(clean, func(lane *gitscan.LocalLane) { lane.Publication = model.PublicationUnpushed }), nil, false, model.ActionPushBranch},
		{"needs PR", &clean, nil, false, model.ActionCreatePR},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var locals []gitscan.LocalLane
			if test.local != nil {
				locals = append(locals, *test.local)
			}
			var pullRequests []model.PullRequest
			if test.pullRequest != nil {
				pullRequests = append(pullRequests, *test.pullRequest)
			}
			lanes := Build(repo, locals, pullRequests, nil, nil, 24*time.Hour, now)
			if len(lanes) != 1 {
				t.Fatalf("lanes = %#v", lanes)
			}
			if lanes[0].ReviewReady != test.ready || lanes[0].NextAction != test.action {
				t.Fatalf("lane ready/action = %t/%q, want %t/%q; lane=%#v", lanes[0].ReviewReady, lanes[0].NextAction, test.ready, test.action, lanes[0])
			}
		})
	}
}

func TestBuildCorrelatesIssuesAndProgress(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	repo := config.Repository{Name: "example", GitHub: "owner/example", Base: "main"}
	issues := []model.Issue{
		{Number: 4, Title: "Active feature", URL: "https://github.com/owner/example/issues/4", UpdatedAt: now},
		{Number: 9, Title: "Old queued work", URL: "https://github.com/owner/example/issues/9", UpdatedAt: now.Add(-48 * time.Hour)},
	}
	locals := []gitscan.LocalLane{{
		ID: "git:owner/example@GH-4", Branch: "GH-4", Publication: model.PublicationPublished,
		Worktree: model.Worktree{Path: "/tmp/GH-4", HeadOID: "abc", AheadBase: 1, UpdatedAt: now},
	}}
	progressByIssue := map[int]model.Progress{4: {Source: "kit", FeatureID: "0004", Phase: "implement"}}
	lanes := Build(repo, locals, nil, issues, progressByIssue, 24*time.Hour, now)
	if len(lanes) != 2 {
		t.Fatalf("lanes = %#v", lanes)
	}
	if lanes[0].Issue == nil || lanes[0].Issue.Number != 4 || lanes[0].Progress == nil || lanes[0].Progress.FeatureID != "0004" || lanes[0].NextAction != model.ActionCreatePR {
		t.Fatalf("local issue lane = %#v", lanes[0])
	}
	if lanes[1].Issue == nil || lanes[1].Issue.Number != 9 || lanes[1].NextAction != model.ActionResumeOrClose {
		t.Fatalf("queued issue lane = %#v", lanes[1])
	}
}

func TestBuildCorrelatesKitIssueReferenceFromBranchWithoutScopedIssue(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	repo := config.Repository{Name: "example", GitHub: "owner/example", Base: "main"}
	local := gitscan.LocalLane{
		ID: "git:owner/example@GH-12", Branch: "GH-12", Publication: model.PublicationPublished,
		Worktree: model.Worktree{Path: "/tmp/GH-12", HeadOID: "abc", AheadBase: 1, UpdatedAt: now},
	}
	lanes := Build(repo, []gitscan.LocalLane{local}, nil, nil, map[int]model.Progress{12: {Source: "kit", FeatureID: "0012"}}, time.Hour, now)
	if len(lanes) != 1 || lanes[0].Issue != nil || lanes[0].Progress == nil || lanes[0].Progress.FeatureID != "0012" {
		t.Fatalf("lanes = %#v", lanes)
	}
}

func TestBuildKeepsSameBranchWorktreesDistinct(t *testing.T) {
	now := time.Now()
	repo := config.Repository{Name: "example", GitHub: "owner/example", Base: "main"}
	locals := []gitscan.LocalLane{
		{ID: "one", Branch: "feature", Publication: model.PublicationPublished, Worktree: model.Worktree{Path: "/one", HeadOID: "one", UpdatedAt: now}},
		{ID: "two", Branch: "feature", Publication: model.PublicationPublished, Worktree: model.Worktree{Path: "/two", HeadOID: "two", UpdatedAt: now}},
	}
	pullRequests := []model.PullRequest{{Number: 2, HeadRefName: "feature", HeadRefOID: "two", BaseRefName: "main", UpdatedAt: now, CI: model.CINone, MergeState: "CLEAN"}}
	lanes := Build(repo, locals, pullRequests, nil, nil, time.Hour, now)
	if len(lanes) != 2 || lanes[0].Worktree == nil || lanes[0].Worktree.Path != "/two" {
		t.Fatalf("lanes = %#v", lanes)
	}
}

func mutatePR(value model.PullRequest, mutate func(*model.PullRequest)) *model.PullRequest {
	mutate(&value)
	return &value
}

func mutateLocal(value gitscan.LocalLane, mutate func(*gitscan.LocalLane)) *gitscan.LocalLane {
	mutate(&value)
	return &value
}
