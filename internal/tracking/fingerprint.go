package tracking

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jamesonstone/beacon/internal/model"
)

type projectEvidence struct {
	GitHub string         `json:"github"`
	Base   string         `json:"base"`
	Lanes  []laneEvidence `json:"lanes"`
}

type laneEvidence struct {
	ID          string             `json:"id"`
	Branch      string             `json:"branch"`
	Worktree    *worktreeEvidence  `json:"worktree,omitempty"`
	PullRequest *model.PullRequest `json:"pull_request,omitempty"`
	Issue       *model.Issue       `json:"issue,omitempty"`
	Signals     signalEvidence     `json:"signals"`
}

type worktreeEvidence struct {
	model.Worktree
	StatusHash string `json:"status_hash"`
}

type signalEvidence struct {
	Worktree    model.WorktreeState    `json:"worktree"`
	Publication model.PublicationState `json:"publication"`
	PullRequest model.PullRequestState `json:"pull_request"`
	CI          model.CIState          `json:"ci"`
	Review      model.ReviewState      `json:"review"`
	Merge       model.MergeState       `json:"merge"`
	Issue       model.IssueState       `json:"issue"`
}

func Fingerprint(project model.Project, lanes []model.Lane) (string, error) {
	evidence := projectEvidence{GitHub: project.GitHub, Base: project.Base, Lanes: []laneEvidence{}}
	for _, lane := range lanes {
		if lane.GitHub != project.GitHub {
			continue
		}
		item := laneEvidence{
			ID: lane.ID, Branch: lane.Branch,
			Signals: signalEvidence{
				Worktree: lane.Signals.Worktree, Publication: lane.Signals.Publication,
				PullRequest: lane.Signals.PullRequest, CI: lane.Signals.CI,
				Review: lane.Signals.Review, Merge: lane.Signals.Merge, Issue: lane.Signals.Issue,
			},
		}
		if lane.Worktree != nil {
			worktree := *lane.Worktree
			item.Worktree = &worktreeEvidence{Worktree: worktree, StatusHash: worktree.StatusHash}
		}
		if lane.PullRequest != nil {
			pullRequest := *lane.PullRequest
			pullRequest.ClosingIssues = cloneIssues(lane.PullRequest.ClosingIssues)
			normalizeIssues(pullRequest.ClosingIssues)
			item.PullRequest = &pullRequest
		}
		if lane.Issue != nil {
			issue := cloneIssue(*lane.Issue)
			normalizeIssue(&issue)
			item.Issue = &issue
		}
		evidence.Lanes = append(evidence.Lanes, item)
	}
	sort.SliceStable(evidence.Lanes, func(i, j int) bool { return evidence.Lanes[i].ID < evidence.Lanes[j].ID })
	encoded, err := json.Marshal(evidence)
	if err != nil {
		return "", fmt.Errorf("encode project evidence: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(encoded)), nil
}

func cloneIssues(issues []model.Issue) []model.Issue {
	cloned := make([]model.Issue, len(issues))
	for index, issue := range issues {
		cloned[index] = cloneIssue(issue)
	}
	return cloned
}

func cloneIssue(issue model.Issue) model.Issue {
	issue.Labels = append([]string{}, issue.Labels...)
	issue.Assignees = append([]string{}, issue.Assignees...)
	return issue
}

func normalizeIssues(issues []model.Issue) {
	for index := range issues {
		normalizeIssue(&issues[index])
	}
	sort.SliceStable(issues, func(i, j int) bool { return issues[i].Number < issues[j].Number })
}

func normalizeIssue(issue *model.Issue) {
	sort.Strings(issue.Labels)
	sort.Strings(issue.Assignees)
}
