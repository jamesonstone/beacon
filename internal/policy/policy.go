package policy

import (
	"fmt"
	"regexp"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
)

var issueBranch = regexp.MustCompile(`(?i)^GH-(\d+)$`)

func Build(repo config.Repository, locals []gitscan.LocalLane, pullRequests []model.PullRequest, issues []model.Issue, progressByIssue map[int]model.Progress, staleAfter time.Duration, now time.Time) []model.Lane {
	usedLocals := make([]bool, len(locals))
	usedIssues := make(map[int]bool, len(issues))
	issuesByNumber := make(map[int]model.Issue, len(issues))
	for _, issue := range issues {
		issuesByNumber[issue.Number] = issue
	}
	lanes := make([]model.Lane, 0, len(locals)+len(pullRequests)+len(issues))

	for index := range pullRequests {
		pullRequest := pullRequests[index]
		localIndex := matchingLocal(locals, usedLocals, pullRequest)
		var local *gitscan.LocalLane
		if localIndex >= 0 {
			usedLocals[localIndex] = true
			local = &locals[localIndex]
		}
		issue := matchingIssue(pullRequest, issuesByNumber)
		if issue != nil {
			usedIssues[issue.Number] = true
		}
		lanes = append(lanes, buildLane(repo, local, &pullRequest, issue, progressForLane(issue, pullRequest.HeadRefName, progressByIssue), staleAfter, now))
	}

	for index := range locals {
		if usedLocals[index] {
			continue
		}
		local := locals[index]
		issue := issueForBranch(local.Branch, issuesByNumber)
		if issue != nil {
			usedIssues[issue.Number] = true
		}
		lanes = append(lanes, buildLane(repo, &local, nil, issue, progressForLane(issue, local.Branch, progressByIssue), staleAfter, now))
	}

	for index := range issues {
		issue := issues[index]
		if usedIssues[issue.Number] {
			continue
		}
		lanes = append(lanes, buildLane(repo, nil, nil, &issue, progressFor(&issue, progressByIssue), staleAfter, now))
	}
	return lanes
}

func matchingLocal(locals []gitscan.LocalLane, used []bool, pullRequest model.PullRequest) int {
	fallback := -1
	for index, local := range locals {
		if used[index] || local.Branch != pullRequest.HeadRefName {
			continue
		}
		if local.Worktree.HeadOID == pullRequest.HeadRefOID {
			return index
		}
		if fallback == -1 {
			fallback = index
		}
	}
	return fallback
}

func matchingIssue(pullRequest model.PullRequest, issues map[int]model.Issue) *model.Issue {
	for _, closing := range pullRequest.ClosingIssues {
		if issue, ok := issues[closing.Number]; ok {
			copy := issue
			return &copy
		}
		copy := closing
		return &copy
	}
	return issueForBranch(pullRequest.HeadRefName, issues)
}

func issueForBranch(branch string, issues map[int]model.Issue) *model.Issue {
	match := issueBranch.FindStringSubmatch(branch)
	if len(match) != 2 {
		return nil
	}
	var number int
	if _, err := fmt.Sscanf(match[1], "%d", &number); err != nil {
		return nil
	}
	issue, ok := issues[number]
	if !ok {
		return nil
	}
	return &issue
}

func progressFor(issue *model.Issue, progressByIssue map[int]model.Progress) *model.Progress {
	if issue == nil {
		return nil
	}
	progress, ok := progressByIssue[issue.Number]
	if !ok {
		return nil
	}
	return &progress
}

func progressForLane(issue *model.Issue, branch string, progressByIssue map[int]model.Progress) *model.Progress {
	if progress := progressFor(issue, progressByIssue); progress != nil {
		return progress
	}
	match := issueBranch.FindStringSubmatch(branch)
	if len(match) != 2 {
		return nil
	}
	var number int
	if _, err := fmt.Sscanf(match[1], "%d", &number); err != nil {
		return nil
	}
	progress, ok := progressByIssue[number]
	if !ok {
		return nil
	}
	return &progress
}

func buildLane(repo config.Repository, local *gitscan.LocalLane, pullRequest *model.PullRequest, issue *model.Issue, progress *model.Progress, staleAfter time.Duration, now time.Time) model.Lane {
	lane := model.Lane{
		Repository: repo.Name, GitHub: repo.GitHub, Base: repo.Base,
		Signals: model.Signals{CI: model.CINone, Review: model.ReviewNone, Merge: model.MergeUnknown, Issue: model.IssueNone},
		Reasons: []string{}, Warnings: []string{}, Blockers: []string{}, Progress: progress,
	}
	if local != nil {
		lane.ID = local.ID
		lane.Branch = local.Branch
		worktree := local.Worktree
		lane.Worktree = &worktree
		lane.Signals.Worktree = worktreeState(worktree)
		lane.Signals.Publication = local.Publication
		lane.UpdatedAt = worktree.UpdatedAt
	} else {
		lane.Signals.Worktree = model.WorktreeNotLocal
		lane.Signals.Publication = model.PublicationUnknown
	}
	if pullRequest != nil {
		lane.ID = fmt.Sprintf("gh:%s#%d", repo.GitHub, pullRequest.Number)
		lane.Branch = pullRequest.HeadRefName
		lane.Base = pullRequest.BaseRefName
		copy := *pullRequest
		lane.PullRequest = &copy
		lane.Signals.PullRequest = model.PullRequestOpen
		if pullRequest.IsDraft {
			lane.Signals.PullRequest = model.PullRequestDraft
		}
		lane.Signals.CI = pullRequest.CI
		lane.Signals.Review = reviewState(*pullRequest)
		lane.Signals.Merge = mergeState(pullRequest.MergeState, pullRequest.Mergeable)
		lane.Signals.Publication = remotePublication(lane.Signals.Publication, local != nil)
		lane.UpdatedAt = pullRequest.UpdatedAt
	} else {
		lane.Signals.PullRequest = model.PullRequestNone
	}
	if issue != nil {
		copy := *issue
		lane.Issue = &copy
		lane.Signals.Issue = model.IssueOpen
		if lane.ID == "" {
			lane.ID = fmt.Sprintf("gh-issue:%s#%d", repo.GitHub, issue.Number)
		}
		if issue.UpdatedAt.After(lane.UpdatedAt) {
			lane.UpdatedAt = issue.UpdatedAt
		}
	}
	if lane.UpdatedAt.IsZero() {
		lane.UpdatedAt = now
	}
	if now.Sub(lane.UpdatedAt) > staleAfter {
		lane.Signals.Freshness = model.FreshnessStale
		lane.Warnings = append(lane.Warnings, "lane has not been updated within the stale threshold")
	} else {
		lane.Signals.Freshness = model.FreshnessCurrent
	}
	if local != nil && pullRequest != nil && local.Worktree.HeadOID != pullRequest.HeadRefOID {
		lane.Warnings = append(lane.Warnings, "local and pull request head commits differ")
	}
	applyEvidence(&lane)
	lane.ReviewReady = reviewReady(lane)
	lane.NextAction = nextAction(lane)
	if lane.ReviewReady {
		lane.Reasons = append(lane.Reasons, "pull request is ready for human review")
	}
	return lane
}
