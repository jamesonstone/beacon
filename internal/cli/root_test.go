package cli

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestRootRegistersInitAndBareDashboard(t *testing.T) {
	app := App{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &recordingRunner{}}
	root := app.Root()
	if root.RunE == nil {
		t.Fatal("bare root command has no dashboard runner")
	}
	command, _, err := root.Find([]string{"init"})
	if err != nil || command == nil || command.Name() != "init" {
		t.Fatalf("init command = %v, %v", command, err)
	}
}

func TestResolveColorModesAndNoColor(t *testing.T) {
	app := App{OutputIsTTY: func() bool { return true }}
	t.Setenv("NO_COLOR", "")
	if enabled, err := app.resolveColor("auto"); err != nil || !enabled {
		t.Fatalf("auto = %t, %v", enabled, err)
	}
	t.Setenv("NO_COLOR", "1")
	if enabled, err := app.resolveColor("auto"); err != nil || enabled {
		t.Fatalf("NO_COLOR auto = %t, %v", enabled, err)
	}
	if enabled, err := app.resolveColor("always"); err != nil || !enabled {
		t.Fatalf("always = %t, %v", enabled, err)
	}
	if _, err := app.resolveColor("sometimes"); err == nil || ExitCode(err) != 2 {
		t.Fatalf("invalid mode error = %v", err)
	}
}

func TestOpenLanePrefersPullRequestThenIssueThenWorktree(t *testing.T) {
	for _, test := range []struct {
		name string
		lane model.Lane
		want string
	}{
		{
			name: "pull request",
			lane: model.Lane{PullRequest: &model.PullRequest{URL: "https://github.com/owner/repo/pull/2"}, Issue: &model.Issue{URL: "https://github.com/owner/repo/issues/1"}, Worktree: &model.Worktree{Path: "/tmp/repo"}},
			want: "https://github.com/owner/repo/pull/2",
		},
		{name: "issue", lane: model.Lane{Issue: &model.Issue{URL: "https://github.com/owner/repo/issues/1"}, Worktree: &model.Worktree{Path: "/tmp/repo"}}, want: "https://github.com/owner/repo/issues/1"},
		{name: "worktree", lane: model.Lane{Worktree: &model.Worktree{Path: "/tmp/repo"}}, want: "/tmp/repo"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &recordingRunner{}
			app := App{Runner: runner}
			if err := app.openLane(context.Background(), test.lane); err != nil {
				t.Fatal(err)
			}
			if runner.target != test.want {
				t.Fatalf("target = %q, want %q", runner.target, test.want)
			}
		})
	}
}

func TestBareMissingConfigInNonTTYIncludesInitHint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	app := App{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &recordingRunner{}, InputIsTTY: func() bool { return false }}
	err := app.runHumanScan(context.Background(), path, "", false, "never", true, false)
	if err == nil || !strings.Contains(err.Error(), "run beacon init") || !strings.Contains(err.Error(), path) {
		t.Fatalf("error = %v", err)
	}
}

func TestBareMissingConfigTTYCanDeclineInit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	prompter := &fakeInitPrompter{confirmations: []bool{false}}
	app := App{
		Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &recordingRunner{},
		InputIsTTY: func() bool { return true }, prompter: prompter,
	}
	err := app.runHumanScan(context.Background(), path, "", false, "never", true, false)
	if err == nil || !strings.Contains(err.Error(), "configuration is required") || prompter.confirmCalls != 1 {
		t.Fatalf("error = %v, confirmations = %d", err, prompter.confirmCalls)
	}
}

type recordingRunner struct{ target string }

func (r *recordingRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	if name != "open" || len(args) != 1 {
		return nil, fmt.Errorf("unexpected command: %s %v", name, args)
	}
	r.target = args[0]
	return nil, nil
}
