package workscan

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestScannerShowsOnlyProjectsWithWorkByDefault(t *testing.T) {
	now := time.Date(2026, 7, 24, 14, 0, 0, 0, time.UTC)
	repositories := []config.Repository{
		{Name: "dirty", Path: "/dirty", GitHub: "owner/dirty", Base: "main", Remote: "origin"},
		{Name: "review", Path: "/review", GitHub: "owner/review", Base: "main", Remote: "origin"},
		{Name: "idle", Path: "/idle", GitHub: "owner/idle", Base: "main", Remote: "origin"},
	}
	git := fakeGitScanner{results: map[string]gitscan.Result{
		"owner/dirty": {Lanes: []gitscan.LocalLane{localLane("dirty-main", "main", model.PublicationBase, model.Worktree{
			Path: "/dirty", Unstaged: 2, UpdatedAt: now,
		})}},
		"owner/review": {Lanes: []gitscan.LocalLane{localLane("review-main", "main", model.PublicationBase, model.Worktree{
			Path: "/review", UpdatedAt: now,
		})}},
		"owner/idle": {Lanes: []gitscan.LocalLane{localLane("idle-main", "main", model.PublicationBase, model.Worktree{
			Path: "/idle", UpdatedAt: now,
		})}},
	}}
	github := fakePullRequestClient{pullRequests: map[string][]model.PullRequest{
		"owner/review": {{
			Number: 8, Title: "Ready change", URL: "https://github.com/owner/review/pull/8",
			HeadRefName: "feature", HeadRefOID: "abc", BaseRefName: "main",
			UpdatedAt: now, CI: model.CISuccess, MergeState: "CLEAN", Mergeable: "MERGEABLE",
		}},
	}}
	scanner := Scanner{
		Git: git, GitHub: github,
		Discovery: fakeDiscoverer{result: discovery.Result{Repositories: repositories}},
		Now:       func() time.Time { return now },
	}
	cfg := testConfig()

	result, err := scanner.Scan(context.Background(), cfg, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != model.WorkScanSchemaVersion {
		t.Fatalf("schema version = %d", result.SchemaVersion)
	}
	if result.Summary.Projects != 3 || result.Summary.ActiveProjects != 2 ||
		result.Summary.WorkItems != 2 || result.Summary.IdleProjects != 1 {
		t.Fatalf("summary = %#v", result.Summary)
	}
	if len(result.Items) != 2 || result.Items[0].State != model.WorkDirty ||
		result.Items[1].State != model.WorkPullRequest {
		t.Fatalf("items = %#v", result.Items)
	}
	if result.Items[1].PullRequest == nil || result.Items[1].PullRequest.Number != 8 {
		t.Fatalf("pull request item = %#v", result.Items[1])
	}

	withIdle, err := scanner.Scan(context.Background(), cfg, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(withIdle.Items) != 3 || withIdle.Items[2].State != model.WorkIdle ||
		withIdle.Summary.WorkItems != 2 {
		t.Fatalf("items with idle = %#v, summary = %#v", withIdle.Items, withIdle.Summary)
	}
}

func TestScannerPreservesLocalWorkWhenGitHubFails(t *testing.T) {
	now := time.Date(2026, 7, 24, 14, 0, 0, 0, time.UTC)
	repository := config.Repository{
		Name: "local", Path: "/local", GitHub: "owner/local", Base: "main", Remote: "origin",
	}
	scanner := Scanner{
		Git: fakeGitScanner{results: map[string]gitscan.Result{
			repository.GitHub: {
				Lanes: []gitscan.LocalLane{localLane("local", "GH-1", model.PublicationNoUpstream, model.Worktree{
					Path: "/local", AheadBase: 1, UpdatedAt: now,
				})},
				Warnings: []model.ScanError{{Repository: "local", Stage: "worktree", Message: "test warning"}},
			},
		}},
		GitHub: fakePullRequestClient{errors: map[string]error{
			repository.GitHub: errors.New("authentication unavailable"),
		}},
		Discovery: fakeDiscoverer{result: discovery.Result{
			Repositories: []config.Repository{repository},
			Warnings:     []discovery.Warning{{Path: "/other", Stage: "inspect", Message: "not a GitHub repository"}},
		}},
		Now: func() time.Time { return now },
	}

	result, err := scanner.Scan(context.Background(), testConfig(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].State != model.WorkUnpublished {
		t.Fatalf("items = %#v", result.Items)
	}
	if len(result.Errors) != 1 || result.Errors[0].Stage != "github-pull-requests" {
		t.Fatalf("errors = %#v", result.Errors)
	}
	if len(result.Warnings) != 2 || result.Summary.Errors != 1 || result.Summary.Warnings != 2 {
		t.Fatalf("warnings = %#v, summary = %#v", result.Warnings, result.Summary)
	}
}

func TestScannerRejectsSourcesWithoutGitHubRepositories(t *testing.T) {
	scanner := Scanner{
		Git:       fakeGitScanner{},
		GitHub:    fakePullRequestClient{},
		Discovery: fakeDiscoverer{},
	}
	_, err := scanner.Scan(context.Background(), testConfig(), false, false)
	if err == nil || !strings.Contains(err.Error(), "no accessible GitHub repositories") {
		t.Fatalf("error = %v", err)
	}
}

func TestScannerDoesNotClassifyIncompleteRepositoryAsIdle(t *testing.T) {
	repository := config.Repository{
		Name: "unknown", Path: "/unknown", GitHub: "owner/unknown", Base: "main", Remote: "origin",
	}
	scanner := Scanner{
		Git: fakeGitScanner{results: map[string]gitscan.Result{
			repository.GitHub: {
				Errors: []model.ScanError{{Repository: "unknown", Stage: "worktrees", Message: "unavailable"}},
			},
		}},
		GitHub: fakePullRequestClient{errors: map[string]error{
			repository.GitHub: errors.New("offline"),
		}},
		Discovery: fakeDiscoverer{result: discovery.Result{Repositories: []config.Repository{repository}}},
	}
	result, err := scanner.Scan(context.Background(), testConfig(), false, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 0 || result.Summary.IdleProjects != 0 || result.Summary.UnknownProjects != 1 {
		t.Fatalf("items = %#v, summary = %#v", result.Items, result.Summary)
	}
}

func TestInProgressClassification(t *testing.T) {
	cleanBase := model.Lane{
		Branch: "main", Base: "main", Worktree: &model.Worktree{},
		Signals: model.Signals{Publication: model.PublicationBase},
	}
	tests := []struct {
		name string
		lane model.Lane
		want bool
	}{
		{name: "clean base", lane: cleanBase, want: false},
		{name: "open pull request", lane: model.Lane{PullRequest: &model.PullRequest{}}, want: true},
		{name: "dirty base", lane: mutateLane(cleanBase, func(lane *model.Lane) { lane.Worktree.Untracked = 1 }), want: true},
		{name: "feature branch", lane: mutateLane(cleanBase, func(lane *model.Lane) { lane.Branch = "feature" }), want: true},
		{name: "detached", lane: mutateLane(cleanBase, func(lane *model.Lane) { lane.Worktree.Detached = true }), want: true},
		{name: "prunable", lane: mutateLane(cleanBase, func(lane *model.Lane) { lane.Worktree.Prunable = true }), want: false},
		{name: "ahead base", lane: mutateLane(cleanBase, func(lane *model.Lane) { lane.Worktree.AheadBase = 1 }), want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := inProgress(test.lane); actual != test.want {
				t.Fatalf("inProgress() = %t, want %t", actual, test.want)
			}
		})
	}
}

func testConfig() config.Config {
	return config.Config{
		Version: config.Version,
		Settings: config.Settings{
			MaxParallel: 4, GitHubAuthor: "@me",
			RemoteRefreshInterval: time.Hour, StaleAfter: 24 * time.Hour,
		},
		Sources: []config.Source{{Path: "/source"}},
	}
}

func localLane(id, branch string, publication model.PublicationState, worktree model.Worktree) gitscan.LocalLane {
	return gitscan.LocalLane{ID: id, Branch: branch, Publication: publication, Worktree: worktree}
}

func mutateLane(value model.Lane, mutate func(*model.Lane)) model.Lane {
	worktree := *value.Worktree
	value.Worktree = &worktree
	mutate(&value)
	return value
}

type fakeGitScanner struct {
	results map[string]gitscan.Result
}

func (f fakeGitScanner) Scan(_ context.Context, repository config.Repository, _ bool, _ time.Duration) gitscan.Result {
	return f.results[repository.GitHub]
}

type fakePullRequestClient struct {
	pullRequests map[string][]model.PullRequest
	errors       map[string]error
}

func (f fakePullRequestClient) ListOpen(_ context.Context, repository, _ string) ([]model.PullRequest, error) {
	return f.pullRequests[repository], f.errors[repository]
}

type fakeDiscoverer struct {
	result discovery.Result
}

func (f fakeDiscoverer) Discover(context.Context, []config.Source) discovery.Result {
	return f.result
}
