package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/reposync"
)

func TestSyncCheckJSONUsesLocalRefsWithoutFetch(t *testing.T) {
	configPath := repositorySyncTestConfig(t)
	var stdout bytes.Buffer
	app := App{
		Out: &stdout, Err: &bytes.Buffer{}, Runner: command.ExecRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
	}
	root := app.Root()
	root.SetArgs([]string{"--config", configPath, "sync", "check", "--no-fetch", "--json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	var report reposync.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode output %q: %v", stdout.String(), err)
	}
	if report.FetchAttempt || len(report.Repositories) != 1 || report.Repositories[0].State != reposync.StateCurrent {
		t.Fatalf("report = %+v", report)
	}
}

func TestSyncApplyRequiresExplicitApprovalOutsideTTY(t *testing.T) {
	configPath := repositorySyncTestConfig(t)
	app := App{
		Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: command.ExecRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
	}
	root := app.Root()
	root.SetArgs([]string{"--config", configPath, "sync", "apply", "owner/repo"})
	err := root.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "requires --yes") || ExitCode(err) != 2 {
		t.Fatalf("error = %v", err)
	}
}

func TestSyncInteractiveSelectsAndUpdatesSafeRepositories(t *testing.T) {
	configPath := repositorySyncTestConfig(t)
	makeRepositorySyncRemoteCommit(t, configPath)
	var stdout bytes.Buffer
	prompter := &scriptedRepositorySyncPrompter{
		selected: []string{"owner/repo"},
		confirm:  []bool{true},
	}
	app := App{
		Out: &stdout, Err: &bytes.Buffer{}, Runner: command.ExecRunner{},
		InputIsTTY: func() bool { return true }, OutputIsTTY: func() bool { return true },
		syncPrompterSource: prompter,
	}
	root := app.Root()
	root.SetArgs([]string{"--config", configPath, "sync"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := syncGitOutput(t, cfg.Repositories[0].Path, "rev-parse", "HEAD"), syncGitOutput(t, cfg.Repositories[0].Path, "rev-parse", "origin/main"); got != want {
		t.Fatalf("HEAD = %s, want %s", got, want)
	}
	if len(prompter.candidates) != 1 || prompter.candidates[0].ProjectID != "owner/repo" || prompter.confirmCalls != 1 {
		t.Fatalf("candidates=%+v confirmations=%d", prompter.candidates, prompter.confirmCalls)
	}
}

func TestResolveRepositoryTargetsRequiresUnambiguousNames(t *testing.T) {
	repositories := []config.Repository{
		{Name: "api", GitHub: "one/api"},
		{Name: "api", GitHub: "two/api"},
		{Name: "ui", GitHub: "one/ui"},
	}
	if _, err := resolveRepositoryTargets(repositories, []string{"api"}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous error = %v", err)
	}
	selected, err := resolveRepositoryTargets(repositories, []string{"one/ui", "one/ui"})
	if err != nil || len(selected) != 1 || selected[0].GitHub != "one/ui" {
		t.Fatalf("selected = %#v, err = %v", selected, err)
	}
}

func repositorySyncTestConfig(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	repository := filepath.Join(root, "repo")
	runSyncGit(t, root, "init", "--bare", remote)
	runSyncGit(t, root, "init", "--initial-branch=main", repository)
	runSyncGit(t, repository, "config", "user.name", "Beacon Test")
	runSyncGit(t, repository, "config", "user.email", "beacon@example.com")
	if err := os.WriteFile(filepath.Join(repository, "README.md"), []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runSyncGit(t, repository, "add", "README.md")
	runSyncGit(t, repository, "commit", "-m", "initial")
	runSyncGit(t, repository, "remote", "add", "origin", remote)
	runSyncGit(t, repository, "push", "-u", "origin", "main")
	configPath := filepath.Join(root, "config.yaml")
	contents := "version: 2\nrepositories:\n  - name: repo\n    path: " + repository + "\n    github: owner/repo\n    base: main\n    remote: origin\n"
	if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath
}

func makeRepositorySyncRemoteCommit(t *testing.T, configPath string) {
	t.Helper()
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	repository := cfg.Repositories[0]
	remote := syncGitOutput(t, repository.Path, "remote", "get-url", repository.Remote)
	source := filepath.Join(filepath.Dir(configPath), "source")
	runSyncGit(t, filepath.Dir(configPath), "clone", "--branch", repository.Base, remote, source)
	runSyncGit(t, source, "config", "user.name", "Beacon Test")
	runSyncGit(t, source, "config", "user.email", "beacon@example.com")
	if err := os.WriteFile(filepath.Join(source, "remote.txt"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runSyncGit(t, source, "add", "remote.txt")
	runSyncGit(t, source, "commit", "-m", "remote update")
	runSyncGit(t, source, "push", repository.Remote, repository.Base)
}

type scriptedRepositorySyncPrompter struct {
	selected     []string
	confirm      []bool
	candidates   []reposync.Repository
	confirmCalls int
}

func (p *scriptedRepositorySyncPrompter) SelectRepositoryUpdates(_ context.Context, candidates []reposync.Repository) ([]string, error) {
	p.candidates = append([]reposync.Repository(nil), candidates...)
	return append([]string(nil), p.selected...), nil
}

func (p *scriptedRepositorySyncPrompter) Confirm(_ context.Context, _ string) (bool, error) {
	value := p.confirm[p.confirmCalls]
	p.confirmCalls++
	return value, nil
}

func runSyncGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func syncGitOutput(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
