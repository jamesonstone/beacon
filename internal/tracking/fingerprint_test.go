package tracking

import (
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestFingerprintIsOrderAndPolicyTimeIndependent(t *testing.T) {
	project, lanes := trackingFixture()
	first, err := Fingerprint(project, lanes)
	if err != nil {
		t.Fatal(err)
	}
	lanes[0], lanes[1] = lanes[1], lanes[0]
	for index := range lanes {
		lanes[index].Signals.Freshness = model.FreshnessStale
		lanes[index].NextAction = model.ActionResumeOrClose
		lanes[index].UpdatedAt = lanes[index].UpdatedAt.Add(72 * time.Hour)
	}
	second, err := Fingerprint(project, lanes)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("policy-only changes altered fingerprint: %s != %s", first, second)
	}
}

func TestFingerprintChangesForSupportedEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func([]model.Lane)
	}{
		{name: "head", mutate: func(lanes []model.Lane) { lanes[0].Worktree.HeadOID = "new-head" }},
		{name: "status", mutate: func(lanes []model.Lane) { lanes[0].Worktree.StatusHash = "new-status" }},
		{name: "worktree counts", mutate: func(lanes []model.Lane) { lanes[0].Worktree.Untracked++ }},
		{name: "upstream publication", mutate: func(lanes []model.Lane) {
			lanes[0].Worktree.Upstream = "origin/main"
			lanes[0].Signals.Publication = model.PublicationPublished
		}},
		{name: "pull request update", mutate: func(lanes []model.Lane) {
			lanes[1].PullRequest.UpdatedAt = lanes[1].PullRequest.UpdatedAt.Add(time.Minute)
		}},
		{name: "pull request draft", mutate: func(lanes []model.Lane) { lanes[1].PullRequest.IsDraft = true }},
		{name: "checks", mutate: func(lanes []model.Lane) { lanes[1].PullRequest.Checks.Pending++ }},
		{name: "review feedback", mutate: func(lanes []model.Lane) { lanes[1].PullRequest.Feedback.UnresolvedThreads++ }},
		{name: "review decision", mutate: func(lanes []model.Lane) { lanes[1].PullRequest.ReviewDecision = "APPROVED" }},
		{name: "merge state", mutate: func(lanes []model.Lane) { lanes[1].PullRequest.Mergeable = "CONFLICTING" }},
		{name: "linked issue", mutate: func(lanes []model.Lane) {
			lanes[1].PullRequest.ClosingIssues = []model.Issue{{Number: 3, Title: "Linked", UpdatedAt: lanes[1].UpdatedAt}}
		}},
		{name: "issue update", mutate: func(lanes []model.Lane) { lanes[1].Issue.UpdatedAt = lanes[1].Issue.UpdatedAt.Add(time.Minute) }},
		{name: "issue labels", mutate: func(lanes []model.Lane) { lanes[1].Issue.Labels = append(lanes[1].Issue.Labels, "new") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			project, lanes := trackingFixture()
			before, err := Fingerprint(project, lanes)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(lanes)
			after, err := Fingerprint(project, lanes)
			if err != nil {
				t.Fatal(err)
			}
			if before == after {
				t.Fatal("evidence change did not alter fingerprint")
			}
		})
	}
}

func trackingFixture() (model.Project, []model.Lane) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	project := model.Project{Name: "repo", Path: "/repo", GitHub: "owner/repo", Base: "main"}
	local := model.Lane{
		ID: "git:owner/repo@main", Repository: "repo", GitHub: "owner/repo", Base: "main", Branch: "main",
		Worktree:  &model.Worktree{Path: "/repo", HeadOID: "head", StatusHash: "status", UpdatedAt: now},
		Signals:   model.Signals{Worktree: model.WorktreeClean, Publication: model.PublicationBase, Freshness: model.FreshnessCurrent},
		UpdatedAt: now,
	}
	remote := model.Lane{
		ID: "gh:owner/repo#2", Repository: "repo", GitHub: "owner/repo", Base: "main", Branch: "feature",
		PullRequest: &model.PullRequest{
			Number: 2, Title: "Feature", HeadRefName: "feature", HeadRefOID: "pr-head", UpdatedAt: now,
			CI: model.CISuccess, Checks: model.CheckSummary{Total: 1, Success: 1}, Feedback: model.Feedback{},
		},
		Issue:     &model.Issue{Number: 1, Title: "Issue", Labels: []string{"b", "a"}, Assignees: []string{"z", "a"}, UpdatedAt: now},
		Signals:   model.Signals{PullRequest: model.PullRequestOpen, CI: model.CISuccess, Issue: model.IssueOpen, Freshness: model.FreshnessCurrent},
		UpdatedAt: now,
	}
	return project, []model.Lane{local, remote}
}
