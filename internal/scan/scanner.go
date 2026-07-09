package scan

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/policy"
)

type Scanner struct {
	Git    GitScanner
	GitHub GitHubClient
	Now    func() time.Time
}

type GitScanner interface {
	Scan(context.Context, config.Repository, bool, time.Duration) gitscan.Result
}

type GitHubClient interface {
	ListOpen(context.Context, string, string) ([]model.PullRequest, error)
}

type repositoryResult struct {
	index   int
	lanes   []model.Lane
	refresh model.Refresh
	errors  []model.ScanError
}

func (s Scanner) Scan(ctx context.Context, cfg config.Config, repositoryName string, refresh bool) (model.Snapshot, error) {
	if s.Now == nil {
		s.Now = time.Now
	}
	repositories := cfg.Repositories
	if repositoryName != "" {
		repositories = nil
		for _, repository := range cfg.Repositories {
			if repository.Name == repositoryName {
				repositories = append(repositories, repository)
				break
			}
		}
		if len(repositories) == 0 {
			return model.Snapshot{}, errors.New("configured repository not found: " + repositoryName)
		}
	}

	results := make(chan repositoryResult, len(repositories))
	semaphore := make(chan struct{}, cfg.Settings.MaxParallel)
	var waitGroup sync.WaitGroup
	for index, repository := range repositories {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results <- s.scanRepository(ctx, cfg, index, repository, refresh)
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
			Ready: []string{}, Action: []string{}, Waiting: []string{}, Idle: []string{},
		},
		Lanes:  []model.Lane{},
		Errors: []model.ScanError{},
	}
	for _, result := range ordered {
		snapshot.Lanes = append(snapshot.Lanes, result.lanes...)
		snapshot.Refresh = append(snapshot.Refresh, result.refresh)
		snapshot.Errors = append(snapshot.Errors, result.errors...)
	}
	orderLanes(snapshot.Lanes)
	group(&snapshot)
	return snapshot, nil
}

func (s Scanner) scanRepository(ctx context.Context, cfg config.Config, index int, repository config.Repository, refresh bool) repositoryResult {
	local := s.Git.Scan(ctx, repository, refresh, cfg.Settings.RemoteRefreshInterval)
	result := repositoryResult{index: index, refresh: local.Refresh, errors: local.Errors}
	pullRequests, err := s.GitHub.ListOpen(ctx, repository.GitHub, cfg.Settings.GitHubAuthor)
	if err != nil {
		result.errors = append(result.errors, model.ScanError{Repository: repository.Name, Stage: "github", Message: err.Error()})
	}
	result.lanes = policy.Build(repository, local.Lanes, pullRequests, cfg.Settings.StaleAfter, s.Now())
	return result
}

func orderLanes(lanes []model.Lane) {
	sort.SliceStable(lanes, func(left, right int) bool {
		leftLane, rightLane := lanes[left], lanes[right]
		if leftLane.ReviewReady != rightLane.ReviewReady {
			return leftLane.ReviewReady
		}
		leftPriority, rightPriority := actionPriority(leftLane.NextAction), actionPriority(rightLane.NextAction)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if !leftLane.UpdatedAt.Equal(rightLane.UpdatedAt) {
			return leftLane.UpdatedAt.Before(rightLane.UpdatedAt)
		}
		if leftLane.Repository != rightLane.Repository {
			return leftLane.Repository < rightLane.Repository
		}
		return leftLane.Branch < rightLane.Branch
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
	case model.ActionReviewPR:
		return 9
	case model.ActionResumeOrClose:
		return 10
	default:
		return 11
	}
}

func group(snapshot *model.Snapshot) {
	for _, lane := range snapshot.Lanes {
		snapshot.Summary.Total++
		switch {
		case lane.ReviewReady:
			snapshot.Groups.Ready = append(snapshot.Groups.Ready, lane.ID)
			snapshot.Summary.ReviewReady++
		case lane.NextAction != model.ActionNone:
			snapshot.Groups.Action = append(snapshot.Groups.Action, lane.ID)
			snapshot.Summary.NeedsAction++
		case lane.PullRequest != nil || lane.Branch != lane.Base:
			snapshot.Groups.Waiting = append(snapshot.Groups.Waiting, lane.ID)
			snapshot.Summary.Waiting++
		default:
			snapshot.Groups.Idle = append(snapshot.Groups.Idle, lane.ID)
			snapshot.Summary.Idle++
		}
	}
	snapshot.Summary.Errors = len(snapshot.Errors)
}
