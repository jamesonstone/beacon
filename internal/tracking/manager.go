package tracking

import (
	"fmt"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

type Manager struct {
	Store        Store
	Now          func() time.Time
	RecentWindow time.Duration
}

type ProbeUpdate struct {
	GitHub   string
	Format   string
	Baseline string
	Local    string
	Remote   string
	At       time.Time
}

var managerMutex sync.Mutex

func (m Manager) Reconcile(snapshot model.Snapshot) (model.Snapshot, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	return m.reconcile(snapshot, true)
}

func (m Manager) ReconcilePartial(snapshot model.Snapshot) (model.Snapshot, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	return m.reconcile(snapshot, false)
}

func (m Manager) ReconcileAt(snapshot model.Snapshot, path string) (model.Snapshot, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	return m.reconcileAt(snapshot, path, true)
}

func (m Manager) ApplyAt(snapshot model.Snapshot, path string) (model.Snapshot, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	state, err := m.loadState(snapshot.ConfigPath, path)
	if err != nil {
		return model.Snapshot{}, err
	}
	changed, err := m.reconcileState(snapshot, &state, false)
	if err != nil {
		return model.Snapshot{}, err
	}
	if changed {
		if err := m.store().Write(path, state); err != nil {
			return model.Snapshot{}, fmt.Errorf("persist project following state: %w", err)
		}
	}
	apply(&snapshot, path, entryMap(state.Untracked), m.now(), m.recentWindow())
	return snapshot, nil
}

func (m Manager) reconcile(snapshot model.Snapshot, finalizeMigration bool) (model.Snapshot, error) {
	path, err := ResolvePath(snapshot.ConfigPath)
	if err != nil {
		return model.Snapshot{}, err
	}
	return m.reconcileAt(snapshot, path, finalizeMigration)
}

func (m Manager) reconcileAt(snapshot model.Snapshot, path string, finalizeMigration bool) (model.Snapshot, error) {
	store := m.store()
	state, err := m.loadState(snapshot.ConfigPath, path)
	if err != nil {
		return model.Snapshot{}, err
	}
	changed, err := m.reconcileState(snapshot, &state, finalizeMigration)
	if err != nil {
		return model.Snapshot{}, err
	}
	if changed {
		if err := store.Write(path, state); err != nil {
			return model.Snapshot{}, fmt.Errorf("persist project following state: %w", err)
		}
	}
	apply(&snapshot, path, entryMap(state.Untracked), m.now(), m.recentWindow())
	return snapshot, nil
}

func (m Manager) RecordActivity(configPath, github, reason string, at time.Time) error {
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
		if state.Untracked[index].GitHub == github {
			state.Untracked[index].LastActivityAt = at.UTC()
			state.Untracked[index].ActivityReason = reason
			return m.store().Write(path, state)
		}
	}
	return fmt.Errorf("project is not outside Following: %s", github)
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

func (m Manager) recentWindow() time.Duration {
	if m.RecentWindow > 0 {
		return m.RecentWindow
	}
	return 24 * time.Hour
}
