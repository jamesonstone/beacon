package policy

import (
	"fmt"
	"strings"

	"github.com/jamesonstone/beacon/internal/model"
)

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
