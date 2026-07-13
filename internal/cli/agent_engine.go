package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/githubapi"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
	"github.com/jamesonstone/beacon/internal/workset"
)

func (a App) agentConfig(path string) (config.Config, agent.Paths, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, agent.Paths{}, err
	}
	paths, err := agent.ResolvePaths(cfg.Path)
	return cfg, paths, err
}

func (a App) newAgentEngine(ctx context.Context, path string) (*agent.Engine, agent.Paths, error) {
	cfg, paths, err := a.agentConfig(path)
	if err != nil {
		return nil, agent.Paths{}, err
	}
	if err := paths.EnsureRuntime(); err != nil {
		return nil, agent.Paths{}, err
	}
	githubRunner := githubapi.NewRunnerWithOptions(a.Runner, githubapi.Options{
		CacheTTL: cfg.Settings.RemoteRefreshInterval, CacheDirectory: filepath.Join(paths.CacheRoot, "github"),
	})
	scanner := a.scannerComponentsWithRunner(githubRunner)
	tracker := tracking.Manager{
		Store: tracking.FileStore{}, Now: time.Now,
		RecentWindow: cfg.Settings.StaleAfter,
	}
	repositories := func(repositoryContext context.Context) ([]config.Repository, error) {
		values, scanErrors, _ := scanner.Repositories(repositoryContext, cfg)
		if len(values) == 0 {
			if len(scanErrors) > 0 {
				return nil, fmt.Errorf("discover repositories: %s", scanErrors[0].Message)
			}
			return nil, errors.New("configuration resolved no repositories")
		}
		inventory := make([]model.Project, 0, len(values))
		for _, repository := range values {
			inventory = append(inventory, model.Project{
				Name: repository.Name, Path: repository.Path, GitHub: repository.GitHub,
				Base: repository.Base, Remote: repository.Remote,
			})
		}
		if err := tracker.InitializeInventory(cfg.Path, inventory); err != nil {
			return nil, fmt.Errorf("initialize project following inventory: %w", err)
		}
		return values, nil
	}
	projectScanner := func(scanContext context.Context, repository config.Repository, refresh bool, stage func(string)) (model.Snapshot, error) {
		snapshot, scanErr := scanner.ScanOne(scanContext, cfg, repository, refresh, stage)
		if scanErr != nil {
			return model.Snapshot{}, scanErr
		}
		return tracker.ReconcilePartial(snapshot)
	}
	cache := agent.Cache{Directory: paths.Projects, Now: time.Now}
	prober := agent.Prober{Runner: githubRunner, Remote: scanner.GitHub}
	engine := agent.NewEngine(cfg, paths, cache, repositories, projectScanner, prober, tracker)
	workingSet := workset.Manager{Store: workset.FileStore{}, Now: time.Now}
	engine.WorkingSet = &workingSet
	engine.ScanBatch = func(
		scanContext context.Context,
		repositories []config.Repository,
		refresh bool,
		stage func(string, string),
	) (map[string]model.Snapshot, error) {
		snapshots, scanErr := scanner.ScanMany(scanContext, cfg, repositories, refresh, stage)
		if scanErr != nil {
			return nil, scanErr
		}
		for projectID, snapshot := range snapshots {
			reconciled, reconcileErr := tracker.ReconcilePartial(snapshot)
			if reconcileErr != nil {
				return nil, reconcileErr
			}
			snapshots[projectID] = reconciled
		}
		return snapshots, nil
	}
	engine.ProbeBatch = prober.ProbeMany
	_ = ctx
	return engine, paths, nil
}
