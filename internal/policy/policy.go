package policy

import (
	"fmt"
	"regexp"
	"strings"
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

func remotePublication(current model.PublicationState, hasLocal bool) model.PublicationState {
	if !hasLocal {
		return model.PublicationPublished
	}
	return current
}

func worktreeState(worktree model.Worktree) model.WorktreeState {
	if worktree.Prunable {
		return model.WorktreeUnavailable
	}
	if worktree.Conflicted > 0 {
		return model.WorktreeConflicted
	}
	if worktree.Staged+worktree.Unstaged+worktree.Untracked > 0 {
		return model.WorktreeDirty
	}
	return model.WorktreeClean
}

func reviewState(pullRequest model.PullRequest) model.ReviewState {
	// reviewDecision is GitHub's current aggregate decision. Feedback counts
	// represent each reviewer's latest submitted state, so an active request is
	// still actionable when branch protection does not publish a decision.
	if pullRequest.Feedback.ChangesRequested > 0 || strings.EqualFold(pullRequest.ReviewDecision, "CHANGES_REQUESTED") {
		return model.ReviewChangesRequested
	}
	if pullRequest.Feedback.UnresolvedThreads > 0 {
		return model.ReviewFeedbackPending
	}
	switch strings.ToUpper(pullRequest.ReviewDecision) {
	case "":
		if pullRequest.Feedback.Approvals > 0 {
			return model.ReviewApproved
		}
		return model.ReviewNone
	case "APPROVED":
		return model.ReviewApproved
	case "REVIEW_REQUIRED":
		return model.ReviewRequired
	default:
		return model.ReviewUnknown
	}
}

func mergeState(state, mergeable string) model.MergeState {
	if strings.EqualFold(mergeable, "CONFLICTING") || strings.EqualFold(state, "DIRTY") {
		return model.MergeConflicting
	}
	switch strings.ToUpper(state) {
	case "CLEAN", "HAS_HOOKS", "UNSTABLE":
		return model.MergeClean
	case "BLOCKED", "BEHIND", "DRAFT":
		return model.MergeBlocked
	default:
		return model.MergeUnknown
	}
}

func reviewReady(lane model.Lane) bool {
	if lane.PullRequest == nil || lane.Signals.PullRequest != model.PullRequestOpen {
		return false
	}
	if lane.Signals.Merge != model.MergeClean && lane.Signals.Merge != model.MergeBlocked {
		return false
	}
	if lane.Signals.CI == model.CIFailure || lane.Signals.CI == model.CIUnknown {
		return false
	}
	if lane.Signals.Review == model.ReviewChangesRequested || lane.Signals.Review == model.ReviewFeedbackPending || lane.Signals.Review == model.ReviewUnknown {
		return false
	}
	if lane.Worktree != nil {
		if lane.Signals.Worktree != model.WorktreeClean {
			return false
		}
		switch lane.Signals.Publication {
		case model.PublicationUnpushed, model.PublicationNoUpstream, model.PublicationDiverged, model.PublicationUnknown:
			return false
		}
	}
	return true
}

func nextAction(lane model.Lane) model.Action {
	if lane.Signals.Worktree == model.WorktreeConflicted || lane.Signals.Merge == model.MergeConflicting {
		return model.ActionResolveConflict
	}
	if lane.Signals.CI == model.CIFailure {
		return model.ActionFixCI
	}
	if lane.Signals.Review == model.ReviewChangesRequested || lane.Signals.Review == model.ReviewFeedbackPending {
		return model.ActionAddressReview
	}
	if lane.Signals.Worktree == model.WorktreeDirty || lane.Signals.Worktree == model.WorktreeUnavailable {
		return model.ActionInspectLocal
	}
	if lane.Signals.Publication == model.PublicationUnpushed || lane.Signals.Publication == model.PublicationNoUpstream {
		return model.ActionPushBranch
	}
	if lane.Signals.Publication == model.PublicationDiverged || (lane.Worktree != nil && lane.Signals.Publication == model.PublicationUnknown) {
		return model.ActionRefreshState
	}
	if lane.PullRequest == nil && lane.Worktree != nil && lane.Branch != lane.Base && lane.Worktree.AheadBase > 0 {
		if lane.Signals.Publication == model.PublicationBehind {
			return model.ActionRefreshState
		}
		return model.ActionCreatePR
	}
	if lane.Signals.PullRequest == model.PullRequestDraft {
		return model.ActionMarkReady
	}
	if lane.PullRequest != nil && lane.Signals.CI == model.CIPending {
		return model.ActionWaitForCI
	}
	if lane.PullRequest != nil && (lane.Signals.CI == model.CIUnknown || lane.Signals.Merge == model.MergeUnknown || lane.Signals.Review == model.ReviewUnknown) {
		return model.ActionRefreshState
	}
	if lane.ReviewReady && lane.Signals.CI == model.CISuccess && lane.Signals.Review == model.ReviewApproved && lane.Signals.Merge == model.MergeClean {
		return model.ActionMergePR
	}
	if lane.ReviewReady && lane.Signals.CI == model.CISuccess && lane.Signals.Review == model.ReviewNone && lane.Signals.Merge == model.MergeClean {
		return model.ActionManualTestMerge
	}
	if lane.ReviewReady {
		return model.ActionReviewPR
	}
	if lane.Signals.Freshness == model.FreshnessStale && (lane.PullRequest != nil || lane.Issue != nil || (lane.Worktree != nil && lane.Branch != lane.Base)) {
		return model.ActionResumeOrClose
	}
	if lane.Issue != nil && lane.PullRequest == nil && lane.Worktree == nil {
		return model.ActionStartIssue
	}
	return model.ActionNone
}

func applyEvidence(lane *model.Lane) {
	if lane.Worktree != nil {
		counts := lane.Worktree
		if lane.Signals.Worktree == model.WorktreeClean {
			lane.Reasons = append(lane.Reasons, "local worktree is clean")
		} else if lane.Signals.Worktree == model.WorktreeDirty {
			lane.Blockers = append(lane.Blockers, fmt.Sprintf("local changes: %d staged, %d unstaged, %d untracked", counts.Staged, counts.Unstaged, counts.Untracked))
		} else if lane.Signals.Worktree == model.WorktreeConflicted {
			lane.Blockers = append(lane.Blockers, fmt.Sprintf("local worktree has %d conflicted paths", counts.Conflicted))
		}
		if counts.Locked {
			lane.Warnings = append(lane.Warnings, "worktree is locked")
		}
	}
	if lane.Issue != nil {
		lane.Reasons = append(lane.Reasons, fmt.Sprintf("linked open issue #%d", lane.Issue.Number))
	}
	switch lane.Signals.Publication {
	case model.PublicationUnpushed:
		lane.Blockers = append(lane.Blockers, "local commits have not been pushed")
	case model.PublicationNoUpstream:
		lane.Blockers = append(lane.Blockers, "branch has no upstream")
	case model.PublicationDiverged:
		lane.Blockers = append(lane.Blockers, "local and upstream branches have diverged")
	case model.PublicationBehind:
		lane.Warnings = append(lane.Warnings, "local branch is behind its upstream")
	case model.PublicationPublished:
		lane.Reasons = append(lane.Reasons, "local branch is fully published")
	}
	switch lane.Signals.CI {
	case model.CIFailure:
		lane.Blockers = append(lane.Blockers, "continuous integration is failing")
	case model.CIPending:
		lane.Warnings = append(lane.Warnings, "continuous integration is pending")
	case model.CINone:
		if lane.PullRequest != nil {
			lane.Warnings = append(lane.Warnings, "pull request has no reported checks")
		}
	case model.CIUnknown:
		lane.Blockers = append(lane.Blockers, "continuous integration state is unknown")
	case model.CISuccess:
		lane.Reasons = append(lane.Reasons, "continuous integration is passing")
	}
	if lane.Signals.Review == model.ReviewChangesRequested {
		lane.Blockers = append(lane.Blockers, "review changes are requested")
	}
	if lane.Signals.Review == model.ReviewFeedbackPending && lane.PullRequest != nil {
		lane.Blockers = append(lane.Blockers, fmt.Sprintf("%d unresolved review thread(s)", lane.PullRequest.Feedback.UnresolvedThreads))
	}
	if lane.Signals.Merge == model.MergeConflicting {
		lane.Blockers = append(lane.Blockers, "pull request has merge conflicts")
	}
}
