package workset

import (
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestLaneOrderPersistsAcrossGroupsAndNewLanesLeadTheirGroup(t *testing.T) {
	manager, now := testManager(t)
	first := testLane("first", "owner/repo", "first", model.WorktreeClean, model.PublicationPublished, now)
	first.PullRequest = &model.PullRequest{Number: 1, UpdatedAt: now}
	second := testLane("second", "owner/repo", "second", model.WorktreeClean, model.PublicationPublished, now)
	second.PullRequest = &model.PullRequest{Number: 2, UpdatedAt: now}
	waiting := testLane("waiting", "owner/repo", "waiting", model.WorktreeClean, model.PublicationPublished, now)
	waiting.PullRequest = &model.PullRequest{Number: 3, UpdatedAt: now}
	waiting.Signals.CI = model.CIPending
	waiting.NextAction = model.ActionWaitForCI

	updated, err := manager.Reconcile(testSnapshot(now, first, second, waiting))
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.Reorder(updated, []string{"second", "waiting", "first"})
	if err != nil {
		t.Fatal(err)
	}
	assertOrder(t, updated.WorkingSet.Order, "second", "waiting", "first")
	assertOrder(t, updated.WorkingSet.Active, "second", "first")
	assertOrder(t, updated.WorkingSet.Waiting, "waiting")

	updated.Lanes[1].Signals.CI = model.CIPending
	updated.Lanes[1].NextAction = model.ActionWaitForCI
	updated, err = manager.Reconcile(updated)
	if err != nil {
		t.Fatal(err)
	}
	assertOrder(t, updated.WorkingSet.Waiting, "second", "waiting")
	updated, err = manager.SetPinned(updated, "first", true)
	if err != nil {
		t.Fatal(err)
	}
	assertOrder(t, updated.WorkingSet.Order, "second", "waiting", "first")

	newLane := testLane("new", "owner/repo", "new", model.WorktreeDirty, model.PublicationPublished, now)
	updated.Lanes = append(updated.Lanes, newLane)
	updated.Projects[0].LaneIDs = append(updated.Projects[0].LaneIDs, newLane.ID)
	updated, err = manager.Reconcile(updated)
	if err != nil {
		t.Fatal(err)
	}
	assertOrder(t, updated.WorkingSet.Order, "new", "second", "waiting", "first")
	assertOrder(t, updated.WorkingSet.Active, "new", "first")
}

func TestLaneReorderRejectsIncompleteDuplicateAndUnknownInputs(t *testing.T) {
	manager, now := testManager(t)
	snapshot, err := manager.Reconcile(testSnapshot(now,
		testLane("one", "owner/repo", "one", model.WorktreeDirty, model.PublicationPublished, now),
		testLane("two", "owner/repo", "two", model.WorktreeDirty, model.PublicationPublished, now),
	))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		ids     []string
		message string
	}{
		{ids: []string{"one"}, message: "every working-set lane"},
		{ids: []string{"one", "one"}, message: "duplicate"},
		{ids: []string{"one", "missing"}, message: "unknown"},
	} {
		if _, err := manager.Reorder(snapshot, test.ids); err == nil || !strings.Contains(err.Error(), test.message) {
			t.Fatalf("reorder %v error = %v, want %q", test.ids, err, test.message)
		}
	}
	loaded, err := (FileStore{}).Load(snapshot.WorkingSet.Path)
	if err != nil {
		t.Fatal(err)
	}
	assertOrder(t, loaded.Order, "one", "two")
}

func TestOlderLaneStateWithoutOrderGetsDeterministicOrder(t *testing.T) {
	manager, now := testManager(t)
	path, err := ResolvePath()
	if err != nil {
		t.Fatal(err)
	}
	observation := model.LaneObservation{ObservedAt: now}
	if err := (FileStore{}).Write(path, State{
		Version: Version, Migrated: true,
		Entries: []Entry{
			{ID: "zeta", GitHub: "owner/repo", State: model.AttentionActive, Previous: observation, Current: observation},
			{ID: "alpha", GitHub: "owner/repo", State: model.AttentionActive, Previous: observation, Current: observation},
		},
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	updated, err := manager.Reconcile(testSnapshot(now,
		testLane("zeta", "owner/repo", "zeta", model.WorktreeDirty, model.PublicationPublished, now),
		testLane("alpha", "owner/repo", "alpha", model.WorktreeDirty, model.PublicationPublished, now),
	))
	if err != nil {
		t.Fatal(err)
	}
	assertOrder(t, updated.WorkingSet.Order, "alpha", "zeta")
}

func assertOrder(t *testing.T, actual []string, expected ...string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("order = %#v, want %#v", actual, expected)
	}
	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf("order = %#v, want %#v", actual, expected)
		}
	}
}
