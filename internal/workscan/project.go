package workscan

import (
	"sort"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

func inProgress(lane model.Lane) bool {
	if lane.PullRequest != nil {
		return true
	}
	if lane.Worktree == nil {
		return false
	}
	worktree := lane.Worktree
	if worktree.Prunable {
		return false
	}
	if worktree.Detached || worktree.Conflicted > 0 ||
		worktree.Staged+worktree.Unstaged+worktree.Untracked > 0 ||
		worktree.Ahead > 0 || worktree.AheadBase > 0 {
		return true
	}
	if lane.Branch != "" && lane.Branch != lane.Base {
		return true
	}
	switch lane.Signals.Publication {
	case model.PublicationNoUpstream, model.PublicationUnpushed,
		model.PublicationDiverged, model.PublicationUnknown:
		return lane.Branch != lane.Base
	default:
		return false
	}
}

func workItem(repository config.Repository, lane model.Lane) model.WorkItem {
	item := model.WorkItem{
		Repository: repository.Name, GitHub: repository.GitHub,
		RepositoryPath: repository.Path, Branch: lane.Branch, Base: lane.Base,
		State: workState(lane), Publication: lane.Signals.Publication,
		NextAction: lane.NextAction, UpdatedAt: lane.UpdatedAt,
	}
	if lane.Worktree != nil {
		worktree := lane.Worktree
		item.Worktree = &model.WorktreeSummary{
			Path: worktree.Path, Staged: worktree.Staged, Unstaged: worktree.Unstaged,
			Untracked: worktree.Untracked, Conflicted: worktree.Conflicted,
			Ahead: worktree.Ahead, Behind: worktree.Behind,
			AheadBase: worktree.AheadBase, BehindBase: worktree.BehindBase,
			Detached: worktree.Detached,
		}
	}
	if lane.PullRequest != nil {
		pullRequest := lane.PullRequest
		item.PullRequest = &model.WorkPullRequestSummary{
			Number: pullRequest.Number, Title: pullRequest.Title, URL: pullRequest.URL,
			Draft: pullRequest.IsDraft, CI: lane.Signals.CI, Review: lane.Signals.Review,
		}
	}
	return item
}

func workState(lane model.Lane) model.WorkState {
	switch {
	case lane.Signals.Worktree == model.WorktreeConflicted || lane.Signals.Merge == model.MergeConflicting:
		return model.WorkConflict
	case lane.Signals.CI == model.CIFailure:
		return model.WorkCIFailed
	case lane.Signals.Review == model.ReviewChangesRequested || lane.Signals.Review == model.ReviewFeedbackPending:
		return model.WorkFeedback
	case lane.Signals.Worktree == model.WorktreeDirty:
		return model.WorkDirty
	case lane.Signals.Publication == model.PublicationNoUpstream ||
		lane.Signals.Publication == model.PublicationUnpushed ||
		lane.Signals.Publication == model.PublicationDiverged:
		return model.WorkUnpublished
	case lane.Signals.PullRequest == model.PullRequestDraft:
		return model.WorkDraft
	case lane.PullRequest != nil:
		return model.WorkPullRequest
	default:
		return model.WorkBranch
	}
}

func sortWorkItems(items []model.WorkItem) {
	sort.SliceStable(items, func(left, right int) bool {
		leftPriority, rightPriority := workPriority(items[left].State), workPriority(items[right].State)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if items[left].Repository != items[right].Repository {
			return items[left].Repository < items[right].Repository
		}
		if items[left].Branch != items[right].Branch {
			return items[left].Branch < items[right].Branch
		}
		if items[left].PullRequest != nil && items[right].PullRequest != nil {
			return items[left].PullRequest.Number < items[right].PullRequest.Number
		}
		return items[left].RepositoryPath < items[right].RepositoryPath
	})
}

func workPriority(state model.WorkState) int {
	switch state {
	case model.WorkConflict:
		return 1
	case model.WorkCIFailed:
		return 2
	case model.WorkFeedback:
		return 3
	case model.WorkDirty:
		return 4
	case model.WorkUnpublished:
		return 5
	case model.WorkDraft:
		return 6
	case model.WorkPullRequest:
		return 7
	case model.WorkBranch:
		return 8
	default:
		return 9
	}
}
