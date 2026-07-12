package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
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

func TestSchedulerBoundsConcurrencyAndCoalescesDuplicates(t *testing.T) {
	var active atomic.Int32
	var maximum atomic.Int32
	var calls atomic.Int32
	Scheduler{MaxParallel: 2}.Run(context.Background(), []string{"a", "b", "a", "c"}, func(context.Context, string) {
		current := active.Add(1)
		for {
			previous := maximum.Load()
			if current <= previous || maximum.CompareAndSwap(previous, current) {
				break
			}
		}
		calls.Add(1)
		time.Sleep(15 * time.Millisecond)
		active.Add(-1)
	})
	if calls.Load() != 3 || maximum.Load() > 2 {
		t.Fatalf("calls=%d maximum=%d", calls.Load(), maximum.Load())
	}
}

func TestProtocolRejectsUnsupportedAndMalformedRequests(t *testing.T) {
	valid := []byte(`{"protocol_version":1,"request_id":"one","type":"get_snapshot"}` + "\n")
	request, err := DecodeRequest(bytes.NewReader(valid))
	if err != nil || request.Type != RequestGetSnapshot {
		t.Fatalf("request=%#v err=%v", request, err)
	}
	for _, payload := range []string{
		`{"protocol_version":2,"request_id":"one","type":"get_snapshot"}`,
		`{"protocol_version":1,"request_id":"","type":"get_snapshot"}`,
		`{"protocol_version":1,"request_id":"one","type":"get_snapshot","unknown":true}`,
	} {
		if _, err := DecodeRequest(bytes.NewBufferString(payload)); err == nil {
			t.Fatalf("accepted invalid request %s", payload)
		}
	}
}

func TestSlowSubscriberRetainsTerminalSnapshotEvent(t *testing.T) {
	hub := newEventHub()
	events, unsubscribe := hub.Subscribe()
	defer unsubscribe()
	for index := 0; index < 80; index++ {
		hub.Publish(Event{ProtocolVersion: ProtocolVersion, Type: EventProjectQueued, ProjectID: fmt.Sprintf("project-%d", index)})
	}
	snapshot := model.Snapshot{SchemaVersion: model.SchemaVersion}
	hub.Publish(Event{ProtocolVersion: ProtocolVersion, Type: EventScanCompleted, ScanID: "scan", Snapshot: &snapshot})
	for index := 0; index < 64; index++ {
		event := <-events
		if event.Type == EventScanCompleted && event.ScanID == "scan" && event.Snapshot != nil {
			return
		}
	}
	t.Fatal("terminal snapshot event was dropped for a slow subscriber")
}

func TestServerClientSnapshotStatusAndShutdown(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "beacon-agent-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	paths := testPaths(root)
	cfg := config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Hour, UntrackedProbeInterval: time.Hour}}
	engine := NewEngine(cfg, paths, Cache{Directory: paths.Projects}, func(context.Context) ([]config.Repository, error) {
		return []config.Repository{}, nil
	}, nil, nil, tracking.Manager{})
	server := &Server{Paths: paths, Engine: engine}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(context.Background()) }()
	waitForFile(t, paths.Socket)
	client := Client{Socket: paths.Socket}
	event, err := client.Request(context.Background(), Request{Type: RequestGetSnapshot})
	if err != nil || event.Snapshot == nil || event.Snapshot.SchemaVersion != model.SchemaVersion {
		t.Fatalf("snapshot event=%#v err=%v", event, err)
	}
	statusEvent, err := client.Request(context.Background(), Request{Type: RequestGetAgentStatus})
	if err != nil || statusEvent.Status == nil || !statusEvent.Status.Running {
		t.Fatalf("status event=%#v err=%v", statusEvent, err)
	}
	if _, err := client.Request(context.Background(), Request{Type: RequestShutdown}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-serverDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop")
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

func TestProbeScopesPullRequestsAndIssuesToConfiguredIdentity(t *testing.T) {
	runner := &probeCommandRunner{}
	result, err := (Prober{Runner: runner}).Probe(context.Background(), config.Repository{Name: "repo", Path: "/repo", GitHub: "owner/repo"}, "@me")
	if err != nil {
		t.Fatal(err)
	}
	if result.Combined == "" || len(runner.commands) != 5 {
		t.Fatalf("result=%#v commands=%v", result, runner.commands)
	}
	joined := strings.Join(runner.commands, "\n")
	if !strings.Contains(joined, "gh pr list --repo owner/repo --state open --limit 100 --author @me") ||
		!strings.Contains(joined, "gh issue list --repo owner/repo --state open --limit 100 --assignee @me") {
		t.Fatalf("probe commands did not preserve mine scope:\n%s", joined)
	}
}

func TestMutedProbeSkipsFullScanUntilMaterialDelta(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	cfg := config.Config{
		Path:     filepath.Join(root, "config.yaml"),
		Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Minute, UntrackedProbeInterval: time.Minute, GitHubAuthor: "@me", GitHubScope: config.GitHubScopeMine},
	}
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo", Base: "main", Remote: "origin"}
	tracker := tracking.Manager{Store: tracking.FileStore{}, Now: time.Now}
	initial := cachedRecord(repository.GitHub, 1, model.TrackingTracked).Snapshot
	muted, err := tracker.SetTracked(initial, []string{repository.GitHub}, false)
	if err != nil {
		t.Fatal(err)
	}
	baseline := ProbeResult{Combined: digest([]byte("same")), Local: digest([]byte("local")), Remote: digest([]byte("remote"))}
	if err := tracker.UpdateProbe(cfg.Path, repository.GitHub, baseline.Combined, baseline.Local, baseline.Remote, time.Now()); err != nil {
		t.Fatal(err)
	}
	record := cachedRecord(repository.GitHub, 1, model.TrackingUntracked)
	record.Snapshot = muted
	cache := Cache{Directory: paths.Projects}
	if err := cache.Write(record); err != nil {
		t.Fatal(err)
	}
	prober := &mutableProber{result: baseline}
	var scanCalls atomic.Int32
	engine := NewEngine(cfg, paths, cache, func(context.Context) ([]config.Repository, error) {
		return []config.Repository{repository}, nil
	}, func(_ context.Context, _ config.Repository, _ bool, stage func(string)) (model.Snapshot, error) {
		scanCalls.Add(1)
		stage("local")
		changed := initial
		changed.Lanes[0].Worktree.HeadOID = "new-head"
		return tracker.Reconcile(changed)
	}, prober, tracker)
	if _, err := engine.Refresh(context.Background(), "", true); err != nil {
		t.Fatal(err)
	}
	waitForRefresh(t, engine)
	if scanCalls.Load() != 0 || engine.Snapshot().Projects[0].TrackingState != model.TrackingUntracked {
		t.Fatalf("unchanged probe triggered full scan")
	}
	prober.set(ProbeResult{Combined: digest([]byte("changed")), Local: digest([]byte("new-local")), Remote: baseline.Remote})
	if _, err := engine.Refresh(context.Background(), "", true); err != nil {
		t.Fatal(err)
	}
	waitForRefresh(t, engine)
	if scanCalls.Load() != 1 || engine.Snapshot().Projects[0].TrackingState != model.TrackingTracked {
		t.Fatalf("material probe did not reactivate project")
	}
}

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

func TestLaunchAgentPlistEscapesPathsAndUsesSingleBinary(t *testing.T) {
	paths := testPaths(t.TempDir())
	plist := launchAgentPlist("/Applications/A&B/Beacon", paths)
	for _, expected := range []string{"com.jamesonstone.beacon.agent", "/Applications/A&amp;B/Beacon", "<string>agent</string><string>serve</string>", paths.StandardLog} {
		if !bytes.Contains([]byte(plist), []byte(expected)) {
			t.Fatalf("plist missing %q: %s", expected, plist)
		}
	}
}

func TestLifecycleInstallAndUninstallUseUserOnlyFiles(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("LaunchAgent lifecycle is supported on macOS only")
	}
	paths := testPaths(t.TempDir())
	runner := &lifecycleCommandRunner{}
	lifecycle := Lifecycle{Paths: paths, Runner: runner, Executable: "/Applications/Beacon & Co/Beacon"}
	if err := lifecycle.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(paths.LaunchAgent)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("LaunchAgent mode = %o", info.Mode().Perm())
	}
	contents, err := os.ReadFile(paths.LaunchAgent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "/Applications/Beacon &amp; Co/Beacon") || !strings.Contains(strings.Join(runner.commands, "\n"), "launchctl bootstrap") {
		t.Fatalf("plist=%s commands=%v", contents, runner.commands)
	}
	for _, path := range []string{paths.Socket, paths.PID} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := lifecycle.Uninstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.LaunchAgent, paths.Socket, paths.PID} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("lifecycle file still exists: %s", path)
		}
	}
}

func TestPIDLockRejectsDuplicateAgentAndRecoversAfterRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.pid")
	release, err := acquirePIDLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquirePIDLock(path); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("duplicate lock error = %v", err)
	}
	release()
	releaseAgain, err := acquirePIDLock(path)
	if err != nil {
		t.Fatal(err)
	}
	releaseAgain()
}

type mutableProber struct {
	mutex  sync.Mutex
	result ProbeResult
}

type countingProber struct{ calls atomic.Int32 }

func (p *countingProber) Probe(context.Context, config.Repository, string) (ProbeResult, error) {
	p.calls.Add(1)
	return ProbeResult{}, nil
}

type probeCommandRunner struct{ commands []string }

func (r *probeCommandRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.commands = append(r.commands, strings.TrimSpace(name+" "+strings.Join(args, " ")))
	switch fmt.Sprint(append([]string{name}, args...)) {
	case "[git rev-parse HEAD]":
		return []byte("head\n"), nil
	case "[git status --porcelain=v2 --branch -z]":
		return []byte("# branch.head main\x00"), nil
	case "[git for-each-ref --format=%(refname:short)%00%(objectname) refs/heads]":
		return []byte("main\x00head\n"), nil
	default:
		if name == "gh" {
			return []byte("[]"), nil
		}
		return nil, fmt.Errorf("unexpected command: %s %v", name, args)
	}
}

type lifecycleCommandRunner struct{ commands []string }

func (r *lifecycleCommandRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.commands = append(r.commands, name+" "+strings.Join(args, " "))
	return nil, nil
}

func (p *mutableProber) Probe(context.Context, config.Repository, string) (ProbeResult, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.result, nil
}

func (p *mutableProber) set(result ProbeResult) {
	p.mutex.Lock()
	p.result = result
	p.mutex.Unlock()
}

func cachedRecord(id string, revision uint64, state model.TrackingState) ProjectRecord {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	project := model.Project{Name: "repo", Path: "/repo", GitHub: id, Base: "main", Remote: "origin", TrackingState: state, LaneIDs: []string{"git:" + id + "@main"}, Errors: []model.ScanError{}, Warnings: []model.ScanError{}}
	lane := model.Lane{ID: project.LaneIDs[0], Repository: project.Name, GitHub: id, Base: "main", Branch: "main", Worktree: &model.Worktree{Path: "/repo", HeadOID: "head", StatusHash: "status", UpdatedAt: now}, Signals: model.Signals{Worktree: model.WorktreeClean, Publication: model.PublicationBase, Freshness: model.FreshnessCurrent}, NextAction: model.ActionNone, UpdatedAt: now, Reasons: []string{}, Warnings: []string{}, Blockers: []string{}}
	snapshot := model.Snapshot{SchemaVersion: model.SchemaVersion, GeneratedAt: now, ConfigPath: "/config.yaml", Projects: []model.Project{project}, Lanes: []model.Lane{lane}, Refresh: []model.Refresh{}, Errors: []model.ScanError{}, Warnings: []model.ScanError{}}
	return ProjectRecord{Version: CacheVersion, ProjectID: id, Revision: revision, Stage: "ready", UpdatedAt: now, Snapshot: snapshot}
}

func testPaths(root string) Paths {
	return Paths{Config: filepath.Join(root, "config.yaml"), State: filepath.Join(root, "state", "beacon", "tracking.json"), CacheRoot: filepath.Join(root, "cache"), Projects: filepath.Join(root, "cache", "projects"), Socket: filepath.Join(root, "cache", "agent.sock"), PID: filepath.Join(root, "cache", "agent.pid"), LaunchAgent: filepath.Join(root, "LaunchAgents", "agent.plist"), Logs: filepath.Join(root, "logs"), StandardLog: filepath.Join(root, "logs", "agent.log"), ErrorLog: filepath.Join(root, "logs", "agent-error.log")}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("file did not appear: %s", path)
}

func waitForRefresh(t *testing.T, engine *Engine) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if !engine.Status().Refreshing {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("refresh did not complete")
}
