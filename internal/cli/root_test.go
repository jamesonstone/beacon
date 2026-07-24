package cli

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
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
	command, _, err = root.Find([]string{"agent", "start"})
	if err != nil || command == nil || command.Name() != "start" {
		t.Fatalf("agent start command = %v, %v", command, err)
	}
}

func TestDirectCLICommandsSelectAgentActivationWithoutLifecycleRecursion(t *testing.T) {
	root := App{}.Root()
	for _, test := range []struct {
		args []string
		want bool
	}{
		{args: nil, want: true},
		{args: []string{"notes", "list"}, want: true},
		{args: []string{"refresh"}, want: true},
		{args: []string{"activity", "prune"}, want: false},
		{args: []string{"integrations", "status", "codex"}, want: false},
		{args: []string{"ollama", "models"}, want: false},
		{args: []string{"ollama", "chat"}, want: false},
		{args: []string{"agent", "start"}, want: false},
		{args: []string{"agent", "stop"}, want: false},
		{args: []string{"doctor"}, want: false},
		{args: []string{"init"}, want: false},
		{args: []string{"projects"}, want: false},
		{args: []string{"scan"}, want: false},
		{args: []string{"config", "init"}, want: false},
		{args: []string{"version"}, want: false},
	} {
		command := root
		if len(test.args) > 0 {
			var err error
			command, _, err = root.Find(test.args)
			if err != nil {
				t.Fatalf("find %v: %v", test.args, err)
			}
		}
		if got := shouldAutoStartAgent(command); got != test.want {
			t.Fatalf("shouldAutoStartAgent(%v) = %t, want %t", test.args, got, test.want)
		}
	}
}

func TestDirectCLIExecutionStartsAgentBestEffortOnMacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("automatic LaunchAgent activation is macOS-only")
	}
	configPath := writeBareDashboardConfig(t)
	starter := &recordingAgentStarter{}
	app := App{
		Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &recordingRunner{},
		autoStartAgent: true,
		agentStarterSource: func(agent.Paths) agentStarter {
			return starter
		},
	}
	command := app.Root()
	command.SetArgs([]string{"--config", configPath, "config", "path"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if starter.calls != 1 {
		t.Fatalf("agent start calls = %d", starter.calls)
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
			wantName, wantArgs, err := openTargetCommand(runtime.GOOS, test.want)
			if err != nil {
				t.Fatal(err)
			}
			if runner.name != wantName || !sameStrings(runner.args, wantArgs) {
				t.Fatalf("opener = %s %v, want %s %v", runner.name, runner.args, wantName, wantArgs)
			}
		})
	}
}

func TestNextActiveLaneNeverFallsBackToIdle(t *testing.T) {
	snapshot := model.Snapshot{
		Groups: model.Groups{Action: []string{"active"}, Idle: []string{"idle"}},
		Lanes: []model.Lane{
			{ID: "idle", Repository: "quiet"},
			{ID: "active", Repository: "needs-action", Worktree: &model.Worktree{Path: "/tmp/active"}},
		},
	}
	lane, ok := nextActiveLane(snapshot)
	if !ok || lane.ID != "active" {
		t.Fatalf("next active lane = %#v, %t", lane, ok)
	}

	snapshot.Groups.Action = nil
	if lane, ok := nextActiveLane(snapshot); ok {
		t.Fatalf("idle-only snapshot returned %#v", lane)
	}
}

func TestBareMissingConfigInNonTTYIncludesInitHint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	app := App{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &recordingRunner{}, InputIsTTY: func() bool { return false }}
	err := app.runHumanScan(context.Background(), path, "", false, "never", false, true, false, false)
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
	err := app.runHumanScan(context.Background(), path, "", false, "never", false, true, false, false)
	if err == nil || !strings.Contains(err.Error(), "configuration is required") || prompter.confirmCalls != 1 {
		t.Fatalf("error = %v, confirmations = %d", err, prompter.confirmCalls)
	}
}

func TestHumanScanCanRevealIdleFollowingProjectsExplicitlyOrByRepository(t *testing.T) {
	repository := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath, `version: 2
repositories:
  - name: quiet
    path: `+repository+`
    github: owner/quiet
`)
	snapshot := model.Snapshot{
		Groups: model.Groups{Idle: []string{"quiet-base"}},
		Lanes: []model.Lane{{
			ID: "quiet-base", Repository: "quiet", GitHub: "owner/quiet", Branch: "quiet-main", NextAction: model.ActionNone,
		}},
	}

	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "bare include idle flag", args: []string{"--include-idle"}},
		{name: "targeted repository", args: []string{"scan", "--repo", "quiet", "--no-refresh"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			app := App{
				Out: &output, Err: &bytes.Buffer{}, Runner: &recordingRunner{},
				OutputIsTTY: func() bool { return false }, TerminalWidth: func() int { return 120 },
				scannerSource: fixedSnapshotScanner{snapshot: snapshot},
			}
			command := app.Root()
			command.SetArgs(append([]string{"--config", configPath, "--color", "never"}, test.args...))
			if err := command.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "Idle Following Projects") || !strings.Contains(output.String(), "quiet-main") {
				t.Fatalf("terminal output = %q", output.String())
			}
		})
	}
}

func TestScanSnapshotUsesPartialReconciliationForRepositoryFilter(t *testing.T) {
	tracker := &recordingProjectTracker{}
	app := App{
		scannerSource: fixedSnapshotScanner{snapshot: model.Snapshot{Projects: []model.Project{}}},
		trackerSource: tracker,
	}
	cfg := config.Config{Path: filepath.Join(t.TempDir(), "config.yaml")}
	if _, err := app.scanSnapshot(context.Background(), cfg, "repo", false); err != nil {
		t.Fatal(err)
	}
	if tracker.partialCalls != 1 || tracker.fullCalls != 0 {
		t.Fatalf("filtered reconciliation: full=%d partial=%d", tracker.fullCalls, tracker.partialCalls)
	}
	if _, err := app.scanSnapshot(context.Background(), cfg, "", false); err != nil {
		t.Fatal(err)
	}
	if tracker.partialCalls != 1 || tracker.fullCalls != 1 {
		t.Fatalf("complete reconciliation: full=%d partial=%d", tracker.fullCalls, tracker.partialCalls)
	}
}

type recordingRunner struct {
	name   string
	args   []string
	target string
}

type recordingAgentStarter struct{ calls int }

func (s *recordingAgentStarter) Start(context.Context) error {
	s.calls++
	return nil
}

type recordingSnapshotScanner struct {
	snapshot model.Snapshot
	calls    int
	refresh  bool
}

func (s *recordingSnapshotScanner) Scan(_ context.Context, _ config.Config, _ string, refresh bool) (model.Snapshot, error) {
	s.calls++
	s.refresh = refresh
	return s.snapshot, nil
}

func (r *recordingRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	switch name {
	case "open", "xdg-open", "rundll32":
	default:
		return nil, fmt.Errorf("unexpected command: %s %v", name, args)
	}
	r.name = name
	r.args = append([]string{}, args...)
	if len(args) > 0 {
		r.target = args[len(args)-1]
	}
	return nil, nil
}
