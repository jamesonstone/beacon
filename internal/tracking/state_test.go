package tracking

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileStoreMissingStateIsEmptyWithoutCreatingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tracking.json")
	state, err := (FileStore{}).Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != Version || !state.Initialized || state.Known == nil || state.Untracked == nil || len(state.Untracked) != 0 {
		t.Fatalf("state = %#v", state)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("missing state created a file: %v", err)
	}
}

func TestFileStoreRoundTripsSortedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "tracking.json")
	state := State{Version: Version, Initialized: true, Untracked: []Entry{
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
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("state mode = %v, %v", info, err)
	}
}

func TestFileStoreLoadsVersionOneForDeferredMembershipMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tracking.json")
	contents := fmt.Sprintf(`{"version":1,"projects":[{"github":"owner/repo","name":"repo","path":"/repo","state":"muted","muted_at":"2026-07-11T12:00:00Z","baseline":"%s"}]}`, strings.Repeat("a", 64))
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := (FileStore{}).Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != Version || state.Initialized || len(state.Known) != 1 || len(state.Untracked) != 1 {
		t.Fatalf("migrated state = %#v", state)
	}
}

func TestFileStoreRejectsUnknownDuplicateAndInvalidState(t *testing.T) {
	for _, test := range []struct {
		name     string
		contents string
	}{
		{name: "unknown", contents: `{"version":1,"projects":[],"unknown":true}`},
		{name: "duplicate", contents: fmt.Sprintf(`{"version":1,"projects":[{"github":"owner/repo","name":"repo","path":"/repo","state":"muted","muted_at":"2026-07-11T12:00:00Z","baseline":"%s"},{"github":"owner/repo","name":"repo","path":"/repo","state":"muted","muted_at":"2026-07-11T12:00:00Z","baseline":"%s"}]}`, strings.Repeat("a", 64), strings.Repeat("b", 64))},
		{name: "baseline", contents: `{"version":1,"projects":[{"github":"owner/repo","name":"repo","path":"/repo","state":"muted","muted_at":"2026-07-11T12:00:00Z","baseline":"invalid"}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tracking.json")
			if err := os.WriteFile(path, []byte(test.contents), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := (FileStore{}).Load(path); err == nil {
				t.Fatal("expected state validation failure")
			}
		})
	}
}

func TestMigrateLegacyTrackingYAML(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")
	legacyPath := filepath.Join(root, "tracking.yaml")
	statePath := filepath.Join(root, "state", "tracking.json")
	contents := "version: 1\nuntracked:\n  - github: owner/repo\n    name: repo\n    path: /repo\n    untracked_at: 2026-07-11T12:00:00Z\n    baseline: " + strings.Repeat("a", 64) + "\n"
	if err := os.WriteFile(legacyPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	migrated, err := MigrateLegacy(configPath, statePath)
	if err != nil || !migrated {
		t.Fatalf("migrated=%t err=%v", migrated, err)
	}
	state, err := (FileStore{}).Load(statePath)
	if err != nil || state.Version != Version || state.Initialized || len(state.Untracked) != 1 || state.Untracked[0].State != StateMuted {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	if _, err := os.Stat(legacyPath + ".migrated"); err != nil {
		t.Fatalf("legacy state was not archived: %v", err)
	}
}

func TestResolvePathUsesResolvedConfigDirectory(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	configPath := filepath.Join(t.TempDir(), "custom", "config.yaml")
	path, err := ResolvePath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(stateHome, "beacon", "tracking.json") {
		t.Fatalf("tracking path = %q", path)
	}
}
