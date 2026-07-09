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
		{"ready", &clean, &open, true, model.ActionReviewPR},
		{"remote only", nil, &open, true, model.ActionReviewPR},
		{"pending allowed", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.CI = model.CIPending }), true, model.ActionReviewPR},
		{"failure", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.CI = model.CIFailure }), false, model.ActionFixCI},
		{"draft", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.IsDraft = true }), false, model.ActionMarkReady},
		{"changes requested", &clean, mutatePR(open, func(pr *model.PullRequest) { pr.ReviewDecision = "CHANGES_REQUESTED" }), false, model.ActionAddressReview},
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
			lanes := Build(repo, locals, pullRequests, 24*time.Hour, now)
			if len(lanes) != 1 {
				t.Fatalf("lanes = %#v", lanes)
			}
			if lanes[0].ReviewReady != test.ready || lanes[0].NextAction != test.action {
				t.Fatalf("lane ready/action = %t/%q, want %t/%q; lane=%#v", lanes[0].ReviewReady, lanes[0].NextAction, test.ready, test.action, lanes[0])
			}
		})
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
	lanes := Build(repo, locals, pullRequests, time.Hour, now)
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
