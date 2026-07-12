package tracking

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

type Manager struct {
	Store Store
	Now   func() time.Time
}

var managerMutex sync.Mutex

func (m Manager) Reconcile(snapshot model.Snapshot) (model.Snapshot, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	return m.reconcile(snapshot)
}

func (m Manager) ReconcileAt(snapshot model.Snapshot, path string) (model.Snapshot, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	return m.reconcileAt(snapshot, path)
}

func (m Manager) ApplyAt(snapshot model.Snapshot, path string) (model.Snapshot, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	state, err := m.loadState(snapshot.ConfigPath, path)
	if err != nil {
		return model.Snapshot{}, err
	}
	untracked := make(map[string]struct{}, len(state.Untracked))
	for _, entry := range state.Untracked {
		untracked[entry.GitHub] = struct{}{}
	}
	apply(&snapshot, path, untracked, nil)
	return snapshot, nil
}

func (m Manager) reconcile(snapshot model.Snapshot) (model.Snapshot, error) {
	path, err := ResolvePath(snapshot.ConfigPath)
	if err != nil {
		return model.Snapshot{}, err
	}
	return m.reconcileAt(snapshot, path)
}

func (m Manager) reconcileAt(snapshot model.Snapshot, path string) (model.Snapshot, error) {
	store := m.store()
	state, err := m.loadState(snapshot.ConfigPath, path)
	if err != nil {
		return model.Snapshot{}, err
	}
	projects := projectMap(snapshot.Projects)
	remaining := make([]Entry, 0, len(state.Untracked))
	untracked := make(map[string]struct{}, len(state.Untracked))
	autoReactivated := make([]string, 0)
	changed := false
	for _, entry := range state.Untracked {
		if entry.State == StateIgnored {
			remaining = append(remaining, entry)
			untracked[entry.GitHub] = struct{}{}
			continue
		}
		project, found := projects[entry.GitHub]
		if !found || !hasCompleteEvidence(snapshot, project) {
			remaining = append(remaining, entry)
			untracked[entry.GitHub] = struct{}{}
			continue
		}
		fingerprint, err := Fingerprint(project, snapshot.Lanes)
		if err != nil {
			return model.Snapshot{}, fmt.Errorf("fingerprint %s: %w", entry.GitHub, err)
		}
		if entry.Baseline == "" {
			entry.Baseline = fingerprint
			remaining = append(remaining, entry)
			untracked[entry.GitHub] = struct{}{}
			changed = true
			continue
		}
		if fingerprint != entry.Baseline {
			changed = true
			autoReactivated = append(autoReactivated, entry.GitHub)
			continue
		}
		remaining = append(remaining, entry)
		untracked[entry.GitHub] = struct{}{}
	}
	if changed {
		state.Untracked = remaining
		for _, github := range autoReactivated {
			state.Reactivations = append(state.Reactivations, Reactivation{GitHub: github, At: m.now(), Reason: "material project evidence changed"})
		}
		if err := store.Write(path, state); err != nil {
			return model.Snapshot{}, fmt.Errorf("persist automatic project reactivation: %w", err)
		}
	}
	sort.Strings(autoReactivated)
	apply(&snapshot, path, untracked, autoReactivated)
	return snapshot, nil
}

func (m Manager) RecordReactivation(configPath, github, reason string) error {
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
	for index := len(state.Reactivations) - 1; index >= 0; index-- {
		if state.Reactivations[index].GitHub == github {
			state.Reactivations[index].Reason = reason
			return m.store().Write(path, state)
		}
	}
	state.Reactivations = append(state.Reactivations, Reactivation{GitHub: github, At: m.now(), Reason: reason})
	return m.store().Write(path, state)
}

func (m Manager) SetTracked(snapshot model.Snapshot, targets []string, tracked bool) (model.Snapshot, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	if len(targets) == 0 {
		return model.Snapshot{}, errors.New("at least one project is required")
	}
	path, err := ResolvePath(snapshot.ConfigPath)
	if err != nil {
		return model.Snapshot{}, err
	}
	store := m.store()
	state, err := m.loadState(snapshot.ConfigPath, path)
	if err != nil {
		return model.Snapshot{}, err
	}
	projects, err := resolveProjects(snapshot.Projects, targets)
	if err != nil {
		return model.Snapshot{}, err
	}
	entries := entryMap(state.Untracked)
	changed := false
	for _, project := range projects {
		if tracked {
			if _, exists := entries[project.GitHub]; exists {
				delete(entries, project.GitHub)
				changed = true
			}
			continue
		}
		baseline := ""
		if hasCompleteEvidence(snapshot, project) {
			baseline, err = Fingerprint(project, snapshot.Lanes)
			if err != nil {
				return model.Snapshot{}, fmt.Errorf("fingerprint %s: %w", project.GitHub, err)
			}
		}
		entries[project.GitHub] = Entry{
			GitHub: project.GitHub, Name: project.Name, Path: project.Path,
			State: StateMuted, UntrackedAt: m.now(), Baseline: baseline,
		}
		changed = true
	}
	if changed {
		state.Untracked = entriesSlice(entries)
		if err := store.Write(path, state); err != nil {
			return model.Snapshot{}, err
		}
	}
	return m.reconcile(snapshot)
}

func (m Manager) SetSelection(snapshot model.Snapshot, trackedGitHub []string) (model.Snapshot, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	desired := make(map[string]struct{}, len(trackedGitHub))
	for _, github := range trackedGitHub {
		desired[github] = struct{}{}
	}
	path, err := ResolvePath(snapshot.ConfigPath)
	if err != nil {
		return model.Snapshot{}, err
	}
	store := m.store()
	state, err := m.loadState(snapshot.ConfigPath, path)
	if err != nil {
		return model.Snapshot{}, err
	}
	entries := entryMap(state.Untracked)
	changed := false
	for _, project := range snapshot.Projects {
		_, shouldTrack := desired[project.GitHub]
		_, isUntracked := entries[project.GitHub]
		switch {
		case shouldTrack && isUntracked:
			delete(entries, project.GitHub)
			changed = true
		case !shouldTrack:
			baseline := ""
			if hasCompleteEvidence(snapshot, project) {
				baseline, err = Fingerprint(project, snapshot.Lanes)
				if err != nil {
					return model.Snapshot{}, fmt.Errorf("fingerprint %s: %w", project.GitHub, err)
				}
			}
			entries[project.GitHub] = Entry{
				GitHub: project.GitHub, Name: project.Name, Path: project.Path,
				State: StateMuted, UntrackedAt: m.now(), Baseline: baseline,
			}
			changed = true
		}
	}
	if changed {
		state.Untracked = entriesSlice(entries)
		if err := store.Write(path, state); err != nil {
			return model.Snapshot{}, err
		}
	}
	return m.reconcile(snapshot)
}

func (m Manager) Entry(configPath, github string) (Entry, bool, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	path, err := ResolvePath(configPath)
	if err != nil {
		return Entry{}, false, err
	}
	state, err := m.loadState(configPath, path)
	if err != nil {
		return Entry{}, false, err
	}
	for _, entry := range state.Untracked {
		if entry.GitHub == github {
			return entry, true, nil
		}
	}
	return Entry{}, false, nil
}

func (m Manager) UpdateProbe(configPath, github, baseline, local, remote string, at time.Time) error {
	if !fingerprintPattern.MatchString(baseline) {
		return errors.New("probe baseline must be a SHA-256 fingerprint")
	}
	if !fingerprintPattern.MatchString(local) || !fingerprintPattern.MatchString(remote) {
		return errors.New("local and remote probe values must be SHA-256 fingerprints")
	}
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
	for index := range state.Untracked {
		if state.Untracked[index].GitHub != github {
			continue
		}
		state.Untracked[index].ProbeBaseline = baseline
		state.Untracked[index].ProbeLocal = local
		state.Untracked[index].ProbeRemote = remote
		state.Untracked[index].LastProbeAt = at.UTC()
		return m.store().Write(path, state)
	}
	return nil
}

func (m Manager) loadState(configPath, path string) (State, error) {
	store := m.store()
	if _, ok := store.(FileStore); ok {
		if _, err := MigrateLegacy(configPath, path); err != nil {
			return State{}, err
		}
	}
	return store.Load(path)
}

func hasCompleteEvidence(snapshot model.Snapshot, project model.Project) bool {
	if len(project.Errors) > 0 {
		return false
	}
	for _, scanError := range snapshot.Errors {
		if scanError.Repository == "" || scanError.Repository == project.Name || scanError.Repository == project.GitHub {
			return false
		}
	}
	return true
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

func apply(snapshot *model.Snapshot, path string, untracked map[string]struct{}, autoReactivated []string) {
	for index := range snapshot.Projects {
		if _, found := untracked[snapshot.Projects[index].GitHub]; found {
			snapshot.Projects[index].TrackingState = model.TrackingUntracked
		} else {
			snapshot.Projects[index].TrackingState = model.TrackingTracked
		}
	}
	untrackedLanes := make(map[string]struct{})
	snapshot.Groups.Untracked = []string{}
	for _, lane := range snapshot.Lanes {
		if _, found := untracked[lane.GitHub]; found {
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
		if project.TrackingState == model.TrackingUntracked {
			summary.UntrackedProjects++
		} else {
			summary.TrackedProjects++
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
	snapshot.Tracking = model.Tracking{Path: path, AutoReactivated: append([]string{}, autoReactivated...)}
}

func ApplyCached(snapshot *model.Snapshot, path string) {
	untracked := make(map[string]struct{})
	for _, project := range snapshot.Projects {
		if project.TrackingState == model.TrackingUntracked {
			untracked[project.GitHub] = struct{}{}
		}
	}
	apply(snapshot, path, untracked, nil)
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
