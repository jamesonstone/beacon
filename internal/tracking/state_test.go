package tracking

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileStoreMissingStateIsEmptyWithoutCreatingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tracking.yaml")
	state, err := (FileStore{}).Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != Version || state.Untracked == nil || len(state.Untracked) != 0 {
		t.Fatalf("state = %#v", state)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("missing state created a file: %v", err)
	}
}

func TestFileStoreRoundTripsSortedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "tracking.yaml")
	state := State{Version: Version, Untracked: []Entry{
		{GitHub: "owner/zeta", Name: "zeta", Path: "/zeta", UntrackedAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC), Baseline: strings.Repeat("b", 64)},
		{GitHub: "owner/alpha", Name: "alpha", Path: "/alpha", UntrackedAt: time.Date(2026, 7, 11, 11, 0, 0, 0, time.UTC), Baseline: strings.Repeat("a", 64)},
	}}
	store := FileStore{}
	if err := store.Write(path, state); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Untracked) != 2 || loaded.Untracked[0].GitHub != "owner/alpha" {
		t.Fatalf("loaded state = %#v", loaded)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o644 {
		t.Fatalf("state mode = %v, %v", info, err)
	}
}

func TestFileStoreRejectsUnknownDuplicateAndInvalidState(t *testing.T) {
	for _, test := range []struct {
		name     string
		contents string
	}{
		{name: "unknown", contents: "version: 1\nunknown: true\n"},
		{name: "duplicate", contents: "version: 1\nuntracked:\n  - github: owner/repo\n    name: repo\n    path: /repo\n    untracked_at: 2026-07-11T12:00:00Z\n    baseline: " + strings.Repeat("a", 64) + "\n  - github: owner/repo\n    name: repo\n    path: /repo\n    untracked_at: 2026-07-11T12:00:00Z\n    baseline: " + strings.Repeat("b", 64) + "\n"},
		{name: "baseline", contents: "version: 1\nuntracked:\n  - github: owner/repo\n    name: repo\n    path: /repo\n    untracked_at: 2026-07-11T12:00:00Z\n    baseline: invalid\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tracking.yaml")
			if err := os.WriteFile(path, []byte(test.contents), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := (FileStore{}).Load(path); err == nil {
				t.Fatal("expected state validation failure")
			}
		})
	}
}

func TestResolvePathUsesResolvedConfigDirectory(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom", "config.yaml")
	path, err := ResolvePath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(filepath.Dir(configPath), "tracking.yaml") {
		t.Fatalf("tracking path = %q", path)
	}
}
