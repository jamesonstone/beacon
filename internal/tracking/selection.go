package tracking

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

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
	entries, err := m.Entries(configPath)
	if err != nil {
		return Entry{}, false, err
	}
	entry, found := entries[github]
	return entry, found, nil
}

func (m Manager) Entries(configPath string) (map[string]Entry, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	path, err := ResolvePath(configPath)
	if err != nil {
		return nil, err
	}
	state, err := m.loadState(configPath, path)
	if err != nil {
		return nil, err
	}
	entries := make(map[string]Entry, len(state.Untracked))
	for _, entry := range state.Untracked {
		entries[entry.GitHub] = entry
	}
	return entries, nil
}

func (m Manager) UpdateProbe(configPath, github, format, baseline, local, remote string, at time.Time) error {
	return m.UpdateProbes(configPath, []ProbeUpdate{{
		GitHub: github, Format: format, Baseline: baseline,
		Local: local, Remote: remote, At: at,
	}})
}

func (m Manager) UpdateProbes(configPath string, updates []ProbeUpdate) error {
	for _, update := range updates {
		if strings.TrimSpace(update.Format) == "" {
			return errors.New("probe format is required")
		}
		if !fingerprintPattern.MatchString(update.Baseline) {
			return errors.New("probe baseline must be a SHA-256 fingerprint")
		}
		if !fingerprintPattern.MatchString(update.Local) || !fingerprintPattern.MatchString(update.Remote) {
			return errors.New("local and remote probe values must be SHA-256 fingerprints")
		}
	}
	if len(updates) == 0 {
		return nil
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
	updatesByGitHub := make(map[string]ProbeUpdate, len(updates))
	for _, update := range updates {
		updatesByGitHub[update.GitHub] = update
	}
	changed := false
	for index := range state.Untracked {
		update, found := updatesByGitHub[state.Untracked[index].GitHub]
		if !found {
			continue
		}
		state.Untracked[index].ProbeBaseline = update.Baseline
		state.Untracked[index].ProbeFormat = update.Format
		state.Untracked[index].ProbeLocal = update.Local
		state.Untracked[index].ProbeRemote = update.Remote
		state.Untracked[index].LastProbeAt = update.At.UTC()
		changed = true
	}
	if !changed {
		return nil
	}
	return m.store().Write(path, state)
}
