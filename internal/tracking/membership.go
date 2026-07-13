package tracking

import (
	"fmt"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func (m Manager) reconcileState(snapshot model.Snapshot, state *State, finalizeMigration bool) (bool, error) {
	known := make(map[string]struct{}, len(state.Known)+len(snapshot.Projects))
	for _, github := range state.Known {
		known[github] = struct{}{}
	}
	entries := entryMap(state.Untracked)
	changed := false
	for _, project := range snapshot.Projects {
		if _, found := known[project.GitHub]; !found {
			known[project.GitHub] = struct{}{}
			state.Known = append(state.Known, project.GitHub)
			changed = true
			if state.Initialized {
				entry, err := m.newUntrackedEntry(snapshot, project)
				if err != nil {
					return false, err
				}
				entries[project.GitHub] = entry
			}
		}
		entry, found := entries[project.GitHub]
		if !found {
			continue
		}
		if entry.Name != project.Name || entry.Path != project.Path {
			entry.Name = project.Name
			entry.Path = project.Path
			changed = true
		}
		if !hasCompleteEvidence(snapshot, project) {
			entries[project.GitHub] = entry
			continue
		}
		fingerprint, err := Fingerprint(project, snapshot.Lanes)
		if err != nil {
			return false, fmt.Errorf("fingerprint %s: %w", entry.GitHub, err)
		}
		switch {
		case entry.Baseline == "":
			entry.Baseline = fingerprint
			changed = true
		case entry.Baseline != fingerprint:
			entry.Baseline = fingerprint
			entry.LastActivityAt = m.now()
			entry.ActivityReason = "material project evidence changed"
			entry.ReactivationReason = ""
			changed = true
		}
		entries[project.GitHub] = entry
	}
	if changed {
		state.Untracked = entriesSlice(entries)
	}
	if finalizeMigration && !state.Initialized && followingMigrationComplete(snapshot) {
		state.Initialized = true
		changed = true
	}
	return changed, nil
}

func (m Manager) InitializeInventory(configPath string, projects []model.Project) error {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	path, err := ResolvePath(configPath)
	if err != nil {
		return err
	}
	state, err := m.loadState(configPath, path)
	if err != nil {
		return err
	}
	known := make(map[string]struct{}, len(state.Known)+len(projects))
	for _, github := range state.Known {
		known[github] = struct{}{}
	}
	entries := entryMap(state.Untracked)
	changed := false
	for _, project := range projects {
		if _, found := known[project.GitHub]; found {
			continue
		}
		known[project.GitHub] = struct{}{}
		state.Known = append(state.Known, project.GitHub)
		changed = true
		if state.Initialized {
			entries[project.GitHub] = Entry{
				GitHub: project.GitHub, Name: project.Name, Path: project.Path,
				State: StateMuted, UntrackedAt: m.now(),
			}
		}
	}
	if !state.Initialized {
		state.Initialized = true
		changed = true
	}
	if !changed {
		return nil
	}
	state.Untracked = entriesSlice(entries)
	if err := m.store().Write(path, state); err != nil {
		return fmt.Errorf("persist project following inventory: %w", err)
	}
	return nil
}

func followingMigrationComplete(snapshot model.Snapshot) bool {
	if len(snapshot.Projects) == 0 || len(snapshot.Errors) > 0 {
		return false
	}
	for _, project := range snapshot.Projects {
		if !hasCompleteEvidence(snapshot, project) {
			return false
		}
	}
	return true
}

func (m Manager) newUntrackedEntry(snapshot model.Snapshot, project model.Project) (Entry, error) {
	entry := Entry{
		GitHub: project.GitHub, Name: project.Name, Path: project.Path,
		State: StateMuted, UntrackedAt: m.now(),
	}
	if !hasCompleteEvidence(snapshot, project) {
		return entry, nil
	}
	fingerprint, err := Fingerprint(project, snapshot.Lanes)
	if err != nil {
		return Entry{}, fmt.Errorf("fingerprint %s: %w", project.GitHub, err)
	}
	entry.Baseline = fingerprint
	return entry, nil
}

func apply(snapshot *model.Snapshot, path string, entries map[string]Entry, now time.Time, recentWindow time.Duration) {
	for index := range snapshot.Projects {
		project := &snapshot.Projects[index]
		entry, outside := entries[project.GitHub]
		if !outside {
			project.TrackingState = model.TrackingTracked
			project.FollowState = model.FollowFollowing
			project.LastActivityAt = time.Time{}
			project.ActivityReason = ""
			continue
		}
		project.TrackingState = model.TrackingUntracked
		project.FollowState = model.FollowQuiet
		project.LastActivityAt = entry.LastActivityAt
		project.ActivityReason = entry.ActivityReason
		if !entry.LastActivityAt.IsZero() && now.Sub(entry.LastActivityAt) <= recentWindow {
			project.FollowState = model.FollowRecent
		}
	}

	untrackedLanes := make(map[string]struct{})
	snapshot.Groups.Untracked = []string{}
	for _, lane := range snapshot.Lanes {
		if _, found := entries[lane.GitHub]; found {
			untrackedLanes[lane.ID] = struct{}{}
			snapshot.Groups.Untracked = append(snapshot.Groups.Untracked, lane.ID)
		}
	}
	snapshot.Groups.Ready = filterTracked(snapshot.Groups.Ready, untrackedLanes)
	snapshot.Groups.Action = filterTracked(snapshot.Groups.Action, untrackedLanes)
	snapshot.Groups.Waiting = filterTracked(snapshot.Groups.Waiting, untrackedLanes)
	snapshot.Groups.Idle = filterTracked(snapshot.Groups.Idle, untrackedLanes)

	summary := model.Summary{Errors: len(snapshot.Errors), Warnings: len(snapshot.Warnings)}
	summary.Projects = len(snapshot.Projects)
	for _, project := range snapshot.Projects {
		switch project.FollowState {
		case model.FollowRecent:
			summary.UntrackedProjects++
			summary.RecentProjects++
		case model.FollowQuiet:
			summary.UntrackedProjects++
			summary.QuietProjects++
		default:
			summary.TrackedProjects++
			summary.FollowingProjects++
		}
	}
	summary.ReviewReady = len(snapshot.Groups.Ready)
	summary.NeedsAction = len(snapshot.Groups.Action)
	summary.Waiting = len(snapshot.Groups.Waiting)
	summary.Idle = len(snapshot.Groups.Idle)
	summary.Total = summary.ReviewReady + summary.NeedsAction + summary.Waiting + summary.Idle
	for _, lane := range snapshot.Lanes {
		if _, found := untrackedLanes[lane.ID]; found {
			continue
		}
		if lane.Issue != nil {
			summary.OpenIssues++
		}
		if lane.PullRequest != nil {
			summary.UnresolvedFeedback += lane.PullRequest.Feedback.UnresolvedThreads
		}
	}
	snapshot.Summary = summary
	snapshot.Tracking = model.Tracking{Path: path, AutoReactivated: []string{}}
}

func ApplyCached(snapshot *model.Snapshot, path string, now time.Time, recentWindow time.Duration) {
	entries := make(map[string]Entry)
	for _, project := range snapshot.Projects {
		if project.TrackingState == model.TrackingUntracked {
			entries[project.GitHub] = Entry{
				GitHub: project.GitHub, LastActivityAt: project.LastActivityAt,
				ActivityReason: project.ActivityReason,
			}
		}
	}
	if recentWindow <= 0 {
		recentWindow = 24 * time.Hour
	}
	apply(snapshot, path, entries, now.UTC(), recentWindow)
}

func filterTracked(ids []string, untracked map[string]struct{}) []string {
	filtered := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, found := untracked[id]; !found {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

func resolveProjects(projects []model.Project, targets []string) ([]model.Project, error) {
	resolved := make([]model.Project, 0, len(targets))
	seen := make(map[string]struct{})
	for _, target := range targets {
		var matches []model.Project
		for _, project := range projects {
			if project.GitHub == target || project.Name == target {
				matches = append(matches, project)
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("project not found: %s", target)
		}
		if len(matches) > 1 {
			return nil, fmt.Errorf("project name is ambiguous; use owner/repo: %s", target)
		}
		if _, exists := seen[matches[0].GitHub]; !exists {
			resolved = append(resolved, matches[0])
			seen[matches[0].GitHub] = struct{}{}
		}
	}
	return resolved, nil
}

func projectMap(projects []model.Project) map[string]model.Project {
	result := make(map[string]model.Project, len(projects))
	for _, project := range projects {
		result[project.GitHub] = project
	}
	return result
}

func entryMap(entries []Entry) map[string]Entry {
	result := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		result[entry.GitHub] = entry
	}
	return result
}

func entriesSlice(entries map[string]Entry) []Entry {
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}
	sortEntries(result)
	return result
}
