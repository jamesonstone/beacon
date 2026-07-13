package scan

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
	"github.com/jamesonstone/beacon/internal/githubapi"
	"github.com/jamesonstone/beacon/internal/githubscan"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/policy"
	"github.com/jamesonstone/beacon/internal/progress"
)

type Scanner struct {
	Git       GitScanner
	GitHub    GitHubClient
	Discovery RepositoryDiscoverer
	Now       func() time.Time
}

type GitScanner interface {
	Scan(context.Context, config.Repository, bool, time.Duration) gitscan.Result
}

type GitHubClient interface {
	Collect(context.Context, []config.Repository, string, string, int) model.RemoteCollection
}

type RepositoryDiscoverer interface {
	Discover(context.Context, []config.Source) discovery.Result
}

type commonDirectoryResolver interface {
	CommonDirectory(context.Context, string) (string, error)
}

type repositoryResult struct {
	index    int
	project  model.Project
	lanes    []model.Lane
	refresh  model.Refresh
	errors   []model.ScanError
	warnings []model.ScanError
}

func (s Scanner) Scan(ctx context.Context, cfg config.Config, repositoryName string, refresh bool) (model.Snapshot, error) {
	if s.Now == nil {
		s.Now = time.Now
	}
	repositories, discoveryErrors, discoveryWarnings := s.Repositories(ctx, cfg)
	if repositoryName != "" {
		filtered := repositories[:0]
		for _, repository := range repositories {
			if repository.Name == repositoryName {
				filtered = append(filtered, repository)
				break
			}
		}
		repositories = filtered
		if len(repositories) == 0 {
			return model.Snapshot{}, errors.New("configured or discovered repository not found: " + repositoryName)
		}
	}
	if len(repositories) == 0 {
		return model.Snapshot{}, errors.New("configuration did not resolve to any accessible GitHub repositories")
	}

	remoteContext := githubapi.WithFreshEvidence(githubscan.WithInactivePullRequests(ctx))
	remote := s.GitHub.Collect(remoteContext, repositories, string(cfg.Settings.GitHubScope), cfg.Settings.GitHubAuthor, cfg.Settings.MaxParallel)
	results := make(chan repositoryResult, len(repositories))
	semaphore := make(chan struct{}, cfg.Settings.MaxParallel)
	var waitGroup sync.WaitGroup
	for index, repository := range repositories {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results <- s.scanRepository(ctx, cfg, index, repository, remote.Repositories[repository.GitHub], refresh)
		}()
	}
	waitGroup.Wait()
	close(results)

	ordered := make([]repositoryResult, len(repositories))
	for result := range results {
		ordered[result.index] = result
	}
	snapshot := model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		GeneratedAt:   s.Now(),
		ConfigPath:    cfg.Path,
		Refresh:       []model.Refresh{},
		Groups: model.Groups{
			Ready: []string{}, Action: []string{}, Waiting: []string{}, Idle: []string{}, Untracked: []string{},
		},
		Projects: []model.Project{},
		Lanes:    []model.Lane{},
		Errors:   append(discoveryErrors, remote.Errors...),
		Warnings: append(discoveryWarnings, remote.Warnings...),
	}
	for _, result := range ordered {
		snapshot.Projects = append(snapshot.Projects, result.project)
		snapshot.Lanes = append(snapshot.Lanes, result.lanes...)
		snapshot.Refresh = append(snapshot.Refresh, result.refresh)
		snapshot.Errors = append(snapshot.Errors, result.errors...)
		snapshot.Warnings = append(snapshot.Warnings, result.warnings...)
	}
	Finalize(&snapshot)
	return snapshot, nil
}

func (s Scanner) ScanOne(ctx context.Context, cfg config.Config, repository config.Repository, refresh bool, stage func(string)) (model.Snapshot, error) {
	if s.Now == nil {
		s.Now = time.Now
	}
	local := s.Git.Scan(ctx, repository, refresh, cfg.Settings.RemoteRefreshInterval)
	if stage != nil {
		stage("local")
	}
	remote := s.GitHub.Collect(ctx, []config.Repository{repository}, string(cfg.Settings.GitHubScope), cfg.Settings.GitHubAuthor, 1)
	if stage != nil {
		stage("github")
	}
	result := s.buildRepository(cfg, 0, repository, local, remote.Repositories[repository.GitHub])
	snapshot := model.Snapshot{
		SchemaVersion: model.SchemaVersion, GeneratedAt: s.Now(), ConfigPath: cfg.Path,
		Refresh: []model.Refresh{result.refresh}, Projects: []model.Project{result.project},
		Lanes: result.lanes, Errors: append(append([]model.ScanError{}, remote.Errors...), result.errors...),
		Warnings: append(append([]model.ScanError{}, remote.Warnings...), result.warnings...),
	}
	Finalize(&snapshot)
	return snapshot, nil
}

func orderProjectLanes(projects []model.Project, lanes []model.Lane) {
	byGitHub := make(map[string]int, len(projects))
	for index := range projects {
		projects[index].LaneIDs = []string{}
		byGitHub[projects[index].GitHub] = index
	}
	for _, lane := range lanes {
		if index, ok := byGitHub[lane.GitHub]; ok {
			projects[index].LaneIDs = append(projects[index].LaneIDs, lane.ID)
		}
	}
}

func (s Scanner) scanRepository(ctx context.Context, cfg config.Config, index int, repository config.Repository, remote model.RemoteEvidence, refresh bool) repositoryResult {
	local := s.Git.Scan(ctx, repository, refresh, cfg.Settings.RemoteRefreshInterval)
	return s.buildRepository(cfg, index, repository, local, remote)
}

func (s Scanner) buildRepository(cfg config.Config, index int, repository config.Repository, local gitscan.Result, remote model.RemoteEvidence) repositoryResult {
	progressResult := progress.Load(repository.Path)
	projectProgress, progressByIssue := correlateProgress(repository, progressResult)
	errors := append([]model.ScanError{}, local.Errors...)
	errors = append(errors, remote.Errors...)
	warnings := append([]model.ScanError{}, local.Warnings...)
	warnings = append(warnings, remote.Warnings...)
	for _, diagnostic := range progressResult.Diagnostics {
		warnings = append(warnings, model.ScanError{
			Repository: repository.Name,
			Stage:      "progress",
			Message:    diagnostic.Path + ": " + diagnostic.Message,
		})
	}
	lanes := policy.Build(repository, local.Lanes, remote.PullRequests, remote.Issues, progressByIssue, cfg.Settings.StaleAfter, s.Now())
	laneIDs := make([]string, 0, len(lanes))
	for _, lane := range lanes {
		laneIDs = append(laneIDs, lane.ID)
	}
	projectErrors := append([]model.ScanError{}, errors...)
	projectWarnings := append([]model.ScanError{}, warnings...)
	return repositoryResult{
		index: index, lanes: lanes, refresh: local.Refresh, errors: errors, warnings: warnings,
		project: model.Project{
			Name: repository.Name, Path: repository.Path, GitHub: repository.GitHub,
			Base: repository.Base, Remote: repository.Remote, Progress: projectProgress,
			LaneIDs: laneIDs, Errors: projectErrors, Warnings: projectWarnings,
		},
	}
}
