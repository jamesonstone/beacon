package scan

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestScanPreservesHealthyResultsWhenRepositoryFails(t *testing.T) {
	now := time.Date(2026, 7, 9, 16, 0, 0, 0, time.UTC)
	cfg := config.Config{
		Path:     "/tmp/config.yaml",
		Settings: config.Settings{MaxParallel: 2, RemoteRefreshInterval: time.Minute, StaleAfter: time.Hour, GitHubAuthor: "@me"},
		Repositories: []config.Repository{
			{Name: "healthy", GitHub: "owner/healthy", Base: "main"},
			{Name: "failing", GitHub: "owner/failing", Base: "main"},
		},
	}
	scanner := Scanner{Git: fakeGitScanner{now: now}, GitHub: fakeGitHubClient{}, Now: func() time.Time { return now }}
	snapshot, err := scanner.Scan(context.Background(), cfg, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Lanes) != 2 {
		t.Fatalf("lanes = %#v", snapshot.Lanes)
	}
	if len(snapshot.Errors) != 1 || snapshot.Errors[0].Repository != "failing" || snapshot.Errors[0].Stage != "github" {
		t.Fatalf("errors = %#v", snapshot.Errors)
	}
	if snapshot.Summary.Total != 2 || snapshot.Summary.Errors != 1 {
		t.Fatalf("summary = %#v", snapshot.Summary)
	}
}

type fakeGitScanner struct{ now time.Time }

func (f fakeGitScanner) Scan(_ context.Context, repo config.Repository, _ bool, _ time.Duration) gitscan.Result {
	return gitscan.Result{
		Lanes: []gitscan.LocalLane{{
			ID: "git:" + repo.GitHub + "@main", Branch: "main", Publication: model.PublicationBase,
			Worktree: model.Worktree{Path: "/tmp/" + repo.Name, HeadOID: repo.Name, UpdatedAt: f.now},
		}},
		Refresh: model.Refresh{Repository: repo.Name},
	}
}

type fakeGitHubClient struct{}

func (fakeGitHubClient) ListOpen(_ context.Context, repository, _ string) ([]model.PullRequest, error) {
	if repository == "owner/failing" {
		return nil, errors.New("GitHub unavailable")
	}
	return []model.PullRequest{}, nil
}
