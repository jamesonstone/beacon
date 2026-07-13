package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
)

func TestSetSelectionUpdatesManyProjectsWithoutProbing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	cache := Cache{Directory: paths.Projects}
	const projectCount = 25
	for index := 0; index < projectCount; index++ {
		id := fmt.Sprintf("owner/repo-%02d", index)
		record := cachedRecord(id, 1, model.TrackingTracked)
		record.Snapshot.Projects[0].Name = fmt.Sprintf("repo-%02d", index)
		record.Snapshot.Projects[0].GitHub = id
		record.Snapshot.Lanes[0].GitHub = id
		record.Snapshot.Lanes[0].ID = "git:" + id + "@main"
		record.Snapshot.Projects[0].LaneIDs = []string{record.Snapshot.Lanes[0].ID}
		if err := cache.Write(record); err != nil {
			t.Fatal(err)
		}
	}
	prober := &countingProber{}
	tracker := tracking.Manager{Store: tracking.FileStore{}, Now: time.Now}
	engine := NewEngine(
		config.Config{Path: filepath.Join(root, "config.yaml")}, paths, cache,
		func(context.Context) ([]config.Repository, error) { return nil, nil }, nil, prober, tracker,
	)
	if err := engine.SetSelection([]string{"owner/repo-00"}); err != nil {
		t.Fatal(err)
	}
	snapshot := engine.Snapshot()
	if snapshot.Summary.TrackedProjects != 1 || snapshot.Summary.UntrackedProjects != projectCount-1 {
		t.Fatalf("summary = %#v", snapshot.Summary)
	}
	if prober.calls.Load() != 0 {
		t.Fatalf("tracking selection performed %d probes", prober.calls.Load())
	}
	records, failures := cache.LoadAll()
	if len(failures) != 0 || len(records) != projectCount {
		t.Fatalf("records=%d failures=%v", len(records), failures)
	}
	for _, record := range records {
		if record.ProjectID != "owner/repo-00" && record.LastProbeAt.IsZero() {
			t.Fatalf("newly untracked %s was immediately due for a probe", record.ProjectID)
		}
	}
}

func TestNewEngineAppliesDurableTrackingStateBeforeScheduling(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	cfg := config.Config{Path: filepath.Join(root, "config.yaml")}
	record := cachedRecord("owner/repo", 1, model.TrackingTracked)
	record.Snapshot.ConfigPath = cfg.Path
	tracker := tracking.Manager{Store: tracking.FileStore{}, Now: time.Now}
	untracked, err := tracker.SetSelection(record.Snapshot, []string{})
	if err != nil {
		t.Fatal(err)
	}
	paths.State = untracked.Tracking.Path
	state, err := (tracking.FileStore{}).Load(paths.State)
	if err != nil || len(state.Untracked) != 1 {
		t.Fatalf("durable state=%#v err=%v", state, err)
	}
	reconciled, err := tracker.ReconcileAt(record.Snapshot, paths.State)
	if err != nil || reconciled.Summary.UntrackedProjects != 1 {
		t.Fatalf("direct reconciliation=%#v err=%v", reconciled.Summary, err)
	}
	cache := Cache{Directory: paths.Projects}
	if err := cache.Write(record); err != nil {
		t.Fatal(err)
	}
	assembled := Assemble([]ProjectRecord{record}, cfg.Path, paths.State, time.Now())
	assembledFingerprint, err := tracking.Fingerprint(assembled.Projects[0], assembled.Lanes)
	if err != nil || assembledFingerprint != state.Untracked[0].Baseline {
		t.Fatalf("assembled fingerprint=%s baseline=%s err=%v", assembledFingerprint, state.Untracked[0].Baseline, err)
	}
	assembledReconciled, err := tracker.ReconcileAt(assembled, paths.State)
	if err != nil || assembledReconciled.Summary.UntrackedProjects != 1 {
		t.Fatalf("assembled reconciliation=%#v err=%v", assembledReconciled.Summary, err)
	}
	engine := NewEngine(cfg, paths, cache, nil, nil, nil, tracker)
	stored, found := engine.record("owner/repo")
	if !found || stored.Snapshot.Projects[0].TrackingState != model.TrackingUntracked {
		t.Fatalf("engine record=%#v found=%t", stored.Snapshot.Projects, found)
	}
	if stored.LastProbeAt.IsZero() {
		t.Fatal("startup tracking application left newly untracked project immediately due for a probe")
	}
	snapshot := engine.Snapshot()
	if snapshot.Summary.TrackedProjects != 0 || snapshot.Summary.UntrackedProjects != 1 {
		t.Fatalf("startup snapshot ignored durable tracking state: %#v", snapshot.Summary)
	}
}

func TestTrackingSelectionSupersedesInFlightScan(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo", Base: "main", Remote: "origin"}
	cache := Cache{Directory: paths.Projects}
	if err := cache.Write(cachedRecord(repository.GitHub, 1, model.TrackingTracked)); err != nil {
		t.Fatal(err)
	}
	scanStarted := make(chan struct{})
	releaseScan := make(chan struct{})
	tracker := tracking.Manager{Store: tracking.FileStore{}, Now: time.Now}
	engine := NewEngine(
		config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 1, UntrackedProbeInterval: time.Hour}},
		paths, cache,
		func(context.Context) ([]config.Repository, error) { return []config.Repository{repository}, nil },
		func(context.Context, config.Repository, bool, func(string)) (model.Snapshot, error) {
			close(scanStarted)
			<-releaseScan
			updated := cachedRecord(repository.GitHub, 1, model.TrackingTracked).Snapshot
			updated.Lanes[0].Worktree.HeadOID = "newer-scan-head"
			return updated, nil
		},
		nil, tracker,
	)
	if _, err := engine.Refresh(context.Background(), repository.GitHub, true); err != nil {
		t.Fatal(err)
	}
	select {
	case <-scanStarted:
	case <-time.After(time.Second):
		t.Fatal("scan did not start")
	}
	if err := engine.SetSelection([]string{}); err != nil {
		t.Fatal(err)
	}
	close(releaseScan)
	waitForRefresh(t, engine)
	snapshot := engine.Snapshot()
	if snapshot.Summary.TrackedProjects != 0 || snapshot.Summary.UntrackedProjects != 1 {
		t.Fatalf("in-flight scan replaced tracking selection: %#v", snapshot.Summary)
	}
}
