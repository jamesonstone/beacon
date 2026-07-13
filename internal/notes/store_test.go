package notes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestResolvePathHonorsXDGDataHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)
	path, err := ResolvePath()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, "beacon", "notes.md") {
		t.Fatalf("path = %q", path)
	}
}

func TestFileStoreSerializesConcurrentAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	const writers = 24
	var waitGroup sync.WaitGroup
	for index := 0; index < writers; index++ {
		waitGroup.Add(1)
		go func(value int) {
			defer waitGroup.Done()
			if _, err := store.Append(path, fmt.Sprintf("note-%02d", value)); err != nil {
				t.Errorf("append note %d: %v", value, err)
			}
		}(index)
	}
	waitGroup.Wait()
	document, err := store.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < writers; index++ {
		if !strings.Contains(document.Content, fmt.Sprintf("note-%02d\n", index)) {
			t.Fatalf("missing note %d in %q", index, document.Content)
		}
	}
}

func TestFileStoreSerializesCrossProcessAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	const writers = 8
	start := make(chan struct{})
	results := make(chan error, writers)
	for index := 0; index < writers; index++ {
		go func(value int) {
			<-start
			command := exec.Command(os.Args[0], "-test.run=^TestFileStoreAppendHelper$", "--", path, fmt.Sprintf("process-%02d", value))
			command.Env = append(os.Environ(), "BEACON_NOTES_APPEND_HELPER=1")
			output, err := command.CombinedOutput()
			if err != nil {
				results <- fmt.Errorf("append helper: %w: %s", err, output)
				return
			}
			results <- nil
		}(index)
	}
	close(start)
	for index := 0; index < writers; index++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	document, err := (FileStore{}).Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < writers; index++ {
		if !strings.Contains(document.Content, fmt.Sprintf("process-%02d\n", index)) {
			t.Fatalf("missing process note %d in %q", index, document.Content)
		}
	}
}

func TestFileStoreAppendHelper(t *testing.T) {
	if os.Getenv("BEACON_NOTES_APPEND_HELPER") != "1" {
		return
	}
	arguments := os.Args
	separator := -1
	for index, argument := range arguments {
		if argument == "--" {
			separator = index
			break
		}
	}
	if separator < 0 || len(arguments) != separator+3 {
		t.Fatalf("helper arguments = %v", arguments)
	}
	if _, err := (FileStore{}).Append(arguments[separator+1], arguments[separator+2]); err != nil {
		t.Fatal(err)
	}
}

func TestFileStoreRoundTripAppendAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "beacon", "notes.md")
	store := FileStore{}
	document, err := store.Load(path)
	if err != nil || document.Content != "" || document.Path != path {
		t.Fatalf("missing document = %#v, %v", document, err)
	}
	document, err = store.Write(path, "# Signal Log\n\nFirst thought")
	if err != nil {
		t.Fatal(err)
	}
	if document.Content != "# Signal Log\n\nFirst thought" || document.UpdatedAt.IsZero() {
		t.Fatalf("written document = %#v", document)
	}
	document, err = store.Append(path, "- another thought")
	if err != nil {
		t.Fatal(err)
	}
	if document.Content != "# Signal Log\n\nFirst thought\n- another thought\n" {
		t.Fatalf("appended content = %q", document.Content)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %o", info.Mode().Perm())
	}
	info, err = os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode = %o", info.Mode().Perm())
	}
}

func TestFileStoreRejectsOversizeAndNUL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	store := FileStore{}
	for _, content := range []string{strings.Repeat("x", MaxBytes+1), "bad\x00note"} {
		if _, err := store.Write(path, content); err == nil {
			t.Fatalf("accepted invalid content of length %d", len(content))
		}
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", MaxBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(path); err == nil {
		t.Fatal("loaded oversized notes file")
	}
}
