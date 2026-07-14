package reposync

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
)

func TestCheckIsLocalUntilFetchAndApplyFastForwardsDefault(t *testing.T) {
	fixture := newGitFixture(t)
	fixture.commitSource("remote change", "two\n")
	runner := &recordingRunner{delegate: command.ExecRunner{}}
	service := Service{Runner: runner, MaxParallel: 2, Now: fixedNow}

	local := service.Check(context.Background(), []config.Repository{fixture.repository}, false)
	if local.Repositories[0].State != StateCurrent || local.FetchAttempt {
		t.Fatalf("local check = %+v", local.Repositories[0])
	}
	if runner.containsArgs("fetch") {
		t.Fatal("local check unexpectedly fetched")
	}

	refreshed := service.Check(context.Background(), []config.Repository{fixture.repository}, true)
	candidate := refreshed.Repositories[0]
	if candidate.State != StateBehind || !candidate.CanUpdate || candidate.Action != ActionFastForward || candidate.CurrentBehind != 1 {
		t.Fatalf("refreshed candidate = %+v", candidate)
	}
	if !runner.containsArgs("fetch", "--prune", "--no-tags") || runner.containsName("gh") {
		t.Fatalf("commands = %#v", runner.commands)
	}

	applied := service.Apply(context.Background(), []config.Repository{fixture.repository}, []string{fixture.repository.GitHub})
	result := applied.Repositories[0]
	if !result.Updated || result.State != StateCurrent || result.NeedsUpdate {
		t.Fatalf("applied result = %+v", result)
	}
	if got, want := fixture.git(fixture.work, "rev-parse", "HEAD"), fixture.git(fixture.work, "rev-parse", "origin/main"); got != want {
		t.Fatalf("HEAD = %s, want %s", got, want)
	}
}

func TestApplyReturnsFullyMergedFeatureBranchToDefault(t *testing.T) {
	fixture := newGitFixture(t)
	fixture.git(fixture.work, "switch", "-c", "GH-12")
	fixture.write(fixture.work, "feature.txt", "merged\n")
	fixture.git(fixture.work, "add", "feature.txt")
	fixture.git(fixture.work, "commit", "-m", "feature")
	fixture.git(fixture.work, "push", "origin", "GH-12")
	fixture.git(fixture.source, "fetch", "origin", "GH-12")
	fixture.git(fixture.source, "merge", "--ff-only", "origin/GH-12")
	fixture.git(fixture.source, "push", "origin", "main")
	service := Service{Runner: command.ExecRunner{}, Now: fixedNow}

	checked := service.Check(context.Background(), []config.Repository{fixture.repository}, true).Repositories[0]
	if checked.Action != ActionSwitchAndFastForward || !checked.CanUpdate {
		t.Fatalf("checked = %+v", checked)
	}
	result := service.Apply(context.Background(), []config.Repository{fixture.repository}, []string{fixture.repository.GitHub}).Repositories[0]
	if !result.Updated || result.CurrentBranch != "main" || result.State != StateCurrent {
		t.Fatalf("result = %+v", result)
	}
}

func TestCheckRefusesDirtyAndDivergedRepositories(t *testing.T) {
	t.Run("dirty", func(t *testing.T) {
		fixture := newGitFixture(t)
		fixture.commitSource("remote change", "two\n")
		fixture.write(fixture.work, "draft.txt", "local\n")
		result := (Service{Runner: command.ExecRunner{}}).Check(context.Background(), []config.Repository{fixture.repository}, true).Repositories[0]
		if result.State != StateBlocked || result.CanUpdate || !strings.Contains(result.Reason, "local changes") {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("diverged default", func(t *testing.T) {
		fixture := newGitFixture(t)
		fixture.commitSource("remote change", "remote\n")
		fixture.write(fixture.work, "local.txt", "local\n")
		fixture.git(fixture.work, "add", "local.txt")
		fixture.git(fixture.work, "commit", "-m", "local change")
		result := (Service{Runner: command.ExecRunner{}}).Check(context.Background(), []config.Repository{fixture.repository}, true).Repositories[0]
		if result.State != StateDiverged || result.CanUpdate || result.DefaultAhead != 1 || result.DefaultBehind != 1 {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("unmerged feature branch", func(t *testing.T) {
		fixture := newGitFixture(t)
		fixture.commitSource("remote change", "remote\n")
		fixture.git(fixture.work, "switch", "-c", "unfinished")
		fixture.write(fixture.work, "local.txt", "local\n")
		fixture.git(fixture.work, "add", "local.txt")
		fixture.git(fixture.work, "commit", "-m", "unfinished change")
		result := (Service{Runner: command.ExecRunner{}}).Check(context.Background(), []config.Repository{fixture.repository}, true).Repositories[0]
		if result.State != StateDiverged || result.CanUpdate || result.Action != ActionNone || !strings.Contains(result.Reason, "merge or rebase manually") {
			t.Fatalf("result = %+v", result)
		}
	})
}

func TestCheckSortsRepositoriesDeterministically(t *testing.T) {
	fixture := newGitFixture(t)
	repositories := []config.Repository{
		{Name: "z", GitHub: "owner/z", Path: fixture.work, Base: "main", Remote: "origin"},
		{Name: "a", GitHub: "owner/a", Path: fixture.work, Base: "main", Remote: "origin"},
	}
	report := (Service{Runner: command.ExecRunner{}, MaxParallel: 2, Now: fixedNow}).Check(context.Background(), repositories, false)
	if report.Repositories[0].ProjectID != "owner/a" || report.Repositories[1].ProjectID != "owner/z" {
		t.Fatalf("order = %s, %s", report.Repositories[0].ProjectID, report.Repositories[1].ProjectID)
	}
	if !report.CheckedAt.Equal(fixedNow()) {
		t.Fatalf("checked_at = %s", report.CheckedAt)
	}
}

func TestCheckRefusesDetachedMissingDefaultAndMultipleWorktrees(t *testing.T) {
	t.Run("detached head", func(t *testing.T) {
		fixture := newGitFixture(t)
		fixture.git(fixture.work, "checkout", "--detach")
		fixture.commitSource("remote change", "two\n")
		result := (Service{Runner: command.ExecRunner{}}).Check(context.Background(), []config.Repository{fixture.repository}, true).Repositories[0]
		if result.State != StateBlocked || !result.Detached || result.CanUpdate || !strings.Contains(result.Reason, "detached") {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("missing local default", func(t *testing.T) {
		fixture := newGitFixture(t)
		fixture.git(fixture.work, "branch", "-m", "topic")
		result := (Service{Runner: command.ExecRunner{}}).Check(context.Background(), []config.Repository{fixture.repository}, false).Repositories[0]
		if result.State != StateBlocked || result.CanUpdate || !strings.Contains(result.Reason, "missing") {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("default branch in another worktree", func(t *testing.T) {
		fixture := newGitFixture(t)
		fixture.git(fixture.work, "switch", "-c", "GH-12")
		fixture.write(fixture.work, "feature.txt", "merged\n")
		fixture.git(fixture.work, "add", "feature.txt")
		fixture.git(fixture.work, "commit", "-m", "feature")
		fixture.git(fixture.work, "push", "origin", "GH-12")
		fixture.git(fixture.source, "fetch", "origin", "GH-12")
		fixture.git(fixture.source, "merge", "--ff-only", "origin/GH-12")
		fixture.git(fixture.source, "push", "origin", "main")
		otherWorktree := filepath.Join(filepath.Dir(fixture.work), "main-worktree")
		fixture.git(fixture.work, "worktree", "add", otherWorktree, "main")
		result := (Service{Runner: command.ExecRunner{}}).Check(context.Background(), []config.Repository{fixture.repository}, true).Repositories[0]
		if result.State != StateBlocked || result.CanUpdate || result.BaseWorktree == "" || !strings.Contains(result.Reason, "another worktree") {
			t.Fatalf("result = %+v", result)
		}
	})
}

func TestApplyRefusesWorktreeChangeAfterInspection(t *testing.T) {
	fixture := newGitFixture(t)
	originalHead := fixture.git(fixture.work, "rev-parse", "HEAD")
	fixture.commitSource("remote change", "two\n")
	runner := &dirtyAfterInspectionRunner{delegate: command.ExecRunner{}, path: fixture.work}
	result := (Service{Runner: runner}).Apply(
		context.Background(),
		[]config.Repository{fixture.repository},
		[]string{fixture.repository.GitHub},
	).Repositories[0]
	if result.State != StateBlocked || result.CanUpdate || !strings.Contains(result.Reason, "changed after inspection") {
		t.Fatalf("result = %+v", result)
	}
	if got := fixture.git(fixture.work, "rev-parse", "HEAD"); got != originalHead {
		t.Fatalf("HEAD changed: got %s, want %s", got, originalHead)
	}
}

type gitFixture struct {
	t          *testing.T
	remote     string
	source     string
	work       string
	repository config.Repository
}

func newGitFixture(t *testing.T) gitFixture {
	t.Helper()
	root := t.TempDir()
	fixture := gitFixture{
		t: t, remote: filepath.Join(root, "remote.git"),
		source: filepath.Join(root, "source"), work: filepath.Join(root, "work"),
	}
	fixture.git(root, "init", "--bare", fixture.remote)
	fixture.git(fixture.remote, "symbolic-ref", "HEAD", "refs/heads/main")
	fixture.git(root, "init", "--initial-branch=main", fixture.source)
	fixture.configure(fixture.source)
	fixture.write(fixture.source, "state.txt", "one\n")
	fixture.git(fixture.source, "add", "state.txt")
	fixture.git(fixture.source, "commit", "-m", "initial")
	fixture.git(fixture.source, "remote", "add", "origin", fixture.remote)
	fixture.git(fixture.source, "push", "-u", "origin", "main")
	fixture.git(root, "clone", fixture.remote, fixture.work)
	fixture.configure(fixture.work)
	fixture.repository = config.Repository{
		Name: "work", GitHub: "owner/work", Path: fixture.work,
		Base: "main", Remote: "origin",
	}
	return fixture
}

func (f gitFixture) commitSource(message, content string) {
	f.t.Helper()
	f.write(f.source, "state.txt", content)
	f.git(f.source, "add", "state.txt")
	f.git(f.source, "commit", "-m", message)
	f.git(f.source, "push", "origin", "main")
}

func (f gitFixture) configure(path string) {
	f.git(path, "config", "user.name", "Beacon Test")
	f.git(path, "config", "user.email", "beacon@example.com")
}

func (f gitFixture) write(path, name, content string) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(path, name), []byte(content), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

func (f gitFixture) git(path string, args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

type recordedCommand struct {
	name string
	args []string
}

type recordingRunner struct {
	delegate command.Runner
	mutex    sync.Mutex
	commands []recordedCommand
}

type dirtyAfterInspectionRunner struct {
	delegate    command.Runner
	path        string
	statusCalls int
}

func (r *dirtyAfterInspectionRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if name == "git" && len(args) > 0 && args[0] == "status" {
		r.statusCalls++
		if r.statusCalls == 2 {
			if err := os.WriteFile(filepath.Join(r.path, "concurrent.txt"), []byte("changed\n"), 0o600); err != nil {
				return nil, err
			}
		}
	}
	return r.delegate.Run(ctx, dir, name, args...)
}

func (r *recordingRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	r.mutex.Lock()
	r.commands = append(r.commands, recordedCommand{name: name, args: append([]string(nil), args...)})
	r.mutex.Unlock()
	return r.delegate.Run(ctx, dir, name, args...)
}

func (r *recordingRunner) containsName(name string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, item := range r.commands {
		if item.name == name {
			return true
		}
	}
	return false
}

func (r *recordingRunner) containsArgs(args ...string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	needle := strings.Join(args, " ")
	for _, item := range r.commands {
		if strings.Contains(strings.Join(item.args, " "), needle) {
			return true
		}
	}
	return false
}

func fixedNow() time.Time {
	return time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
}
