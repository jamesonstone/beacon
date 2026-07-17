package notes

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestWorkspacePinsPersistBeforeStableUnpinnedOrder(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	one, err := store.CreateNote(general, "One\n")
	if err != nil {
		t.Fatal(err)
	}
	two, err := store.CreateNote(general, "Two\n")
	if err != nil {
		t.Fatal(err)
	}
	three, err := store.CreateNote(general, "Three\n")
	if err != nil {
		t.Fatal(err)
	}

	workspace, err := store.SetNotePinned(general, two.ActiveID, true)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err = store.SetNotePinned(general, one.ActiveID, true)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, workspace.PinnedIDs, GeneralID, two.ActiveID, one.ActiveID)
	assertIDs(t, workspace.OpenIDs, GeneralID, two.ActiveID, one.ActiveID, three.ActiveID)

	workspace, err = store.ReorderPinnedNotes(general, []string{"One", "Two"})
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, workspace.PinnedIDs, GeneralID, one.ActiveID, two.ActiveID)
	assertIDs(t, workspace.OpenIDs, GeneralID, one.ActiveID, two.ActiveID, three.ActiveID)

	reloaded, err := store.LoadWorkspace(general)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, reloaded.PinnedIDs, GeneralID, one.ActiveID, two.ActiveID)
	for _, tab := range reloaded.Tabs {
		wantPinned := tab.ID == GeneralID || tab.ID == one.ActiveID || tab.ID == two.ActiveID
		if tab.Pinned != wantPinned {
			t.Fatalf("tab %s pinned=%t want %t", tab.ID, tab.Pinned, wantPinned)
		}
	}

	workspace, err = store.SetNotePinned(general, one.ActiveID, false)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, workspace.PinnedIDs, GeneralID, two.ActiveID)
	assertIDs(t, workspace.OpenIDs, GeneralID, two.ActiveID, one.ActiveID, three.ActiveID)
}

func TestWorkspacePinnedDetailsMustBeUnpinnedBeforeClose(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	created, err := store.CreateNote(general, "Pinned\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetNotePinned(general, created.ActiveID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CloseNote(general, created.ActiveID); err == nil || !strings.Contains(err.Error(), "unpin") {
		t.Fatalf("close pinned error = %v", err)
	}
	if _, err := store.SetNotePinned(general, created.ActiveID, false); err != nil {
		t.Fatal(err)
	}
	closed, err := store.CloseNote(general, created.ActiveID)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, closed.OpenIDs, GeneralID)
}

func TestWorkspacePinValidationAndDeleteCleanup(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	first, err := store.CreateNote(general, "First\n")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateNote(general, "Second\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetNotePinned(general, GeneralID, false); err == nil {
		t.Fatal("unpinned General")
	}
	if _, err := store.SetNotePinned(general, NewTabID, true); err == nil {
		t.Fatal("pinned New Tab")
	}
	if _, err := store.SetNotePinned(general, first.ActiveID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetNotePinned(general, second.ActiveID, true); err != nil {
		t.Fatal(err)
	}
	for _, selectors := range [][]string{
		{first.ActiveID},
		{first.ActiveID, first.ActiveID},
		{GeneralID, first.ActiveID, second.ActiveID},
	} {
		if _, err := store.ReorderPinnedNotes(general, selectors); err == nil {
			t.Fatalf("accepted invalid reorder %v", selectors)
		}
	}

	deleted, err := store.DeleteNote(general, first.ActiveID)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, deleted.PinnedIDs, GeneralID, second.ActiveID)
}

func TestWorkspaceDropsStalePinnedIDsAndSerializesConcurrentPins(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	created, err := store.CreateNote(general, "Concurrent pin\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetNotePinned(general, created.ActiveID, true); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(created.Active.Path); err != nil {
		t.Fatal(err)
	}
	workspace, err := store.LoadWorkspace(general)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, workspace.PinnedIDs, GeneralID)

	recreated, err := store.CreateNote(general, "Concurrent replacement\n")
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	errors := make(chan error, 12)
	for index := 0; index < 12; index++ {
		group.Add(1)
		go func(pinned bool) {
			defer group.Done()
			_, mutationErr := store.SetNotePinned(general, recreated.ActiveID, pinned)
			errors <- mutationErr
		}(index%2 == 0)
	}
	group.Wait()
	close(errors)
	for mutationErr := range errors {
		if mutationErr != nil {
			t.Fatal(mutationErr)
		}
	}
	workspace, err = store.LoadWorkspace(general)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.PinnedIDs[0] != GeneralID || indexOf(workspace.OpenIDs, recreated.ActiveID) < 0 {
		t.Fatalf("concurrent workspace = %#v", workspace)
	}
}

func assertIDs(t *testing.T, actual []string, expected ...string) {
	t.Helper()
	if strings.Join(actual, ",") != strings.Join(expected, ",") {
		t.Fatalf("IDs = %v, want %v", actual, expected)
	}
}
