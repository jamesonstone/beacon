package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestResolvePathsHonorsXDGAndUsesUserScopedLayout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	paths, err := ResolvePaths(filepath.Join(home, ".config", "beacon", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if paths.State != filepath.Join(home, "state", "beacon", "tracking.json") || paths.Socket != filepath.Join(home, "cache", "beacon", "agent.sock") {
		t.Fatalf("paths = %#v", paths)
	}
	if paths.LaunchAgent != filepath.Join(home, "Library", "LaunchAgents", "com.jamesonstone.beacon.agent.plist") {
		t.Fatalf("LaunchAgent path = %s", paths.LaunchAgent)
	}
}

func TestCacheRoundTripAssemblyAndCorruptionQuarantine(t *testing.T) {
	directory := t.TempDir()
	cache := Cache{Directory: directory, Now: func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) }}
	record := cachedRecord("owner/repo", 3, model.TrackingTracked)
	if err := cache.Write(record); err != nil {
		t.Fatal(err)
	}
	records, failures := cache.LoadAll()
	if len(failures) != 0 || len(records) != 1 || records[0].Revision != 3 {
		t.Fatalf("records=%#v failures=%v", records, failures)
	}
	snapshot := Assemble(records, "/config.yaml", "/tracking.json", time.Now())
	if snapshot.Summary.TrackedProjects != 1 || len(snapshot.Groups.Idle) != 1 {
		t.Fatalf("assembled snapshot = %#v", snapshot)
	}
	if err := os.WriteFile(filepath.Join(directory, "broken.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, failures = cache.LoadAll()
	if len(failures) == 0 {
		t.Fatal("expected corrupt cache failure")
	}
	matches, err := filepath.Glob(filepath.Join(directory, "broken.json.corrupt-*"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("quarantined files=%v err=%v", matches, err)
	}
}

func TestCacheLoadUpgradesSchemaTwoSnapshotWithoutQuarantine(t *testing.T) {
	directory := t.TempDir()
	cache := Cache{Directory: directory}
	record := cachedRecord("owner/legacy", 7, model.TrackingTracked)
	record.Snapshot.SchemaVersion = 2
	if err := cache.Write(record); err != nil {
		t.Fatal(err)
	}

	records, failures := cache.LoadAll()
	if len(failures) != 0 || len(records) != 1 {
		t.Fatalf("records=%#v failures=%v", records, failures)
	}
	if records[0].Snapshot.SchemaVersion != model.SchemaVersion || records[0].Snapshot.WorkingSet.Active == nil || records[0].Snapshot.WorkingSet.Parked == nil {
		t.Fatalf("upgraded snapshot = %#v", records[0].Snapshot)
	}
	matches, err := filepath.Glob(filepath.Join(directory, "*.corrupt-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("legacy cache was quarantined: %v err=%v", matches, err)
	}
}
