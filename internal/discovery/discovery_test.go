package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
)

func TestParseGitHubRemote(t *testing.T) {
	tests := map[string]string{
		"git@github.com:owner/repo.git":           "owner/repo",
		"ssh://git@github.com/owner/repo.git":     "owner/repo",
		"https://github.com/owner/repo.git":       "owner/repo",
		"http://github.com/owner/repo":            "owner/repo",
		"git+ssh://git@github.com/owner/repo.git": "owner/repo",
	}
	for remote, expected := range tests {
		actual, ok := ParseGitHubRemote(remote)
		if !ok || actual != expected {
			t.Fatalf("ParseGitHubRemote(%q) = %q, %t", remote, actual, ok)
		}
	}
	for _, remote := range []string{"git@gitlab.com:owner/repo.git", "https://github.com/owner", "local/path"} {
		if actual, ok := ParseGitHubRemote(remote); ok {
			t.Fatalf("ParseGitHubRemote(%q) = %q, true", remote, actual)
		}
	}
}

func TestRepositoryRootsStopsAtRepositoriesAndDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "owner", "first")
	second := filepath.Join(root, "second")
	linked := filepath.Join(root, "linked")
	for _, path := range []string{first, filepath.Join(first, "nested", "hidden"), second} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(first, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, ".git"), []byte("gitdir: elsewhere"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(first, linked); err != nil {
		t.Fatal(err)
	}

	roots, warnings := repositoryRoots(root)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	expected := []string{first, second}
	if fmt.Sprint(roots) != fmt.Sprint(expected) {
		t.Fatalf("roots = %#v, want %#v", roots, expected)
	}
}

func TestRepositoryRootsRejectsSymlinkSource(t *testing.T) {
	target := t.TempDir()
	if err := os.Mkdir(filepath.Join(target, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "repository-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	roots, warnings := repositoryRoots(link)
	if len(roots) != 0 || len(warnings) != 1 || !strings.Contains(warnings[0].Message, "symbolic link") {
		t.Fatalf("roots=%#v warnings=%#v", roots, warnings)
	}
}

func TestDiscoverRepositoryRootPrefersOriginAndUsesGitHubMetadata(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "unusual repository")
	initRepository(t, repository)
	runGit(t, repository, "remote", "add", "upstream", "https://github.com/other/fallback.git")
	runGit(t, repository, "remote", "add", "origin", "git@github.com:Owner/Canonical.git")

	discoverer := Discoverer{Runner: githubRunner{delegate: command.ExecRunner{}, bases: map[string]string{"Owner/Canonical": "trunk"}}}
	result := discoverer.Discover(context.Background(), []config.Source{{Path: repository}})
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	if len(result.Repositories) != 1 {
		t.Fatalf("repositories = %#v", result.Repositories)
	}
	repo := result.Repositories[0]
	if repo.Name != "Canonical" || repo.GitHub != "Owner/Canonical" || repo.Base != "trunk" || repo.Remote != "origin" {
		t.Fatalf("repository = %#v", repo)
	}
	if repo.CommonDir == "" {
		t.Fatalf("repository has no Git common-directory identity: %#v", repo)
	}
}

func TestDiscoverParentDeduplicatesWorktreesAndWarnsForNonGitHub(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "repo")
	worktree := filepath.Join(root, "repo-worktree")
	localOnly := filepath.Join(root, "local-only")
	initRepository(t, repository)
	runGit(t, repository, "remote", "add", "origin", "https://github.com/owner/repo.git")
	runGit(t, repository, "branch", "linked")
	runGit(t, repository, "worktree", "add", worktree, "linked")
	initRepository(t, localOnly)
	runGit(t, localOnly, "remote", "add", "origin", "https://gitlab.com/owner/local.git")

	discoverer := Discoverer{Runner: githubRunner{delegate: command.ExecRunner{}, bases: map[string]string{"owner/repo": "main"}}}
	result := discoverer.Discover(context.Background(), []config.Source{{Path: root}})
	if len(result.Repositories) != 1 || result.Repositories[0].GitHub != "owner/repo" {
		t.Fatalf("repositories = %#v", result.Repositories)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0].Message, "no GitHub remote") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestDiscoverCancellationBecomesScopedWarning(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := (Discoverer{Runner: canceledRunner{}}).Discover(ctx, []config.Source{{Path: root}})
	if len(result.Repositories) != 0 || len(result.Warnings) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if result.Warnings[0].Path != root || !strings.Contains(result.Warnings[0].Message, context.Canceled.Error()) {
		t.Fatalf("warning = %#v", result.Warnings[0])
	}
}

type githubRunner struct {
	delegate command.Runner
	bases    map[string]string
}

type canceledRunner struct{}

func (canceledRunner) Run(ctx context.Context, _ string, _ string, _ ...string) ([]byte, error) {
	return nil, ctx.Err()
}

func (r githubRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if name != "gh" {
		return r.delegate.Run(ctx, dir, name, args...)
	}
	if len(args) < 4 || args[0] != "repo" || args[1] != "view" {
		return nil, fmt.Errorf("unexpected gh arguments: %v", args)
	}
	repository := args[2]
	base, exists := r.bases[repository]
	if !exists {
		return nil, fmt.Errorf("repository inaccessible: %s", repository)
	}
	return json.Marshal(map[string]any{
		"nameWithOwner":    repository,
		"defaultBranchRef": map[string]string{"name": base},
	})
}

func initRepository(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, path, "init", "-b", "main")
	runGit(t, path, "config", "user.name", "Beacon Test")
	runGit(t, path, "config", "user.email", "beacon@example.com")
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, path, "add", "README.md")
	runGit(t, path, "commit", "-m", "initial")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
