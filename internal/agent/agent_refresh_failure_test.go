package agent

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
)

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
