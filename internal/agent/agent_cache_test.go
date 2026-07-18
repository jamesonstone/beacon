package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/checkoutwarn"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestResolvePathsHonorsXDGAndUsesUserScopedLayout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	paths, err := ResolvePaths(filepath.Join(home, ".config", "beacon", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if paths.State != filepath.Join(home, "state", "beacon", "tracking.json") || paths.Socket != filepath.Join(home, "cache", "beacon", "agent.sock") {
		t.Fatalf("paths = %#v", paths)
	}
	if paths.Notes != filepath.Join(home, "data", "beacon", "notes.md") {
		t.Fatalf("notes path = %s", paths.Notes)
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

func TestAssembleNormalizesAdditiveFollowingWorkspaceArraysFromCurrentCache(t *testing.T) {
	record := cachedRecord("owner/current-cache", 8, model.TrackingTracked)
	record.Snapshot.Lanes[0].PullRequest = &model.PullRequest{
		Number:   39,
		Feedback: model.Feedback{Threads: []model.ReviewThread{{ID: "thread-1"}}},
	}
	record.Snapshot.Lanes[0].PullRequest.Feedback.Threads[0].Comments = nil
	legacyLane := record.Snapshot.Lanes[0]
	legacyLane.ID = "legacy-without-threads"
	legacyLane.PullRequest = &model.PullRequest{Number: 38, Feedback: model.Feedback{Threads: nil}}
	record.Snapshot.Lanes = append(record.Snapshot.Lanes, legacyLane)

	snapshot := Assemble([]ProjectRecord{record}, "/config.yaml", "/tracking.json", time.Now())
	if snapshot.WorkingSet.Order == nil {
		t.Fatal("working-set order must encode as an empty array before reconciliation")
	}
	for _, lane := range snapshot.Lanes {
		if lane.PullRequest == nil || lane.PullRequest.Feedback.Threads == nil {
			t.Fatalf("rich evidence arrays were not normalized: %#v", lane.PullRequest)
		}
		for _, thread := range lane.PullRequest.Feedback.Threads {
			if thread.Comments == nil {
				t.Fatalf("review comments were not normalized: %#v", thread)
			}
		}
	}
}

func TestCacheLoadsVersionOneAndPersistsCheckoutConfirmations(t *testing.T) {
	directory := t.TempDir()
	cache := Cache{Directory: directory}
	legacy := cachedRecord("owner/legacy-record", 2, model.TrackingTracked)
	legacy.Version = 1
	if err := cache.Write(ProjectRecord{
		Version: CacheVersion, ProjectID: legacy.ProjectID, Revision: legacy.Revision,
		Stage: legacy.Stage, UpdatedAt: legacy.UpdatedAt, Snapshot: legacy.Snapshot,
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, ProjectFileName(legacy.ProjectID))
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), `"version": 2`, `"version": 1`, 1))
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	records, failures := cache.LoadAll()
	if len(failures) != 0 || len(records) != 1 || records[0].Version != CacheVersion {
		t.Fatalf("records=%#v failures=%v", records, failures)
	}

	records[0].CheckoutConfirmations = []checkoutwarn.Confirmation{{
		PullRequestNumber: 32, Branch: "GH-31", Base: "main", HeadOID: "head",
		Status: checkoutwarn.StatusConfirmed,
	}}
	if err := cache.Write(records[0]); err != nil {
		t.Fatal(err)
	}
	reloaded, failures := cache.LoadAll()
	if len(failures) != 0 || len(reloaded) != 1 || len(reloaded[0].CheckoutConfirmations) != 1 {
		t.Fatalf("records=%#v failures=%v", reloaded, failures)
	}
}

func TestAssembleUsesConfiguredRecentWindow(t *testing.T) {
	now := time.Date(2026, 7, 11, 14, 0, 0, 0, time.UTC)
	record := cachedRecord("owner/repo", 1, model.TrackingUntracked)
	record.Snapshot.Projects[0].FollowState = model.FollowRecent
	record.Snapshot.Projects[0].LastActivityAt = now.Add(-2 * time.Hour)
	record.Snapshot.Projects[0].ActivityReason = "new local changes"

	snapshot := AssembleWithRecentWindow([]ProjectRecord{record}, "/config.yaml", "/tracking.json", now, time.Hour)
	if snapshot.Projects[0].FollowState != model.FollowQuiet || snapshot.Summary.RecentProjects != 0 || snapshot.Summary.QuietProjects != 1 {
		t.Fatalf("configured recent window ignored: project=%#v summary=%#v", snapshot.Projects[0], snapshot.Summary)
	}
}
