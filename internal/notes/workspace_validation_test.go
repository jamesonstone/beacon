package notes

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

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
