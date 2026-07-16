package workset

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

type Manager struct {
	Store        Store
	Now          func() time.Time
	RecentWindow time.Duration
}

var stateMutex sync.Mutex

const staleLaneWarning = "lane has not been updated within the stale threshold"

func (m Manager) Reconcile(snapshot model.Snapshot) (model.Snapshot, error) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	path, err := ResolvePath()
	if err != nil {
		return model.Snapshot{}, err
	}
	state, err := m.store().Load(path)
	if err != nil {
		return model.Snapshot{}, err
	}
	entries := make(map[string]Entry, len(state.Entries))
	for _, entry := range state.Entries {
		entries[entry.ID] = entry
	}
	changed := false
	if !state.Migrated {
		for _, project := range snapshot.Projects {
			if project.TrackingState != model.TrackingUntracked {
				continue
			}
			for _, laneID := range project.LaneIDs {
				if _, found := entries[laneID]; !found {
					entries[laneID] = Entry{ID: laneID, Repository: project.Name, GitHub: project.GitHub, State: model.AttentionParked}
				}
			}
		}
		state.Migrated = true
		changed = true
	}

	working := model.WorkingSet{Path: path, Active: []string{}, Waiting: []string{}, Recent: []string{}, Parked: []string{}}
	following := followingProjects(snapshot.Projects)
	seenLanes := make(map[string]struct{}, len(snapshot.Lanes))
	for index := range snapshot.Lanes {
		lane := &snapshot.Lanes[index]
		lane.Attention = nil
		seenLanes[lane.ID] = struct{}{}
		if !following.includes(lane.GitHub) {
			continue
		}
		entry, found := entries[lane.ID]
		observation := observe(*lane, m.now())
		if found && observation.StatusHash == "" {
			observation.StatusHash = entry.Current.StatusHash
		}
		observationChangedSinceLast := found && !entry.Current.ObservedAt.IsZero() && observationChanged(entry.Current, observation)
		materialObservedAt := observation.ObservedAt
		if found && !entry.Current.ObservedAt.IsZero() && !observationChangedSinceLast {
			materialObservedAt = entry.Current.ObservedAt
		}
		if lane.Signals.Worktree == model.WorktreeDirty || lane.Signals.Worktree == model.WorktreeConflicted {
			applyMaterialActivity(lane, materialObservedAt, m.now().Add(-m.recentWindow()))
		}
		candidateState, candidate := m.candidate(*lane, materialObservedAt)
		if !found && !candidate {
			continue
		}
		if !found {
			entry = Entry{
				ID: lane.ID, Repository: lane.Repository, GitHub: lane.GitHub,
				Branch: lane.Branch, State: candidateState,
				Previous: observation, Current: observation,
			}
			changed = true
		} else {
			if entry.Current.ObservedAt.IsZero() {
				entry.Previous = observation
				entry.Current = observation
				changed = true
			} else if observationChangedSinceLast {
				entry.Current = observation
				if entry.State == model.AttentionParked {
					entry.State = model.AttentionActive
					entry.Explicit = false
					entry.ReactivationReason = delta(entry.Previous, observation)
				}
				changed = true
			}
			if !entry.Pinned && !entry.Explicit && candidate {
				if entry.State != candidateState {
					entry.State = candidateState
					changed = true
				}
			}
		}
		entry.Repository, entry.GitHub, entry.Branch = lane.Repository, lane.GitHub, lane.Branch
		entries[lane.ID] = entry
		if !candidate && !entry.Pinned && !entry.Manual && !entry.Explicit && entry.State != model.AttentionParked {
			if len(entry.Tags) == 0 && entry.Note == "" {
				delete(entries, lane.ID)
				changed = true
			}
			continue
		}
		attention := attention(entry)
		lane.Attention = &attention
		appendGroup(&working, lane.ID, entry.State)
	}

	for _, entry := range entries {
		if _, found := seenLanes[entry.ID]; found {
			continue
		}
		if !entry.Manual && !following.includes(entry.GitHub) {
			continue
		}
		if !entry.Manual && !entry.Pinned && !entry.Explicit && entry.State != model.AttentionParked {
			continue
		}
		lane := retainedLane(entry)
		value := attention(entry)
		lane.Attention = &value
		snapshot.Lanes = append(snapshot.Lanes, lane)
		appendGroup(&working, entry.ID, entry.State)
	}
	sortWorking(&working)
	snapshot.WorkingSet = working
	snapshot.Summary.ActiveLanes = len(working.Active) + len(working.Waiting)
	snapshot.Summary.RecentLanes = len(working.Recent)
	snapshot.Summary.ParkedLanes = len(working.Parked)
	if changed {
		state.Entries = entrySlice(entries)
		state.UpdatedAt = m.now()
		if err := m.store().Write(path, state); err != nil {
			return model.Snapshot{}, err
		}
	}
	return snapshot, nil
}

func applyMaterialActivity(lane *model.Lane, observedAt, cutoff time.Time) {
	lane.UpdatedAt = observedAt
	if observedAt.Before(cutoff) {
		lane.Signals.Freshness = model.FreshnessStale
		for _, warning := range lane.Warnings {
			if warning == staleLaneWarning {
				return
			}
		}
		lane.Warnings = append(lane.Warnings, staleLaneWarning)
		return
	}

	lane.Signals.Freshness = model.FreshnessCurrent
	warnings := lane.Warnings[:0]
	for _, warning := range lane.Warnings {
		if warning != staleLaneWarning {
			warnings = append(warnings, warning)
		}
	}
	lane.Warnings = warnings
}

func retainedLane(entry Entry) model.Lane {
	repository, branch := entry.Repository, entry.Branch
	action := model.ActionRefreshState
	warnings := []string{}
	if entry.Manual {
		repository, branch = "manual", "manual"
		action = model.ActionContinueWork
	} else if entry.State == model.AttentionParked {
		action = model.ActionResumeOrClose
	}
	worktreeState := entry.Current.Worktree
	if worktreeState == "" {
		worktreeState = model.WorktreeNotLocal
	}
	publication := entry.Current.Publication
	if publication == "" {
		publication = model.PublicationUnknown
	}
	ci := entry.Current.CI
	if ci == "" {
		ci = model.CINone
	}
	review := entry.Current.Review
	if review == "" {
		review = model.ReviewNone
	}
	merge := entry.Current.Merge
	if merge == "" {
		merge = model.MergeUnknown
	}
	pullRequestState := model.PullRequestNone
	var pullRequest *model.PullRequest
	if entry.Current.PullRequest > 0 && entry.GitHub != "" {
		pullRequestState = model.PullRequestOpen
		pullRequest = &model.PullRequest{
			Number: entry.Current.PullRequest, Title: fmt.Sprintf("PR #%d", entry.Current.PullRequest),
			URL:         fmt.Sprintf("https://github.com/%s/pull/%d", entry.GitHub, entry.Current.PullRequest),
			HeadRefName: branch, HeadRefOID: entry.Current.HeadOID, UpdatedAt: entry.Current.RemoteUpdatedAt,
			CI: ci, Checks: model.CheckSummary{}, Feedback: model.Feedback{UnresolvedThreads: entry.Current.Unresolved}, ClosingIssues: []model.Issue{},
		}
		warnings = append(warnings, "remote evidence was not enriched in the current snapshot")
	}
	updatedAt := entry.Current.RemoteUpdatedAt
	if updatedAt.IsZero() {
		updatedAt = entry.Current.ObservedAt
	}
	return model.Lane{
		ID: entry.ID, Repository: repository, GitHub: entry.GitHub, Branch: branch,
		PullRequest: pullRequest,
		Signals: model.Signals{
			Worktree: worktreeState, Publication: publication, PullRequest: pullRequestState,
			CI: ci, Review: review, Merge: merge, Freshness: model.FreshnessCurrent, Issue: model.IssueNone,
		},
		NextAction: action, Reasons: []string{"retained from local working-set state"}, Warnings: warnings, Blockers: []string{}, UpdatedAt: updatedAt,
	}
}

func (m Manager) candidate(lane model.Lane, materialObservedAt time.Time) (model.AttentionState, bool) {
	cutoff := m.now().Add(-m.recentWindow())
	if lane.Signals.CI == model.CIPending || lane.NextAction == model.ActionWaitForCI {
		return model.AttentionWaiting, true
	}
	if lane.PullRequest != nil || lane.Issue != nil {
		return model.AttentionActive, true
	}
	if lane.Signals.Worktree == model.WorktreeDirty || lane.Signals.Worktree == model.WorktreeConflicted {
		if materialObservedAt.Before(cutoff) {
			return model.AttentionParked, true
		}
		return model.AttentionActive, true
	}
	if lane.Signals.Publication == model.PublicationUnpushed || lane.Signals.Publication == model.PublicationNoUpstream || lane.Signals.Publication == model.PublicationDiverged {
		return model.AttentionActive, true
	}
	if lane.Worktree != nil && lane.Branch != lane.Base && lane.Worktree.UpdatedAt.After(cutoff) {
		return model.AttentionRecent, true
	}
	return model.AttentionRecent, false
}

func appendGroup(groups *model.WorkingSet, id string, state model.AttentionState) {
	switch state {
	case model.AttentionActive:
		groups.Active = append(groups.Active, id)
	case model.AttentionWaiting:
		groups.Waiting = append(groups.Waiting, id)
	case model.AttentionRecent:
		groups.Recent = append(groups.Recent, id)
	case model.AttentionParked:
		groups.Parked = append(groups.Parked, id)
	}
}

func sortWorking(groups *model.WorkingSet) {
	sort.Strings(groups.Active)
	sort.Strings(groups.Waiting)
	sort.Strings(groups.Recent)
	sort.Strings(groups.Parked)
}

func entrySlice(values map[string]Entry) []Entry {
	result := make([]Entry, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (m Manager) store() Store {
	if m.Store != nil {
		return m.Store
	}
	return FileStore{}
}

func (m Manager) now() time.Time {
	if m.Now != nil {
		return m.Now().UTC()
	}
	return time.Now().UTC()
}

func (m Manager) recentWindow() time.Duration {
	if m.RecentWindow > 0 {
		return m.RecentWindow
	}
	return 6 * time.Hour
}
