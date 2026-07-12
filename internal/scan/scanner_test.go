package scan

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/progress"
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

func TestScanManyCollectsRemoteEvidenceOnceForEntireBatch(t *testing.T) {
	now := time.Date(2026, 7, 12, 16, 0, 0, 0, time.UTC)
	repositories := make([]config.Repository, 80)
	for index := range repositories {
		repositories[index] = config.Repository{
			Name: "repo-" + strconv.Itoa(index), GitHub: "owner/repo-" + strconv.Itoa(index),
			Base: "main", Remote: "origin",
		}
	}
	cfg := config.Config{Settings: config.Settings{
		MaxParallel: 4, RemoteRefreshInterval: 15 * time.Minute,
		StaleAfter: time.Hour, GitHubAuthor: "@me", GitHubScope: config.GitHubScopeMine,
	}}
	remote := &countingGitHubClient{}
	scanner := Scanner{Git: fakeGitScanner{now: now}, GitHub: remote, Now: func() time.Time { return now }}

	snapshots, err := scanner.ScanMany(context.Background(), cfg, repositories, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != len(repositories) || remote.callCount() != 1 || remote.batchSize() != len(repositories) {
		t.Fatalf("snapshots=%d calls=%d batch=%d", len(snapshots), remote.callCount(), remote.batchSize())
	}
}

func TestScanSeparatesOptionalRepositoryWarningsFromErrors(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "specs", "0001-invalid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "PROJECT_PROGRESS_SUMMARY.md"), []byte("# no progress table\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "specs", "0001-invalid", "SPEC.md"), []byte("not front matter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Settings: config.Settings{MaxParallel: 1, RemoteRefreshInterval: time.Minute, StaleAfter: time.Hour, GitHubAuthor: "@me", GitHubScope: config.GitHubScopeMine},
		Sources:  []config.Source{{Path: root}},
	}
	discoverer := fakeRepositoryDiscoverer{result: discovery.Result{
		Repositories: []config.Repository{{Name: "example", Path: root, GitHub: "owner/example", Base: "main", Remote: "origin"}},
		Warnings:     []discovery.Warning{{Path: "/missing/repo", Stage: "inspect", Message: "repository unavailable"}},
	}}
	scanner := Scanner{Git: fakeGitScanner{now: now}, GitHub: fakeGitHubClient{}, Discovery: discoverer, Now: func() time.Time { return now }}
	snapshot, err := scanner.Scan(context.Background(), cfg, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Errors) != 0 || snapshot.Summary.Errors != 0 || len(snapshot.Warnings) != 3 || snapshot.Summary.Warnings != 3 {
		t.Fatalf("snapshot diagnostics = errors:%#v warnings:%#v summary:%#v", snapshot.Errors, snapshot.Warnings, snapshot.Summary)
	}
	if len(snapshot.Projects) != 1 || len(snapshot.Projects[0].Errors) != 0 || len(snapshot.Projects[0].Warnings) != 2 {
		t.Fatalf("project = %#v", snapshot.Projects)
	}
}

func TestScanMergesDiscoveredRepositoriesWithExplicitOverrides(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	discoveredPath := filepath.Join(root, "discovered")
	explicitPath := filepath.Join(root, "explicit")
	secondPath := filepath.Join(root, "second")
	cfg := config.Config{
		Path: "/tmp/config.yaml",
		Settings: config.Settings{
			MaxParallel: 2, RemoteRefreshInterval: time.Minute, StaleAfter: time.Hour,
			GitHubAuthor: "@me", GitHubScope: config.GitHubScopeMine,
		},
		Sources: []config.Source{{Path: root}},
		Repositories: []config.Repository{{
			Name: "explicit", Path: explicitPath, GitHub: "owner/same", Base: "trunk", Remote: "upstream",
		}},
	}
	discoverer := fakeRepositoryDiscoverer{result: discovery.Result{Repositories: []config.Repository{
		{Name: "discovered", Path: discoveredPath, GitHub: "owner/same", Base: "main", Remote: "origin"},
		{Name: "second", Path: secondPath, GitHub: "owner/second", Base: "main", Remote: "origin"},
	}}}
	scanner := Scanner{Git: fakeGitScanner{now: now}, GitHub: fakeGitHubClient{}, Discovery: discoverer, Now: func() time.Time { return now }}
	snapshot, err := scanner.Scan(context.Background(), cfg, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Projects) != 2 || snapshot.Summary.Projects != 2 {
		t.Fatalf("projects = %#v", snapshot.Projects)
	}
	for _, project := range snapshot.Projects {
		if project.GitHub == "owner/same" && (project.Name != "explicit" || project.Path != explicitPath || project.Base != "trunk") {
			t.Fatalf("explicit override was not retained: %#v", project)
		}
	}
}

func TestScanExplicitRepositoryOverridesDiscoveredCommonDirectory(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	commonDir := filepath.Join(root, "repo", ".git")
	discoveredPath := filepath.Join(root, "repo")
	explicitPath := filepath.Join(root, "repo-worktree")
	cfg := config.Config{
		Settings: config.Settings{MaxParallel: 2, RemoteRefreshInterval: time.Minute, StaleAfter: time.Hour, GitHubAuthor: "@me", GitHubScope: config.GitHubScopeMine},
		Sources:  []config.Source{{Path: root}},
		Repositories: []config.Repository{{
			Name: "canonical", Path: explicitPath, GitHub: "owner/canonical", Base: "main", Remote: "origin", CommonDir: commonDir,
		}},
	}
	discoverer := fakeRepositoryDiscoverer{result: discovery.Result{Repositories: []config.Repository{{
		Name: "old-metadata", Path: discoveredPath, GitHub: "owner/old-name", Base: "main", Remote: "origin", CommonDir: commonDir,
	}}}}
	scanner := Scanner{Git: fakeGitScanner{now: now}, GitHub: fakeGitHubClient{}, Discovery: discoverer, Now: func() time.Time { return now }}
	snapshot, err := scanner.Scan(context.Background(), cfg, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Projects) != 1 || snapshot.Projects[0].GitHub != "owner/canonical" || snapshot.Projects[0].Path != explicitPath {
		t.Fatalf("projects = %#v", snapshot.Projects)
	}
}

func TestCorrelateProgressUsesExactIssueURLAndSelectedFeature(t *testing.T) {
	features := []progress.Feature{
		{ID: "0001", Slug: "old", Phase: "deliver", IssueURLs: []string{"https://github.com/owner/repo/issues/1"}},
		{ID: "0002", Slug: "active", Phase: "implement", IssueURLs: []string{"https://github.com/owner/repo/issues/2"}},
	}
	result := progress.Result{Features: features, Selected: &features[1]}
	selected, byIssue := correlateProgress(config.Repository{GitHub: "owner/repo"}, result)
	if selected == nil || selected.FeatureID != "0002" || byIssue[2].Feature != "active" {
		t.Fatalf("selected=%#v byIssue=%#v", selected, byIssue)
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

func (fakeGitHubClient) Collect(_ context.Context, repositories []config.Repository, _, _ string, _ int) model.RemoteCollection {
	collection := model.RemoteCollection{Repositories: make(map[string]model.RemoteEvidence), Errors: []model.ScanError{}, Warnings: []model.ScanError{}}
	for _, repository := range repositories {
		evidence := model.RemoteEvidence{PullRequests: []model.PullRequest{}, Issues: []model.Issue{}, Errors: []model.ScanError{}, Warnings: []model.ScanError{}}
		if repository.GitHub == "owner/failing" {
			evidence.Errors = append(evidence.Errors, model.ScanError{Repository: repository.Name, Stage: "github", Message: "GitHub unavailable"})
		}
		collection.Repositories[repository.GitHub] = evidence
	}
	return collection
}

type countingGitHubClient struct {
	mutex sync.Mutex
	calls int
	size  int
}

func (c *countingGitHubClient) Collect(_ context.Context, repositories []config.Repository, _, _ string, _ int) model.RemoteCollection {
	c.mutex.Lock()
	c.calls++
	c.size = len(repositories)
	c.mutex.Unlock()
	return fakeGitHubClient{}.Collect(context.Background(), repositories, "mine", "@me", 1)
}

func (c *countingGitHubClient) callCount() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.calls
}

func (c *countingGitHubClient) batchSize() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.size
}

type fakeRepositoryDiscoverer struct{ result discovery.Result }

func (d fakeRepositoryDiscoverer) Discover(context.Context, []config.Source) discovery.Result {
	return d.result
}
