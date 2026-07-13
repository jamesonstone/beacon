package agent

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
	"github.com/jamesonstone/beacon/internal/workset"
)

func TestResumedLaneUsesFrequentCadenceInsideUntrackedProject(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	now := time.Date(2026, 7, 11, 12, 2, 0, 0, time.UTC)
	repository := config.Repository{Name: "repo", GitHub: "owner/repo"}
	record := cachedRecord(repository.GitHub, 1, model.TrackingUntracked)
	record.UpdatedAt = now.Add(-2 * time.Minute)
	record.LastProbeAt = now
	manager := workset.Manager{Store: workset.FileStore{}, Now: func() time.Time { return now }}
	working, err := manager.Reconcile(record.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.SetAttention(working, record.Snapshot.Lanes[0].ID, model.AttentionActive); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(
		config.Config{Settings: config.Settings{TrackedRefreshInterval: time.Minute, UntrackedProbeInterval: time.Hour}},
		testPaths(root), Cache{Directory: filepath.Join(root, "cache")}, nil, nil, nil, tracking.Manager{},
	)
	engine.Now = func() time.Time { return now }
	engine.WorkingSet = &manager
	engine.storeRecord(record)

	if !engine.due(repository, engine.frequentRepositories()) {
		t.Fatal("resumed lane did not use frequent local cadence")
	}
	if _, err := manager.SetAttention(working, record.Snapshot.Lanes[0].ID, model.AttentionParked); err != nil {
		t.Fatal(err)
	}
	if engine.due(repository, engine.frequentRepositories()) {
		t.Fatal("parked lane did not return to slow probe cadence")
	}
}

func TestRefreshPublishesUncachedProjectPlaceholderBeforeCollection(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo", Base: "main", Remote: "origin"}
	release := make(chan struct{})
	engine := NewEngine(
		config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Minute, UntrackedProbeInterval: time.Minute}},
		paths,
		Cache{Directory: paths.Projects},
		func(context.Context) ([]config.Repository, error) { return []config.Repository{repository}, nil },
		func(_ context.Context, _ config.Repository, _ bool, _ func(string)) (model.Snapshot, error) {
			<-release
			return cachedRecord(repository.GitHub, 1, model.TrackingTracked).Snapshot, nil
		},
		nil,
		tracking.Manager{Store: tracking.FileStore{}},
	)
	events, unsubscribe := engine.Subscribe()
	defer unsubscribe()
	if _, err := engine.Refresh(context.Background(), "", true); err != nil {
		t.Fatal(err)
	}
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		waitForRefresh(t, engine)
	}()
	for deadline := time.After(2 * time.Second); ; {
		select {
		case event := <-events:
			if event.Type != EventProjectDiscovered {
				continue
			}
			if len(event.Projects) != 1 || event.Projects[0].ProjectID != repository.GitHub || event.Projects[0].Stage != "cached" {
				t.Fatalf("placeholder event = %#v", event)
			}
			return
		case <-deadline:
			t.Fatal("project discovery event was not published")
		}
	}
}

func TestRefreshReturnsBeforeRepositoryDiscoveryCompletes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	discoveryStarted := make(chan struct{})
	releaseDiscovery := make(chan struct{})
	engine := NewEngine(
		config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Minute, UntrackedProbeInterval: time.Minute}},
		paths,
		Cache{Directory: paths.Projects},
		func(context.Context) ([]config.Repository, error) {
			close(discoveryStarted)
			<-releaseDiscovery
			return []config.Repository{}, nil
		},
		nil,
		nil,
		tracking.Manager{},
	)

	startedAt := time.Now()
	scanID, err := engine.Refresh(context.Background(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	if scanID == "" {
		t.Fatal("refresh returned an empty scan ID")
	}
	if elapsed := time.Since(startedAt); elapsed > 100*time.Millisecond {
		t.Fatalf("refresh acknowledgement took %s", elapsed)
	}
	select {
	case <-discoveryStarted:
	case <-time.After(time.Second):
		t.Fatal("repository discovery did not start")
	}
	close(releaseDiscovery)
	waitForRefresh(t, engine)
}

func TestConcurrentDistinctRefreshIsQueuedAndDuplicateIsCoalesced(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	repositories := []config.Repository{
		{Name: "one", Path: root, GitHub: "owner/one"},
		{Name: "two", Path: root, GitHub: "owner/two"},
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var mutex sync.Mutex
	calls := make(map[string]int)
	engine := NewEngine(
		config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 2, TrackedRefreshInterval: time.Minute, UntrackedProbeInterval: time.Minute}},
		paths,
		Cache{Directory: paths.Projects},
		func(context.Context) ([]config.Repository, error) { return repositories, nil },
		func(_ context.Context, repository config.Repository, _ bool, _ func(string)) (model.Snapshot, error) {
			mutex.Lock()
			calls[repository.GitHub]++
			mutex.Unlock()
			if repository.GitHub == "owner/one" {
				close(started)
				<-release
			}
			return cachedRecord(repository.GitHub, 1, model.TrackingTracked).Snapshot, nil
		},
		nil,
		tracking.Manager{Store: tracking.FileStore{}},
	)
	scanID, err := engine.Refresh(context.Background(), "owner/one", true)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	for index := 0; index < 2; index++ {
		queuedID, queueErr := engine.Refresh(context.Background(), "owner/two", true)
		if queueErr != nil || queuedID != scanID {
			t.Fatalf("queued refresh id=%q err=%v", queuedID, queueErr)
		}
	}
	close(release)
	waitForRefresh(t, engine)
	mutex.Lock()
	defer mutex.Unlock()
	if calls["owner/one"] != 1 || calls["owner/two"] != 1 {
		t.Fatalf("scan calls = %#v", calls)
	}
}

func TestFailedRefreshStillAdvancesNextProjectRevision(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo"}
	var calls atomic.Int32
	engine := NewEngine(
		config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Minute, UntrackedProbeInterval: time.Minute}},
		paths,
		Cache{Directory: paths.Projects},
		func(context.Context) ([]config.Repository, error) { return []config.Repository{repository}, nil },
		func(context.Context, config.Repository, bool, func(string)) (model.Snapshot, error) {
			if calls.Add(1) == 1 {
				return model.Snapshot{}, errors.New("temporary failure")
			}
			return cachedRecord(repository.GitHub, 1, model.TrackingTracked).Snapshot, nil
		},
		nil,
		tracking.Manager{Store: tracking.FileStore{}},
	)
	if _, err := engine.Refresh(context.Background(), repository.GitHub, true); err != nil {
		t.Fatal(err)
	}
	waitForRefresh(t, engine)
	if revision := engine.revision(repository.GitHub); revision != 1 {
		t.Fatalf("failed revision = %d", revision)
	}
	if _, err := engine.Refresh(context.Background(), repository.GitHub, true); err != nil {
		t.Fatal(err)
	}
	waitForRefresh(t, engine)
	if revision := engine.revision(repository.GitHub); revision != 2 {
		t.Fatalf("successful retry revision = %d", revision)
	}
}

func TestCollectionErrorPreservesLastGoodProjectCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo"}
	lastGood := cachedRecord(repository.GitHub, 3, model.TrackingTracked)
	cache := Cache{Directory: paths.Projects}
	if err := cache.Write(lastGood); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(
		config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Minute, UntrackedProbeInterval: time.Minute}},
		paths,
		cache,
		func(context.Context) ([]config.Repository, error) { return []config.Repository{repository}, nil },
		func(context.Context, config.Repository, bool, func(string)) (model.Snapshot, error) {
			failed := cachedRecord(repository.GitHub, 4, model.TrackingTracked).Snapshot
			failed.Lanes[0].Worktree.HeadOID = "incomplete-head"
			failed.Errors = []model.ScanError{{Repository: repository.Name, Stage: "github", Message: "timeout"}}
			return failed, nil
		},
		nil,
		tracking.Manager{Store: tracking.FileStore{}},
	)
	if _, err := engine.Refresh(context.Background(), repository.GitHub, true); err != nil {
		t.Fatal(err)
	}
	waitForRefresh(t, engine)
	if got := engine.Snapshot().Lanes[0].Worktree.HeadOID; got != "head" {
		t.Fatalf("last-good head was replaced by %q", got)
	}
	records, failures := cache.LoadAll()
	if len(failures) != 0 || len(records) != 1 || records[0].Revision != 3 {
		t.Fatalf("cache records=%#v failures=%v", records, failures)
	}
}

func TestFirstCollectionErrorCachesPartialProjectEvidence(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo"}
	cache := Cache{Directory: paths.Projects}
	partial := cachedRecord(repository.GitHub, 1, model.TrackingTracked).Snapshot
	partial.Errors = []model.ScanError{{Repository: repository.Name, Stage: "github", Message: "authentication required"}}
	engine := NewEngine(
		config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Minute, UntrackedProbeInterval: time.Minute}},
		paths,
		cache,
		func(context.Context) ([]config.Repository, error) { return []config.Repository{repository}, nil },
		func(context.Context, config.Repository, bool, func(string)) (model.Snapshot, error) {
			return partial, nil
		},
		nil,
		tracking.Manager{Store: tracking.FileStore{}},
	)
	if _, err := engine.Refresh(context.Background(), repository.GitHub, true); err != nil {
		t.Fatal(err)
	}
	waitForRefresh(t, engine)
	snapshot := engine.Snapshot()
	if len(snapshot.Projects) != 1 || len(snapshot.Errors) != 1 {
		t.Fatalf("partial snapshot was not retained: %#v", snapshot)
	}
	records, failures := cache.LoadAll()
	if len(failures) != 0 || len(records) != 1 || records[0].Stage != "failed" {
		t.Fatalf("cache records=%#v failures=%v", records, failures)
	}
}
