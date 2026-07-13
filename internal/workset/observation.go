package workset

import (
	"fmt"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func observe(lane model.Lane, now time.Time) model.LaneObservation {
	value := model.LaneObservation{
		Worktree:    lane.Signals.Worktree,
		Publication: lane.Signals.Publication,
		CI:          lane.Signals.CI,
		Review:      lane.Signals.Review,
		Merge:       lane.Signals.Merge,
		ObservedAt:  now,
	}
	if lane.Worktree != nil {
		value.HeadOID = lane.Worktree.HeadOID
		value.StatusHash = lane.Worktree.StatusHash
	}
	if lane.PullRequest != nil {
		value.PullRequest = lane.PullRequest.Number
		value.Unresolved = lane.PullRequest.Feedback.UnresolvedThreads
		value.RemoteUpdatedAt = lane.PullRequest.UpdatedAt
	}
	return value
}

func observationChanged(previous, current model.LaneObservation) bool {
	return previous.HeadOID != current.HeadOID ||
		previous.StatusHash != current.StatusHash ||
		previous.Worktree != current.Worktree ||
		previous.Publication != current.Publication ||
		previous.PullRequest != current.PullRequest ||
		previous.CI != current.CI ||
		previous.Review != current.Review ||
		previous.Merge != current.Merge ||
		previous.Unresolved != current.Unresolved
}

func delta(previous, current model.LaneObservation) string {
	switch {
	case previous.PullRequest == 0 && current.PullRequest > 0:
		return fmt.Sprintf("PR #%d opened", current.PullRequest)
	case previous.HeadOID != "" && current.HeadOID != "" && previous.HeadOID != current.HeadOID:
		return "new commit observed"
	case previous.Publication != current.Publication:
		return fmt.Sprintf("publication changed from %s to %s", previous.Publication, current.Publication)
	case previous.CI != current.CI:
		return fmt.Sprintf("CI changed from %s to %s", previous.CI, current.CI)
	case previous.Unresolved != current.Unresolved:
		return fmt.Sprintf("unresolved feedback changed from %d to %d", previous.Unresolved, current.Unresolved)
	case previous.Review != current.Review:
		return fmt.Sprintf("review changed from %s to %s", previous.Review, current.Review)
	case previous.Worktree != current.Worktree || previous.StatusHash != current.StatusHash:
		return fmt.Sprintf("local worktree changed to %s", current.Worktree)
	default:
		return "no material change"
	}
}

func attention(entry Entry) model.LaneAttention {
	return model.LaneAttention{
		State:              entry.State,
		Pinned:             entry.Pinned,
		Manual:             entry.Manual,
		Title:              entry.Title,
		Tags:               append([]string{}, entry.Tags...),
		Note:               entry.Note,
		NoteUpdatedAt:      entry.NoteUpdatedAt,
		NoteStale:          entry.Note != "" && entry.NoteUpdatedAt.Before(entry.Current.ObservedAt) && observationChanged(entry.Previous, entry.Current),
		LastSeenAt:         entry.LastSeenAt,
		Delta:              delta(entry.Previous, entry.Current),
		ReactivationReason: entry.ReactivationReason,
		Previous:           entry.Previous,
		Current:            entry.Current,
	}
}
