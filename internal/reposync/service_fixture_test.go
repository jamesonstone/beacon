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
