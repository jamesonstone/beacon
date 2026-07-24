package workscan

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/policy"
)

type GitScanner interface {
	Scan(context.Context, config.Repository, bool, time.Duration) gitscan.Result
}

type PullRequestClient interface {
	ListOpen(context.Context, string, string) ([]model.PullRequest, error)
}

type RepositoryDiscoverer interface {
	Discover(context.Context, []config.Source) discovery.Result
}

type Scanner struct {
	Git       GitScanner
	GitHub    PullRequestClient
	Discovery RepositoryDiscoverer
	Now       func() time.Time
}

type repositoryResult struct {
	index    int
	items    []model.WorkItem
	active   bool
	unknown  bool
	errors   []model.ScanError
	warnings []model.ScanError
}

func (s Scanner) Scan(ctx context.Context, cfg config.Config, refresh, includeIdle bool) (model.WorkScan, error) {
	if s.Git == nil || s.GitHub == nil || s.Discovery == nil {
		return model.WorkScan{}, errors.New("work scanner is not fully configured")
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	discovered := s.Discovery.Discover(ctx, cfg.Sources)
	repositories := selectedRepositories(cfg.Repositories, discovered.Repositories)

	result := model.WorkScan{
		SchemaVersion: model.WorkScanSchemaVersion,
		GeneratedAt:   s.Now(),
		Items:         []model.WorkItem{},
		Errors:        []model.ScanError{},
		Warnings:      discoveryWarnings(discovered.Warnings),
	}
	result.Summary.Projects = len(repositories)
	if len(repositories) == 0 {
		result.Summary.Warnings = len(result.Warnings)
		return result, nil
	}

	results := make(chan repositoryResult, len(repositories))
	semaphore := make(chan struct{}, max(1, cfg.Settings.MaxParallel))
	var waitGroup sync.WaitGroup
	for index, repository := range repositories {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results <- s.scanRepository(ctx, cfg, index, repository, refresh, includeIdle)
		}()
	}
	waitGroup.Wait()
	close(results)

	ordered := make([]repositoryResult, len(repositories))
	for repository := range results {
		ordered[repository.index] = repository
	}
	for _, repository := range ordered {
		result.Items = append(result.Items, repository.items...)
		result.Errors = append(result.Errors, repository.errors...)
		result.Warnings = append(result.Warnings, repository.warnings...)
		if repository.active {
			result.Summary.ActiveProjects++
		} else if repository.unknown {
			result.Summary.UnknownProjects++
		} else {
			result.Summary.IdleProjects++
		}
	}
	sortWorkItems(result.Items)
	sortDiagnostics(result.Errors)
	sortDiagnostics(result.Warnings)
	result.Summary.WorkItems = countActiveItems(result.Items)
	result.Summary.Errors = len(result.Errors)
	result.Summary.Warnings = len(result.Warnings)
	return result, nil
}

func selectedRepositories(configured, discovered []config.Repository) []config.Repository {
	repositories := make([]config.Repository, 0, len(configured)+len(discovered))
	seenGitHub := make(map[string]struct{}, len(configured)+len(discovered))
	for _, candidates := range [][]config.Repository{configured, discovered} {
		for _, repository := range candidates {
			if _, exists := seenGitHub[repository.GitHub]; exists {
				continue
			}
			seenGitHub[repository.GitHub] = struct{}{}
			repositories = append(repositories, repository)
		}
	}
	sort.Slice(repositories, func(i, j int) bool {
		if repositories[i].Path != repositories[j].Path {
			return repositories[i].Path < repositories[j].Path
		}
		return repositories[i].GitHub < repositories[j].GitHub
	})
	return repositories
}

func (s Scanner) scanRepository(
	ctx context.Context,
	cfg config.Config,
	index int,
	repository config.Repository,
	refresh bool,
	includeIdle bool,
) repositoryResult {
	local := s.Git.Scan(ctx, repository, refresh, cfg.Settings.RemoteRefreshInterval)
	pullRequests, err := s.GitHub.ListOpen(ctx, repository.GitHub, cfg.Settings.GitHubAuthor)
	errors := append([]model.ScanError{}, local.Errors...)
	if err != nil {
		errors = append(errors, model.ScanError{
			Repository: repository.Name,
			Stage:      "github-pull-requests",
			Message:    err.Error(),
		})
	}
	lanes := policy.Build(repository, local.Lanes, pullRequests, nil, nil, cfg.Settings.StaleAfter, s.Now())
	items := make([]model.WorkItem, 0, len(lanes))
	for _, lane := range lanes {
		if inProgress(lane) {
			items = append(items, workItem(repository, lane))
		}
	}
	active := len(items) > 0
	unknown := !active && len(errors) > 0
	if !active && !unknown && includeIdle {
		items = append(items, model.WorkItem{
			Repository: repository.Name, GitHub: repository.GitHub,
			RepositoryPath: repository.Path, Branch: repository.Base,
			Base: repository.Base, State: model.WorkIdle,
		})
	}
	return repositoryResult{
		index: index, items: items, active: active, unknown: unknown, errors: errors,
		warnings: append([]model.ScanError{}, local.Warnings...),
	}
}

func discoveryWarnings(warnings []discovery.Warning) []model.ScanError {
	result := make([]model.ScanError, 0, len(warnings))
	for _, warning := range warnings {
		result = append(result, model.ScanError{
			Stage: "discovery-" + warning.Stage, Message: warning.Path + ": " + warning.Message,
		})
	}
	return result
}

func sortDiagnostics(diagnostics []model.ScanError) {
	sort.Slice(diagnostics, func(left, right int) bool {
		if diagnostics[left].Repository != diagnostics[right].Repository {
			return diagnostics[left].Repository < diagnostics[right].Repository
		}
		if diagnostics[left].Stage != diagnostics[right].Stage {
			return diagnostics[left].Stage < diagnostics[right].Stage
		}
		return diagnostics[left].Message < diagnostics[right].Message
	})
}

func countActiveItems(items []model.WorkItem) int {
	count := 0
	for _, item := range items {
		if item.State != model.WorkIdle {
			count++
		}
	}
	return count
}
