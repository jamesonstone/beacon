package scan

import (
	"context"
	"errors"
	"sync"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

// ScanMany collects remote evidence once for the full repository batch and
// returns one independently finalized snapshot per repository. This keeps the
// background agent incremental without paying per-repository GitHub queries.
func (s Scanner) ScanMany(
	ctx context.Context,
	cfg config.Config,
	repositories []config.Repository,
	refresh bool,
	stage func(string, string),
) (map[string]model.Snapshot, error) {
	if len(repositories) == 0 {
		return map[string]model.Snapshot{}, nil
	}
	if s.Git == nil || s.GitHub == nil {
		return nil, errors.New("batch scanner requires Git and GitHub collectors")
	}
	if s.Now == nil {
		return nil, errors.New("batch scanner clock is required")
	}

	remote := s.GitHub.Collect(
		ctx,
		repositories,
		string(cfg.Settings.GitHubScope),
		cfg.Settings.GitHubAuthor,
		cfg.Settings.MaxParallel,
	)
	for _, repository := range repositories {
		if stage != nil {
			stage(repository.GitHub, "github")
		}
	}

	type batchResult struct {
		repository config.Repository
		result     repositoryResult
	}
	results := make(chan batchResult, len(repositories))
	semaphore := make(chan struct{}, max(1, cfg.Settings.MaxParallel))
	var waitGroup sync.WaitGroup
	for index, repository := range repositories {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			local := s.Git.Scan(ctx, repository, refresh, cfg.Settings.RemoteRefreshInterval)
			if stage != nil {
				stage(repository.GitHub, "local")
			}
			results <- batchResult{
				repository: repository,
				result:     s.buildRepository(cfg, index, repository, local, remote.Repositories[repository.GitHub]),
			}
		}()
	}
	waitGroup.Wait()
	close(results)

	snapshots := make(map[string]model.Snapshot, len(repositories))
	for value := range results {
		result := value.result
		snapshot := model.Snapshot{
			SchemaVersion: model.SchemaVersion,
			GeneratedAt:   s.Now(),
			ConfigPath:    cfg.Path,
			Refresh:       []model.Refresh{result.refresh},
			Projects:      []model.Project{result.project},
			Lanes:         result.lanes,
			Errors:        append(append([]model.ScanError{}, remote.Errors...), result.errors...),
			Warnings:      append(append([]model.ScanError{}, remote.Warnings...), result.warnings...),
		}
		Finalize(&snapshot)
		snapshots[value.repository.GitHub] = snapshot
	}
	return snapshots, nil
}
