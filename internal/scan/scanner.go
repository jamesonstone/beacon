package scan

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
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

	remote := s.GitHub.Collect(githubscan.WithInactivePullRequests(ctx), repositories, string(cfg.Settings.GitHubScope), cfg.Settings.GitHubAuthor, cfg.Settings.MaxParallel)
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

func Finalize(snapshot *model.Snapshot) {
	snapshot.Groups = model.Groups{Ready: []string{}, Action: []string{}, Waiting: []string{}, Idle: []string{}, Untracked: []string{}}
	snapshot.WorkingSet = model.WorkingSet{Active: []string{}, Waiting: []string{}, Recent: []string{}, Parked: []string{}}
	snapshot.Summary = model.Summary{}
	orderLanes(snapshot.Lanes)
	orderProjectLanes(snapshot.Projects, snapshot.Lanes)
	group(snapshot)
}

func correlateProgress(repository config.Repository, result progress.Result) (*model.Progress, map[int]model.Progress) {
	byIssue := make(map[int]model.Progress)
	prefix := fmt.Sprintf("https://github.com/%s/issues/", repository.GitHub)
	for _, feature := range result.Features {
		for _, issueURL := range feature.IssueURLs {
			if !strings.HasPrefix(issueURL, prefix) {
				continue
			}
			number, err := strconv.Atoi(strings.TrimPrefix(issueURL, prefix))
			if err == nil && number > 0 {
				// Features are ordered by numeric ID, so the newest exact
				// reference deterministically wins shared delivery issues.
				byIssue[number] = progressModel(feature)
			}
		}
	}
	if result.Selected == nil {
		return nil, byIssue
	}
	selected := progressModel(*result.Selected)
	return &selected, byIssue
}

func progressModel(feature progress.Feature) model.Progress {
	summary := feature.Summary
	if summary == "" {
		summary = feature.OpenItems
	}
	if summary == "" {
		summary = feature.Intent
	}
	return model.Progress{
		Source: "kit", FeatureID: feature.ID, Feature: feature.Slug,
		Phase: feature.Phase, Summary: summary, Path: feature.SpecPath,
	}
}

func orderLanes(lanes []model.Lane) {
	sort.SliceStable(lanes, func(left, right int) bool {
		leftLane, rightLane := lanes[left], lanes[right]
		if leftLane.ReviewReady != rightLane.ReviewReady {
			return leftLane.ReviewReady
		}
		if !leftLane.ReviewReady {
			leftPriority, rightPriority := actionPriority(leftLane.NextAction), actionPriority(rightLane.NextAction)
			if leftPriority != rightPriority {
				return leftPriority < rightPriority
			}
		}
		if !leftLane.UpdatedAt.Equal(rightLane.UpdatedAt) {
			return leftLane.UpdatedAt.Before(rightLane.UpdatedAt)
		}
		if leftLane.Repository != rightLane.Repository {
			return leftLane.Repository < rightLane.Repository
		}
		if leftLane.Branch != rightLane.Branch {
			return leftLane.Branch < rightLane.Branch
		}
		return leftLane.ID < rightLane.ID
	})
}

func actionPriority(action model.Action) int {
	switch action {
	case model.ActionResolveConflict:
		return 1
	case model.ActionFixCI:
		return 2
	case model.ActionAddressReview:
		return 3
	case model.ActionInspectLocal:
		return 4
	case model.ActionPushBranch:
		return 5
	case model.ActionRefreshState:
		return 6
	case model.ActionCreatePR:
		return 7
	case model.ActionMarkReady:
		return 8
	case model.ActionWaitForCI:
		return 9
	case model.ActionMergePR:
		return 10
	case model.ActionManualTestMerge:
		return 11
	case model.ActionReviewPR:
		return 12
	case model.ActionResumeOrClose:
		return 13
	case model.ActionStartIssue:
		return 14
	default:
		return 15
	}
}

func group(snapshot *model.Snapshot) {
	snapshot.Summary.Projects = len(snapshot.Projects)
	for _, lane := range snapshot.Lanes {
		snapshot.Summary.Total++
		if lane.Issue != nil {
			snapshot.Summary.OpenIssues++
		}
		if lane.PullRequest != nil {
			snapshot.Summary.UnresolvedFeedback += lane.PullRequest.Feedback.UnresolvedThreads
		}
		switch {
		case lane.ReviewReady:
			snapshot.Groups.Ready = append(snapshot.Groups.Ready, lane.ID)
			snapshot.Summary.ReviewReady++
		case lane.NextAction != model.ActionNone:
			snapshot.Groups.Action = append(snapshot.Groups.Action, lane.ID)
			snapshot.Summary.NeedsAction++
		case lane.PullRequest != nil || lane.Issue != nil || lane.Branch != lane.Base:
			snapshot.Groups.Waiting = append(snapshot.Groups.Waiting, lane.ID)
			snapshot.Summary.Waiting++
		default:
			snapshot.Groups.Idle = append(snapshot.Groups.Idle, lane.ID)
			snapshot.Summary.Idle++
		}
	}
	snapshot.Summary.Errors = len(snapshot.Errors)
	snapshot.Summary.Warnings = len(snapshot.Warnings)
}
