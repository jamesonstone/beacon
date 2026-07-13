package scan

import (
	"context"
	"sort"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
	"github.com/jamesonstone/beacon/internal/model"
)

func (s Scanner) Repositories(ctx context.Context, cfg config.Config) ([]config.Repository, []model.ScanError, []model.ScanError) {
	discovered := discovery.Result{Repositories: []config.Repository{}, Warnings: []discovery.Warning{}}
	if len(cfg.Sources) > 0 {
		if s.Discovery == nil {
			return nil, []model.ScanError{{Stage: "discovery", Message: "repository discovery is not configured"}}, nil
		}
		discovered = s.Discovery.Discover(ctx, cfg.Sources)
	}

	// Source discoveries are the baseline. Explicit entries replace discoveries
	// for the same GitHub repository or canonical repository path.
	byGitHub := make(map[string]config.Repository, len(discovered.Repositories)+len(cfg.Repositories))
	pathOwner := make(map[string]string, len(discovered.Repositories)+len(cfg.Repositories))
	commonOwner := make(map[string]string, len(discovered.Repositories)+len(cfg.Repositories))
	errors := make([]model.ScanError, 0, len(discovered.Warnings)+len(cfg.Repositories))
	warnings := make([]model.ScanError, 0, len(discovered.Warnings))
	for _, repository := range discovered.Repositories {
		byGitHub[repository.GitHub] = repository
		if repository.Path != "" {
			pathOwner[repository.Path] = repository.GitHub
		}
		if repository.CommonDir != "" {
			commonOwner[repository.CommonDir] = repository.GitHub
		}
	}
	for _, repository := range cfg.Repositories {
		commonDir := repository.CommonDir
		if commonDir == "" {
			if resolver, ok := s.Discovery.(commonDirectoryResolver); ok {
				resolved, err := resolver.CommonDirectory(ctx, repository.Path)
				if err != nil {
					errors = append(errors, model.ScanError{Repository: repository.Name, Stage: "discovery-common-dir", Message: err.Error()})
				} else {
					commonDir = resolved
					repository.CommonDir = resolved
				}
			}
		}
		if previous, ok := commonOwner[commonDir]; commonDir != "" && ok && previous != repository.GitHub {
			delete(byGitHub, previous)
		}
		if repository.Path != "" {
			if previous, ok := pathOwner[repository.Path]; ok && previous != repository.GitHub {
				delete(byGitHub, previous)
			}
			pathOwner[repository.Path] = repository.GitHub
		}
		byGitHub[repository.GitHub] = repository
		if commonDir != "" {
			commonOwner[commonDir] = repository.GitHub
		}
	}
	repositories := make([]config.Repository, 0, len(byGitHub))
	for _, repository := range byGitHub {
		repositories = append(repositories, repository)
	}
	sort.Slice(repositories, func(i, j int) bool {
		if repositories[i].Name != repositories[j].Name {
			return repositories[i].Name < repositories[j].Name
		}
		if repositories[i].GitHub != repositories[j].GitHub {
			return repositories[i].GitHub < repositories[j].GitHub
		}
		return repositories[i].Path < repositories[j].Path
	})
	for _, warning := range discovered.Warnings {
		warnings = append(warnings, model.ScanError{Stage: "discovery-" + warning.Stage, Message: warning.Path + ": " + warning.Message})
	}
	return repositories, errors, warnings
}
