package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestProjectsUntrackedListsOnlyUntrackedInventory(t *testing.T) {
	output, _, err := executeProjectsCommand(t, false, nil, "projects", "--untracked")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Projects Not Followed") || !strings.Contains(output, "owner/quiet") || strings.Contains(output, "owner/active") {
		t.Fatalf("project output = %q", output)
	}
}

func TestProjectsFollowingViewsSeparateRecentFromQuiet(t *testing.T) {
	for _, test := range []struct {
		flag    string
		title   string
		present string
		absent  string
	}{
		{flag: "--followed", title: "Following Projects", present: "owner/active", absent: "owner/recent"},
		{flag: "--recent", title: "Recently Updated Projects", present: "owner/recent", absent: "owner/quiet"},
		{flag: "--quiet", title: "Quiet Projects", present: "owner/quiet", absent: "owner/recent"},
	} {
		t.Run(test.flag, func(t *testing.T) {
			output, _, err := executeProjectsCommand(t, false, nil, "projects", test.flag)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output, test.title) || !strings.Contains(output, test.present) || strings.Contains(output, test.absent) {
				t.Fatalf("project output = %q", output)
			}
		})
	}
}

func TestProjectsRequiresTTYForInteractiveMode(t *testing.T) {
	_, _, err := executeProjectsCommand(t, false, nil, "projects")
	if err == nil || ExitCode(err) != 2 || !strings.Contains(err.Error(), "requires a TTY") {
		t.Fatalf("error = %v", err)
	}
}

func TestProjectsFollowCommandsAndCompatibilityAliasesUseStableTargets(t *testing.T) {
	for _, test := range []struct {
		command string
		tracked bool
	}{
		{command: "follow", tracked: true},
		{command: "unfollow", tracked: false},
		{command: "track", tracked: true},
		{command: "untrack", tracked: false},
	} {
		t.Run(test.command, func(t *testing.T) {
			_, tracker, err := executeProjectsCommand(t, false, nil, "projects", test.command, "owner/quiet")
			if err != nil {
				t.Fatal(err)
			}
			if tracker.setTracked != test.tracked || !reflect.DeepEqual(tracker.targets, []string{"owner/quiet"}) {
				t.Fatalf("tracker call = %t %#v", tracker.setTracked, tracker.targets)
			}
		})
	}
}

func TestRootFollowCommandsUseSharedProjectAuthority(t *testing.T) {
	for _, test := range []struct {
		command  string
		followed bool
	}{
		{command: "follow", followed: true},
		{command: "unfollow", followed: false},
	} {
		t.Run(test.command, func(t *testing.T) {
			_, tracker, err := executeProjectsCommand(t, false, nil, test.command, "owner/quiet")
			if err != nil {
				t.Fatal(err)
			}
			if tracker.setTracked != test.followed || !reflect.DeepEqual(tracker.targets, []string{"owner/quiet"}) {
				t.Fatalf("tracker call = %t %#v", tracker.setTracked, tracker.targets)
			}
		})
	}
}

func TestProjectsInteractiveSelectionConfirmsAndPersists(t *testing.T) {
	prompter := &fakeProjectPrompter{selected: []string{"owner/quiet"}, confirmed: true}
	_, tracker, err := executeProjectsCommand(t, true, prompter, "select")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tracker.selection, []string{"owner/quiet"}) || prompter.confirmTitle == "" {
		t.Fatalf("selection = %#v, confirmation = %q", tracker.selection, prompter.confirmTitle)
	}
}

func TestSelectCommandUsesSharedInteractiveSelection(t *testing.T) {
	prompter := &fakeProjectPrompter{selected: []string{}, confirmed: true}
	output, tracker, err := executeProjectsCommand(t, true, prompter, "select")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracker.selection) != 0 || !strings.Contains(output, "0 followed, 3 outside Following") {
		t.Fatalf("selection=%#v output=%q", tracker.selection, output)
	}
}

func TestProjectSnapshotTreatsAgentCacheAsPartialInventory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	configPath := filepath.Join(root, "config.yaml")
	repositoryPath := filepath.Join(root, "repo")
	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestConfig(t, configPath, `version: 2
repositories:
  - name: repo
    path: `+repositoryPath+`
    github: owner/repo
`)
	paths, err := agent.ResolvePaths(configPath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	snapshot := model.Snapshot{
		SchemaVersion: model.SchemaVersion, ConfigPath: configPath, GeneratedAt: now,
		Projects: []model.Project{{Name: "repo", Path: repositoryPath, GitHub: "owner/repo"}},
		Lanes:    []model.Lane{}, Errors: []model.ScanError{}, Warnings: []model.ScanError{},
	}
	if err := (agent.Cache{Directory: paths.Projects}).Write(agent.ProjectRecord{
		Version: agent.CacheVersion, ProjectID: "owner/repo", Revision: 1,
		UpdatedAt: now, Snapshot: snapshot,
	}); err != nil {
		t.Fatal(err)
	}
	tracker := &recordingProjectTracker{}
	app := App{trackerSource: tracker, Runner: &recordingRunner{}}
	if _, err := app.projectSnapshot(context.Background(), configPath); err != nil {
		t.Fatal(err)
	}
	if tracker.partialCalls != 1 || tracker.fullCalls != 0 {
		t.Fatalf("cache reconciliation: full=%d partial=%d", tracker.fullCalls, tracker.partialCalls)
	}
}

func TestSelectCommandRequiresTTY(t *testing.T) {
	_, _, err := executeProjectsCommand(t, false, nil, "select")
	if err == nil || ExitCode(err) != 2 || !strings.Contains(err.Error(), "beacon select requires a TTY") {
		t.Fatalf("error = %v", err)
	}
}

func TestProjectMutationUsesLocalCacheWhenAgentIsUnavailable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	repository := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")
	writeTestConfig(t, configPath, `version: 2
repositories:
  - name: cached
    path: `+repository+`
    github: owner/cached
`)
	paths, err := agent.ResolvePaths(configPath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	snapshot := model.Snapshot{
		SchemaVersion: model.SchemaVersion, GeneratedAt: now, ConfigPath: configPath,
		Projects: []model.Project{{Name: "cached", Path: repository, GitHub: "owner/cached", TrackingState: model.TrackingTracked}},
		Lanes:    []model.Lane{}, Refresh: []model.Refresh{}, Errors: []model.ScanError{}, Warnings: []model.ScanError{},
	}
	if err := (agent.Cache{Directory: paths.Projects}).Write(agent.ProjectRecord{
		Version: agent.CacheVersion, ProjectID: "owner/cached", Revision: 1,
		Stage: "ready", UpdatedAt: now, Snapshot: snapshot,
	}); err != nil {
		t.Fatal(err)
	}
	tracker := &recordingProjectTracker{}
	app := App{Runner: failingCommandRunner{}, trackerSource: tracker}
	if err := app.setProjects(context.Background(), configPath, []string{"owner/cached"}, false); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tracker.targets, []string{"owner/cached"}) || tracker.setTracked {
		t.Fatalf("tracker call = %t %#v", tracker.setTracked, tracker.targets)
	}
}

func executeProjectsCommand(t *testing.T, tty bool, prompter projectPrompter, args ...string) (string, *recordingProjectTracker, error) {
	t.Helper()
	repository := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath, `version: 2
repositories:
  - name: active
    path: `+repository+`
    github: owner/active
`)
	snapshot := model.Snapshot{
		Projects: []model.Project{
			{Name: "active", Path: repository, GitHub: "owner/active", TrackingState: model.TrackingTracked, FollowState: model.FollowFollowing},
			{Name: "recent", Path: repository, GitHub: "owner/recent", TrackingState: model.TrackingUntracked, FollowState: model.FollowRecent, LastActivityAt: time.Date(2026, 7, 12, 12, 30, 0, 0, time.UTC), ActivityReason: "new GitHub activity"},
			{Name: "quiet", Path: repository, GitHub: "owner/quiet", TrackingState: model.TrackingUntracked, FollowState: model.FollowQuiet},
		},
		Groups: model.Groups{Ready: []string{}, Action: []string{}, Waiting: []string{}, Idle: []string{}, Untracked: []string{}},
		Lanes:  []model.Lane{}, Errors: []model.ScanError{}, Warnings: []model.ScanError{},
		Summary: model.Summary{Projects: 3, FollowingProjects: 1, RecentProjects: 1, QuietProjects: 1, TrackedProjects: 1, UntrackedProjects: 2},
	}
	tracker := &recordingProjectTracker{}
	var output bytes.Buffer
	app := App{
		Out: &output, Err: &bytes.Buffer{}, Runner: &recordingRunner{},
		InputIsTTY: func() bool { return tty }, OutputIsTTY: func() bool { return false },
		TerminalWidth: func() int { return 120 }, scannerSource: fixedSnapshotScanner{snapshot: snapshot},
		trackerSource: tracker, projectPrompterSource: prompter,
	}
	command := app.Root()
	command.SetArgs(append([]string{"--config", configPath, "--color", "never"}, args...))
	err := command.ExecuteContext(context.Background())
	return output.String(), tracker, err
}

type recordingProjectTracker struct {
	targets      []string
	setTracked   bool
	selection    []string
	fullCalls    int
	partialCalls int
}

func (t *recordingProjectTracker) Reconcile(snapshot model.Snapshot) (model.Snapshot, error) {
	t.fullCalls++
	return snapshot, nil
}

func (t *recordingProjectTracker) ReconcilePartial(snapshot model.Snapshot) (model.Snapshot, error) {
	t.partialCalls++
	return snapshot, nil
}

func (t *recordingProjectTracker) SetTracked(snapshot model.Snapshot, targets []string, tracked bool) (model.Snapshot, error) {
	t.targets = append([]string{}, targets...)
	t.setTracked = tracked
	return snapshot, nil
}

func (t *recordingProjectTracker) SetSelection(snapshot model.Snapshot, selected []string) (model.Snapshot, error) {
	t.selection = append([]string{}, selected...)
	return snapshot, nil
}

type fakeProjectPrompter struct {
	selected     []string
	confirmed    bool
	confirmTitle string
}

type failingCommandRunner struct{}

func (failingCommandRunner) Run(context.Context, string, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected external command")
}

func (p *fakeProjectPrompter) SelectFollowedProjects(context.Context, []model.Project) ([]string, error) {
	return append([]string{}, p.selected...), nil
}

func (p *fakeProjectPrompter) Confirm(_ context.Context, title string) (bool, error) {
	p.confirmTitle = title
	return p.confirmed, nil
}
