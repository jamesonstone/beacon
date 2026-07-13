package agent

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/githubscan"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
)

func TestCollectedBatchForceOnlyEnrichesInactivePullRequestsForFollowedProjects(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	cfg := config.Config{
		Path: paths.Config,
		Settings: config.Settings{
			MaxParallel: 2, GitHubAuthor: "@me", GitHubScope: config.GitHubScopeMine,
		},
	}
	repositories := []config.Repository{
		{Name: "followed", Path: root, GitHub: "owner/followed", Base: "main", Remote: "origin"},
		{Name: "quiet", Path: root, GitHub: "owner/quiet", Base: "main", Remote: "origin"},
	}
	cache := Cache{Directory: paths.Projects}
	records := make([]ProjectRecord, 0, len(repositories))
	for _, repository := range repositories {
		record := cachedRecord(repository.GitHub, 1, model.TrackingTracked)
		record.Snapshot.ConfigPath = cfg.Path
		record.Snapshot.Projects[0].Name = repository.Name
		record.Snapshot.Lanes[0].Repository = repository.Name
		if err := cache.Write(record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	tracker := tracking.Manager{Store: tracking.FileStore{}, Now: time.Now}
	if _, err := tracker.SetSelection(Assemble(records, cfg.Path, paths.State, time.Now()), []string{"owner/followed"}); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(cfg, paths, cache, func(context.Context) ([]config.Repository, error) {
		return repositories, nil
	}, nil, nil, tracker)
	engine.ProbeBatch = func(context.Context, []config.Repository, string, string, int) (map[string]ProbeResult, map[string]error) {
		return nil, nil
	}
	engine.ScanBatch = func(ctx context.Context, values []config.Repository, refresh bool, _ func(string, string)) (map[string]model.Snapshot, error) {
		if !refresh || len(values) != 2 {
			return nil, fmt.Errorf("forced repositories=%#v refresh=%v", values, refresh)
		}
		if !githubscan.IncludeInactivePullRequestsFor(ctx, "owner/followed") || githubscan.IncludeInactivePullRequestsFor(ctx, "owner/quiet") {
			return nil, errors.New("inactive PR scope did not match project following")
		}
		return map[string]model.Snapshot{
			"owner/followed": records[0].Snapshot,
			"owner/quiet":    records[1].Snapshot,
		}, nil
	}
	if _, err := engine.Refresh(context.Background(), "", true); err != nil {
		t.Fatal(err)
	}
	waitForRefresh(t, engine)
	if snapshot := engine.Snapshot(); len(snapshot.Errors) != 0 {
		t.Fatalf("snapshot errors = %#v", snapshot.Errors)
	}
}
