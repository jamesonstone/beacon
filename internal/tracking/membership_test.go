package tracking

import (
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestManagerFinalizesLegacyMembershipBeforeNewProjectsArrive(t *testing.T) {
	store := &memoryStore{state: State{
		Version: Version, Initialized: false,
		Known: []string{}, Untracked: []Entry{}, Reactivations: []Reactivation{},
	}}
	manager := Manager{Store: store, Now: func() time.Time {
		return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	}}
	snapshot := managerSnapshot(t)
	migrated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !store.state.Initialized || migrated.Projects[0].FollowState != model.FollowFollowing {
		t.Fatalf("migration = %#v, state = %#v", migrated.Projects, store.state)
	}

	newProject := snapshot.Projects[0]
	newProject.Name = "new"
	newProject.GitHub = "owner/new"
	newProject.Path = "/new"
	newProject.LaneIDs = []string{}
	snapshot.Projects = append(snapshot.Projects, newProject)
	result, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if result.Projects[1].FollowState != model.FollowQuiet {
		t.Fatalf("new project = %#v", result.Projects[1])
	}
}

func TestManagerPartialReconciliationDoesNotFinalizeLegacyInventory(t *testing.T) {
	store := &memoryStore{state: State{
		Version: Version, Initialized: false,
		Known: []string{}, Untracked: []Entry{}, Reactivations: []Reactivation{},
	}}
	manager := Manager{Store: store}
	if _, err := manager.ReconcilePartial(managerSnapshot(t)); err != nil {
		t.Fatal(err)
	}
	if store.state.Initialized {
		t.Fatalf("partial reconciliation finalized migration: %#v", store.state)
	}
}

func TestManagerInventoryPreservesLegacyChoicesAndQuietsLaterDiscoveries(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	store := &memoryStore{state: State{
		Version: Version, Initialized: false, Known: []string{"owner/muted"},
		Untracked: []Entry{{
			GitHub: "owner/muted", Name: "muted", Path: "/muted",
			State: StateMuted, UntrackedAt: now,
		}},
		Reactivations: []Reactivation{},
	}}
	manager := Manager{Store: store, Now: func() time.Time { return now }}
	legacyProjects := []model.Project{
		{Name: "muted", Path: "/muted", GitHub: "owner/muted"},
		{Name: "followed", Path: "/followed", GitHub: "owner/followed"},
	}
	if err := manager.InitializeInventory("/config.yaml", legacyProjects); err != nil {
		t.Fatal(err)
	}
	if !store.state.Initialized || len(store.state.Untracked) != 1 || store.state.Untracked[0].GitHub != "owner/muted" {
		t.Fatalf("migrated inventory = %#v", store.state)
	}
	if err := manager.InitializeInventory("/config.yaml", append(legacyProjects,
		model.Project{Name: "new", Path: "/new", GitHub: "owner/new"},
	)); err != nil {
		t.Fatal(err)
	}
	entries := entryMap(store.state.Untracked)
	if _, found := entries["owner/followed"]; found {
		t.Fatalf("legacy followed project moved outside Following: %#v", store.state)
	}
	if _, found := entries["owner/new"]; !found {
		t.Fatalf("new project did not begin Quiet: %#v", store.state)
	}
}

type memoryStore struct {
	state State
}

func (s *memoryStore) Load(string) (State, error) { return s.state, nil }
func (s *memoryStore) Write(_ string, state State) error {
	s.state = state
	return nil
}
