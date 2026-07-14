package reposync

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
)

type State string

const (
	StateCurrent     State = "current"
	StateBehind      State = "behind"
	StateAhead       State = "ahead"
	StateDiverged    State = "diverged"
	StateBlocked     State = "blocked"
	StateUnavailable State = "unavailable"
)

type Action string

const (
	ActionNone                 Action = "none"
	ActionFastForward          Action = "fast_forward"
	ActionSwitchAndFastForward Action = "switch_and_fast_forward"
)

type Repository struct {
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	Path            string `json:"path"`
	Base            string `json:"base"`
	Remote          string `json:"remote"`
	CurrentBranch   string `json:"current_branch,omitempty"`
	BaseWorktree    string `json:"base_worktree,omitempty"`
	CurrentAhead    int    `json:"current_ahead"`
	CurrentBehind   int    `json:"current_behind"`
	DefaultAhead    int    `json:"default_ahead"`
	DefaultBehind   int    `json:"default_behind"`
	Dirty           bool   `json:"dirty"`
	Detached        bool   `json:"detached"`
	NeedsUpdate     bool   `json:"needs_update"`
	CanUpdate       bool   `json:"can_update"`
	Fetched         bool   `json:"fetched"`
	Updated         bool   `json:"updated"`
	State           State  `json:"state"`
	Action          Action `json:"action"`
	Reason          string `json:"reason"`
	Error           string `json:"error,omitempty"`
	currentID       string
	localDefaultID  string
	remoteDefaultID string
}

type Report struct {
	CheckedAt    time.Time    `json:"checked_at"`
	FetchAttempt bool         `json:"fetch_attempted"`
	Repositories []Repository `json:"repositories"`
}

type Service struct {
	Runner      command.Runner
	MaxParallel int
	Now         func() time.Time
}

func (s Service) Check(ctx context.Context, repositories []config.Repository, fetch bool) Report {
	results := s.mapRepositories(ctx, repositories, func(ctx context.Context, repository config.Repository) Repository {
		return s.inspect(ctx, repository, fetch)
	})
	return Report{CheckedAt: s.now(), FetchAttempt: fetch, Repositories: results}
}

func (s Service) Apply(ctx context.Context, repositories []config.Repository, projectIDs []string) Report {
	selected := selectRepositories(repositories, projectIDs)
	results := s.mapRepositories(ctx, selected, func(ctx context.Context, repository config.Repository) Repository {
		return s.apply(ctx, repository)
	})
	return Report{CheckedAt: s.now(), FetchAttempt: true, Repositories: results}
}

func (s Service) mapRepositories(
	ctx context.Context,
	repositories []config.Repository,
	operation func(context.Context, config.Repository) Repository,
) []Repository {
	ordered := append([]config.Repository(nil), repositories...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].GitHub != ordered[j].GitHub {
			return ordered[i].GitHub < ordered[j].GitHub
		}
		return ordered[i].Path < ordered[j].Path
	})
	results := make([]Repository, len(ordered))
	workers := s.MaxParallel
	if workers <= 0 {
		workers = 4
	}
	if workers > len(ordered) {
		workers = len(ordered)
	}
	if workers == 0 {
		return results
	}
	jobs := make(chan int)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := range jobs {
				results[index] = operation(ctx, ordered[index])
			}
		}()
	}
	for index := range ordered {
		select {
		case jobs <- index:
		case <-ctx.Done():
			close(jobs)
			wait.Wait()
			return results
		}
	}
	close(jobs)
	wait.Wait()
	return results
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func selectRepositories(repositories []config.Repository, projectIDs []string) []config.Repository {
	wanted := make(map[string]struct{}, len(projectIDs))
	for _, projectID := range projectIDs {
		wanted[projectID] = struct{}{}
	}
	selected := make([]config.Repository, 0, len(wanted))
	for _, repository := range repositories {
		if _, ok := wanted[repository.GitHub]; ok {
			selected = append(selected, repository)
		}
	}
	return selected
}
