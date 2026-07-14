package notes

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestWorkspaceMigratesGeneralAndPersistsOpenTabs(t *testing.T) {
	root := t.TempDir()
	general := filepath.Join(root, "beacon", "notes.md")
	store := FileStore{}
	if _, err := store.Write(general, "[labcore] generate endpoints refactor\nsecond idea\n"); err != nil {
		t.Fatal(err)
	}

	workspace, err := store.LoadWorkspace(general)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.ActiveID != GeneralID || strings.Join(workspace.OpenIDs, ",") != GeneralID || workspace.Active == nil || workspace.Active.Content == "" {
		t.Fatalf("migrated workspace = %#v", workspace)
	}

	workspace, err = store.OpenNote(general, NewTabID)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.ActiveID != NewTabID || strings.Join(workspace.OpenIDs, ",") != "general,new" || workspace.Active != nil {
		t.Fatalf("new tab workspace = %#v", workspace)
	}

	workspace, err = store.CreateNote(general, "[labcore] generate endpoints refactor\n\nExpand here.")
	if err != nil {
		t.Fatal(err)
	}
	id := workspace.ActiveID
	if id == GeneralID || id == NewTabID || len(workspace.OpenIDs) != 3 {
		t.Fatalf("created workspace = %#v", workspace)
	}
	if workspace.Active == nil || workspace.Active.ID != id || workspace.Active.Title != "[labcore] generate endpoints refactor" {
		t.Fatalf("active document = %#v", workspace.Active)
	}
	if filepath.Dir(workspace.Active.Path) != filepath.Join(root, "beacon", "notes") {
		t.Fatalf("detail path = %q", workspace.Active.Path)
	}

	reloaded, err := store.LoadWorkspace(general)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ActiveID != id || strings.Join(reloaded.OpenIDs, ",") != strings.Join(workspace.OpenIDs, ",") {
		t.Fatalf("reloaded workspace = %#v", reloaded)
	}
	assertMode(t, filepath.Join(root, "beacon", "notes"), 0o700)
	assertMode(t, workspace.Active.Path, 0o600)
	assertMode(t, filepath.Join(root, "beacon", "notes", "workspace.json"), 0o600)
}

func TestWorkspacePreventsDuplicatesAndClosingNeverDeletes(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	workspace, err := store.CreateNote(general, "One\n")
	if err != nil {
		t.Fatal(err)
	}
	one := workspace.ActiveID
	path := workspace.Active.Path
	workspace, err = store.CreateNote(general, "Two\n")
	if err != nil {
		t.Fatal(err)
	}
	two := workspace.ActiveID

	workspace, err = store.OpenNote(general, one)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err = store.OpenNote(general, one)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(workspace.OpenIDs, ",") != "general,"+one+","+two {
		t.Fatalf("duplicate open IDs = %v", workspace.OpenIDs)
	}

	workspace, err = store.CloseNote(general, one)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.ActiveID != GeneralID || strings.Join(workspace.OpenIDs, ",") != "general,"+two {
		t.Fatalf("closed workspace = %#v", workspace)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("close deleted detail file: %v", err)
	}

	workspace, err = store.OpenNote(general, "One")
	if err != nil {
		t.Fatal(err)
	}
	if workspace.ActiveID != one || workspace.OpenIDs[len(workspace.OpenIDs)-1] != one {
		t.Fatalf("reopened workspace = %#v", workspace)
	}
	if len(workspace.Tabs) < 3 || workspace.Tabs[1].ID != one {
		t.Fatalf("MRU history = %#v", workspace.Tabs)
	}
	if _, err := store.CloseNote(general, GeneralID); err == nil {
		t.Fatal("closed General")
	}
}

func TestWorkspaceDeletesDetailAndSelectsLeftNeighbor(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	first, err := store.CreateNote(general, "First\nbody")
	if err != nil {
		t.Fatal(err)
	}
	firstID := first.ActiveID
	second, err := store.CreateNote(general, "Second\nbody")
	if err != nil {
		t.Fatal(err)
	}
	secondID := second.ActiveID
	secondPath := second.Active.Path

	workspace, err := store.DeleteNote(general, secondID)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.ActiveID != firstID || strings.Join(workspace.OpenIDs, ",") != "general,"+firstID {
		t.Fatalf("workspace after active delete = %#v", workspace)
	}
	if _, err := os.Lstat(secondPath); !os.IsNotExist(err) {
		t.Fatalf("deleted detail still exists: %v", err)
	}
	for _, tab := range workspace.Tabs {
		if tab.ID == secondID {
			t.Fatalf("deleted tab remains in workspace: %#v", workspace.Tabs)
		}
	}

	if _, err := store.CloseNote(general, firstID); err != nil {
		t.Fatal(err)
	}
	workspace, err = store.DeleteNote(general, "First")
	if err != nil {
		t.Fatal(err)
	}
	if workspace.ActiveID != GeneralID || strings.Join(workspace.OpenIDs, ",") != GeneralID {
		t.Fatalf("workspace after closed delete = %#v", workspace)
	}
	if _, err := store.LoadNote(general, firstID); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("load deleted note error = %v", err)
	}
}

func TestWorkspaceDeleteRejectsProtectedAndAmbiguousNotes(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	first, err := store.CreateNote(general, "Duplicate\nfirst")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateNote(general, "Duplicate\nsecond")
	if err != nil {
		t.Fatal(err)
	}
	for _, selector := range []string{GeneralID, NewTabID} {
		if _, err := store.DeleteNote(general, selector); err == nil || !strings.Contains(err.Error(), "cannot be deleted") {
			t.Fatalf("delete %q error = %v", selector, err)
		}
	}
	if _, err := store.DeleteNote(general, "Duplicate"); err == nil || !strings.Contains(err.Error(), first.ActiveID) || !strings.Contains(err.Error(), second.ActiveID) {
		t.Fatalf("ambiguous delete error = %v", err)
	}
	if _, err := store.DeleteNote(general, "../notes.md"); err == nil {
		t.Fatal("accepted traversal selector")
	}
}

func TestWorkspaceDeleteRejectsSymlinkReplacement(t *testing.T) {
	root := t.TempDir()
	general := filepath.Join(root, "notes.md")
	store := FileStore{}
	workspace, err := store.CreateNote(general, "Replace me\nbody")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "target.md")
	if err := os.WriteFile(target, []byte("must remain"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(workspace.Active.Path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, workspace.Active.Path); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteNote(general, workspace.ActiveID); err == nil || !strings.Contains(err.Error(), "regular files") {
		t.Fatalf("delete symlink error = %v", err)
	}
	contents, err := os.ReadFile(target)
	if err != nil || string(contents) != "must remain" {
		t.Fatalf("symlink target contents=%q err=%v", contents, err)
	}
}

func TestWorkspaceRejectsManifestAndDirectorySymlinks(t *testing.T) {
	store := FileStore{}

	t.Run("manifest", func(t *testing.T) {
		root := t.TempDir()
		general := filepath.Join(root, "notes.md")
		directory := filepath.Join(root, "notes")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, "manifest-target.json")
		if err := os.WriteFile(target, []byte(`{"version":1,"active_id":"general","open_ids":["general"],"entries":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(directory, "workspace.json")); err != nil {
			t.Fatal(err)
		}
		if _, err := store.LoadWorkspace(general); err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("manifest symlink error = %v", err)
		}
	})

	t.Run("directory", func(t *testing.T) {
		root := t.TempDir()
		general := filepath.Join(root, "notes.md")
		target := filepath.Join(root, "elsewhere")
		if err := os.Mkdir(target, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, "notes")); err != nil {
			t.Fatal(err)
		}
		if _, err := store.LoadWorkspace(general); err == nil || !strings.Contains(err.Error(), "regular directory") {
			t.Fatalf("directory symlink error = %v", err)
		}
	})
}

func TestWorkspaceDetailValidationAndConcurrentWrites(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	workspace, err := store.CreateNote(general, "Concurrent\nstart")
	if err != nil {
		t.Fatal(err)
	}
	id := workspace.ActiveID
	if _, err := store.WriteNote(general, id, "bad\x00content"); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("NUL error = %v", err)
	}
	if _, err := store.WriteNote(general, id, strings.Repeat("x", MaxBytes+1)); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("size error = %v", err)
	}

	const writers = 16
	var group sync.WaitGroup
	errors := make(chan error, writers)
	for index := 0; index < writers; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			_, writeErr := store.WriteNote(general, id, "Concurrent\nwriter "+string(rune('A'+index)))
			errors <- writeErr
		}(index)
	}
	group.Wait()
	close(errors)
	for writeErr := range errors {
		if writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	document, err := store.LoadNote(general, id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(document.Content, "Concurrent\nwriter ") {
		t.Fatalf("concurrent document = %q", document.Content)
	}
}

func TestWorkspaceSelectorsAndLiveTitleUpdates(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	first, err := store.CreateNote(general, "Duplicate\nfirst")
	if err != nil {
		t.Fatal(err)
	}
	firstID := first.ActiveID
	second, err := store.CreateNote(general, "Duplicate\nsecond")
	if err != nil {
		t.Fatal(err)
	}
	secondID := second.ActiveID
	if _, err := store.LoadNote(general, "Duplicate"); err == nil || !strings.Contains(err.Error(), firstID) || !strings.Contains(err.Error(), secondID) {
		t.Fatalf("ambiguous selector error = %v", err)
	}

	updated, err := store.WriteNote(general, firstID, "Renamed\nbody")
	if err != nil {
		t.Fatal(err)
	}
	document, err := store.LoadNote(general, "Renamed")
	if err != nil {
		t.Fatal(err)
	}
	if document.ID != firstID || document.Title != "Renamed" || updated.Active == nil || updated.Active.ID != secondID {
		t.Fatalf("updated workspace=%#v document=%#v", updated, document)
	}
	if _, err := store.LoadNote(general, "../notes.md"); err == nil {
		t.Fatal("accepted traversal selector")
	}
}

func TestWorkspaceRecoversDiscoveryAndRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	general := filepath.Join(root, "notes.md")
	directory := filepath.Join(root, "notes")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "external.md"), []byte("External\nbody"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "workspace.json"), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := FileStore{}
	workspace, err := store.LoadWorkspace(general)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspace.Tabs) != 2 || workspace.Tabs[1].ID != "external" || workspace.Tabs[1].Title != "External" {
		t.Fatalf("discovered workspace = %#v", workspace)
	}

	target := filepath.Join(root, "target.md")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(directory, "linked.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadWorkspace(general); err == nil || !strings.Contains(err.Error(), "regular files") {
		t.Fatalf("symlink error = %v", err)
	}
}

func TestWorkspaceHasNoTabCountLimit(t *testing.T) {
	general := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	const count = 40
	for index := 0; index < count; index++ {
		if _, err := store.CreateNote(general, "Note "+strings.Repeat("x", index)+"\n"); err != nil {
			t.Fatalf("create %d: %v", index, err)
		}
	}
	workspace, err := store.LoadWorkspace(general)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspace.OpenIDs) != count+1 || len(workspace.Tabs) != count+1 {
		t.Fatalf("workspace counts open=%d tabs=%d", len(workspace.OpenIDs), len(workspace.Tabs))
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != want {
		t.Fatalf("mode for %s = %o, want %o", path, info.Mode().Perm(), want)
	}
}
