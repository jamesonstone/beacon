package agent

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
	"github.com/jamesonstone/beacon/internal/workset"
)

func TestFrequentLaneCadenceRequiresProjectFollowing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	now := time.Date(2026, 7, 11, 12, 2, 0, 0, time.UTC)
	repository := config.Repository{Name: "repo", GitHub: "owner/repo"}
	record := cachedRecord(repository.GitHub, 1, model.TrackingTracked)
	record.Snapshot.Projects[0].FollowState = model.FollowFollowing
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
		config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{TrackedRefreshInterval: time.Minute, UntrackedProbeInterval: time.Hour}},
		testPaths(root), Cache{Directory: filepath.Join(root, "cache")}, nil, nil, nil, tracking.Manager{},
	)
	engine.Now = func() time.Time { return now }
	engine.WorkingSet = &manager
	engine.storeRecord(record)

	if !engine.due(repository, engine.frequentRepositories()) {
		t.Fatal("followed project lane did not use frequent local cadence")
	}
	if err := engine.SetSelection([]string{}); err != nil {
		t.Fatal(err)
	}
	outside, found := engine.record(repository.GitHub)
	if !found || outside.Snapshot.Projects[0].FollowState != model.FollowQuiet {
		t.Fatalf("unfollowed record = %#v, found = %t", outside.Snapshot.Projects, found)
	}
	if engine.due(repository, engine.frequentRepositories()) {
		t.Fatal("outside project lane retained frequent cadence")
	}
	if err := engine.SetSelection([]string{repository.GitHub}); err != nil {
		t.Fatal(err)
	}
	followed, found := engine.record(repository.GitHub)
	if !found || followed.Snapshot.Projects[0].FollowState != model.FollowFollowing {
		t.Fatalf("followed record = %#v, found = %t", followed.Snapshot.Projects, found)
	}
	engine.Now = func() time.Time { return now.Add(2 * time.Minute) }
	if !engine.due(repository, engine.frequentRepositories()) {
		t.Fatal("refollowed project lane did not restore frequent cadence")
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
