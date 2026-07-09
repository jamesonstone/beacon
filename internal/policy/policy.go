package policy

import (
	"fmt"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
)

func Build(repo config.Repository, locals []gitscan.LocalLane, pullRequests []model.PullRequest, staleAfter time.Duration, now time.Time) []model.Lane {
	used := make([]bool, len(locals))
	lanes := make([]model.Lane, 0, len(locals)+len(pullRequests))
	for index := range pullRequests {
		pullRequest := pullRequests[index]
		localIndex := matchingLocal(locals, used, pullRequest)
		var local *gitscan.LocalLane
		if localIndex >= 0 {
			used[localIndex] = true
			local = &locals[localIndex]
		}
		lanes = append(lanes, buildLane(repo, local, &pullRequest, staleAfter, now))
	}
	for index := range locals {
		if !used[index] {
			lanes = append(lanes, buildLane(repo, &locals[index], nil, staleAfter, now))
		}
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

func buildLane(repo config.Repository, local *gitscan.LocalLane, pullRequest *model.PullRequest, staleAfter time.Duration, now time.Time) model.Lane {
	lane := model.Lane{
		Repository: repo.Name, GitHub: repo.GitHub, Base: repo.Base,
		Signals: model.Signals{CI: model.CINone, Review: model.ReviewNone, Merge: model.MergeUnknown},
		Reasons: []string{}, Warnings: []string{}, Blockers: []string{},
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
		lane.Signals.Publication = model.PublicationPublished
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
		lane.Signals.Review = reviewState(pullRequest.ReviewDecision)
		lane.Signals.Merge = mergeState(pullRequest.MergeState, pullRequest.Mergeable)
		lane.UpdatedAt = pullRequest.UpdatedAt
	} else {
		lane.Signals.PullRequest = model.PullRequestNone
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

func reviewState(value string) model.ReviewState {
	switch strings.ToUpper(value) {
	case "":
		return model.ReviewNone
	case "REVIEW_REQUIRED":
		return model.ReviewRequired
	case "CHANGES_REQUESTED":
		return model.ReviewChangesRequested
	case "APPROVED":
		return model.ReviewApproved
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
	if lane.Signals.Review == model.ReviewChangesRequested || lane.Signals.Review == model.ReviewUnknown {
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
	if lane.Signals.Review == model.ReviewChangesRequested {
		return model.ActionAddressReview
	}
	if lane.Signals.Worktree == model.WorktreeDirty || lane.Signals.Worktree == model.WorktreeUnavailable {
		return model.ActionInspectLocal
	}
	if lane.Signals.Publication == model.PublicationUnpushed || lane.Signals.Publication == model.PublicationNoUpstream {
		return model.ActionPushBranch
	}
	if lane.Signals.Publication == model.PublicationDiverged || lane.Signals.Publication == model.PublicationUnknown {
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
	if lane.Signals.CI == model.CIUnknown || lane.Signals.Merge == model.MergeUnknown || lane.Signals.Review == model.ReviewUnknown {
		return model.ActionRefreshState
	}
	if lane.ReviewReady {
		return model.ActionReviewPR
	}
	if lane.Signals.Freshness == model.FreshnessStale && (lane.PullRequest != nil || (lane.Worktree != nil && lane.Branch != lane.Base)) {
		return model.ActionResumeOrClose
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
	if lane.Signals.Merge == model.MergeConflicting {
		lane.Blockers = append(lane.Blockers, "pull request has merge conflicts")
	}
}
