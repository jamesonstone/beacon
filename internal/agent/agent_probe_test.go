package agent

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
)

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

func TestProbeManyUsesOneRemoteCollectionForEntireBatch(t *testing.T) {
	runner := &probeCommandRunner{}
	remote := &countingRemoteCollector{}
	repositories := make([]config.Repository, 80)
	for index := range repositories {
		repositories[index] = config.Repository{
			Name: fmt.Sprintf("repo-%02d", index), Path: "/repo",
			GitHub: fmt.Sprintf("owner/repo-%02d", index),
		}
	}

	results, failures := (Prober{Runner: runner, Remote: remote}).ProbeMany(
		context.Background(), repositories, "mine", "@me", 1,
	)
	if len(failures) != 0 || len(results) != len(repositories) || remote.calls != 1 || remote.size != len(repositories) {
		t.Fatalf("results=%d failures=%v calls=%d size=%d", len(results), failures, remote.calls, remote.size)
	}
	for _, result := range results {
		if result.Format != ProbeFormatBatch {
			t.Fatalf("probe format = %q", result.Format)
		}
	}
	for _, command := range runner.commands {
		if strings.HasPrefix(command, "gh ") {
			t.Fatalf("batch probe issued repository GitHub command: %s", command)
		}
	}
}

func TestCollectedBatchMigratesEightyMutedProbesWithoutFullScans(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	cfg := config.Config{
		Path: filepath.Join(root, "config.yaml"),
		Settings: config.Settings{
			MaxParallel: 4, TrackedRefreshInterval: time.Minute,
			UntrackedProbeInterval: time.Hour, GitHubAuthor: "@me", GitHubScope: config.GitHubScopeMine,
		},
	}
	cache := Cache{Directory: paths.Projects}
	repositories := make([]config.Repository, 80)
	records := make([]ProjectRecord, 0, len(repositories))
	for index := range repositories {
		id := fmt.Sprintf("owner/repo-%02d", index)
		repositories[index] = config.Repository{Name: fmt.Sprintf("repo-%02d", index), Path: "/repo", GitHub: id, Base: "main", Remote: "origin"}
		record := cachedRecord(id, 1, model.TrackingTracked)
		record.Snapshot.ConfigPath = cfg.Path
		record.Snapshot.Projects[0].Name = repositories[index].Name
		record.Snapshot.Projects[0].GitHub = id
		record.Snapshot.Lanes[0].GitHub = id
		if err := cache.Write(record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	tracker := tracking.Manager{Store: tracking.FileStore{}, Now: time.Now}
	assembled := Assemble(records, cfg.Path, paths.State, time.Now())
	if _, err := tracker.SetSelection(assembled, []string{}); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(cfg, paths, cache, func(context.Context) ([]config.Repository, error) {
		return repositories, nil
	}, nil, nil, tracker)
	var probeCalls atomic.Int32
	var scanCalls atomic.Int32
	engine.ProbeBatch = func(_ context.Context, values []config.Repository, _, _ string, _ int) (map[string]ProbeResult, map[string]error) {
		probeCalls.Add(1)
		results := make(map[string]ProbeResult, len(values))
		for _, repository := range values {
			local := digest([]byte("local-" + repository.GitHub))
			remote := digest([]byte("remote-" + repository.GitHub))
			results[repository.GitHub] = ProbeResult{
				Combined: digest([]byte(local), []byte(remote)), Local: local,
				Remote: remote, Format: ProbeFormatBatch,
			}
		}
		return results, map[string]error{}
	}
	engine.ScanBatch = func(context.Context, []config.Repository, bool, func(string, string)) (map[string]model.Snapshot, error) {
		scanCalls.Add(1)
		return nil, errors.New("unexpected full scan")
	}
	engine.Now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, err := engine.Refresh(context.Background(), "", false); err != nil {
		t.Fatal(err)
	}
	waitForRefreshWithin(t, engine, 10*time.Second)
	if probeCalls.Load() != 1 || scanCalls.Load() != 0 {
		t.Fatalf("probe calls=%d scan calls=%d", probeCalls.Load(), scanCalls.Load())
	}
	snapshot := engine.Snapshot()
	if snapshot.Summary.TrackedProjects != 0 || snapshot.Summary.UntrackedProjects != len(repositories) {
		t.Fatalf("tracking summary = %#v", snapshot.Summary)
	}
	state, err := (tracking.FileStore{}).Load(paths.State)
	if err != nil || len(state.Untracked) != len(repositories) {
		t.Fatalf("state projects=%d err=%v", len(state.Untracked), err)
	}
	for _, entry := range state.Untracked {
		if entry.ProbeFormat != ProbeFormatBatch || entry.ProbeBaseline == "" {
			t.Fatalf("probe migration failed: %#v", entry)
		}
	}
}

func TestCollectedBatchForceScansMutedProject(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	cfg := config.Config{
		Path: filepath.Join(root, "config.yaml"),
		Settings: config.Settings{
			MaxParallel: 1, GitHubAuthor: "@me", GitHubScope: config.GitHubScopeMine,
		},
	}
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo", Base: "main", Remote: "origin"}
	cache := Cache{Directory: paths.Projects}
	record := cachedRecord(repository.GitHub, 1, model.TrackingTracked)
	record.Snapshot.ConfigPath = cfg.Path
	if err := cache.Write(record); err != nil {
		t.Fatal(err)
	}
	tracker := tracking.Manager{Store: tracking.FileStore{}, Now: time.Now}
	if _, err := tracker.SetSelection(Assemble([]ProjectRecord{record}, cfg.Path, paths.State, time.Now()), []string{}); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(cfg, paths, cache, func(context.Context) ([]config.Repository, error) {
		return []config.Repository{repository}, nil
	}, nil, nil, tracker)
	var probeCalls atomic.Int32
	var scanCalls atomic.Int32
	engine.ProbeBatch = func(context.Context, []config.Repository, string, string, int) (map[string]ProbeResult, map[string]error) {
		probeCalls.Add(1)
		return nil, nil
	}
	engine.ScanBatch = func(_ context.Context, repositories []config.Repository, refresh bool, _ func(string, string)) (map[string]model.Snapshot, error) {
		scanCalls.Add(1)
		if !refresh || len(repositories) != 1 || repositories[0].GitHub != repository.GitHub {
			return nil, fmt.Errorf("forced repositories=%#v refresh=%v", repositories, refresh)
		}
		snapshot := record.Snapshot
		snapshot.Projects[0].TrackingState = model.TrackingUntracked
		return map[string]model.Snapshot{repository.GitHub: snapshot}, nil
	}
	if _, err := engine.Refresh(context.Background(), "", true); err != nil {
		t.Fatal(err)
	}
	waitForRefresh(t, engine)
	if probeCalls.Load() != 0 || scanCalls.Load() != 1 {
		t.Fatalf("probe calls=%d scan calls=%d", probeCalls.Load(), scanCalls.Load())
	}
}

func TestScheduleSkipsDiscoveryWhenCachedProjectsAreNotDue(t *testing.T) {
	now := time.Date(2026, 7, 12, 16, 0, 0, 0, time.UTC)
	record := cachedRecord("owner/repo", 1, model.TrackingUntracked)
	record.UpdatedAt = now
	record.LastProbeAt = now
	var discoveries atomic.Int32
	engine := &Engine{
		Config: config.Config{Settings: config.Settings{
			TrackedRefreshInterval: 10 * time.Millisecond,
			UntrackedProbeInterval: time.Hour,
		}},
		Now:       func() time.Time { return now },
		records:   map[string]ProjectRecord{record.ProjectID: record},
		revisions: map[string]uint64{record.ProjectID: record.Revision},
		stages:    map[string]string{record.ProjectID: "cached"},
		hub:       newEventHub(),
		Repositories: func(context.Context) ([]config.Repository, error) {
			discoveries.Add(1)
			return nil, nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	engine.RunSchedule(ctx)
	if discoveries.Load() != 0 {
		t.Fatalf("repository discoveries = %d, want 0", discoveries.Load())
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
	baseline := ProbeResult{Combined: digest([]byte("same")), Local: digest([]byte("local")), Remote: digest([]byte("remote")), Format: ProbeFormatRepository}
	if err := tracker.UpdateProbe(cfg.Path, repository.GitHub, baseline.Format, baseline.Combined, baseline.Local, baseline.Remote, time.Now()); err != nil {
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
	prober.set(ProbeResult{Combined: digest([]byte("changed")), Local: digest([]byte("new-local")), Remote: baseline.Remote, Format: ProbeFormatRepository})
	if _, err := engine.Refresh(context.Background(), "", true); err != nil {
		t.Fatal(err)
	}
	waitForRefresh(t, engine)
	updated := engine.Snapshot().Projects[0]
	if scanCalls.Load() != 1 || updated.TrackingState != model.TrackingUntracked || updated.FollowState != model.FollowRecent || updated.ActivityReason != "new local changes" {
		t.Fatalf("material probe did not preserve and flag outside activity: %#v", updated)
	}
}
