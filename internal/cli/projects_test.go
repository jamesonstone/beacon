package cli

import (
	"bytes"
	"context"
	"errors"
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
	if !strings.Contains(output, "Untracked Projects") || !strings.Contains(output, "owner/quiet") || strings.Contains(output, "owner/active") {
		t.Fatalf("project output = %q", output)
	}
}

func TestProjectsRequiresTTYForInteractiveMode(t *testing.T) {
	_, _, err := executeProjectsCommand(t, false, nil, "projects")
	if err == nil || ExitCode(err) != 2 || !strings.Contains(err.Error(), "requires a TTY") {
		t.Fatalf("error = %v", err)
	}
}

func TestProjectsExplicitTrackAndUntrackUseStableTargets(t *testing.T) {
	for _, test := range []struct {
		command string
		tracked bool
	}{
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

func TestProjectsInteractiveSelectionConfirmsAndPersists(t *testing.T) {
	prompter := &fakeProjectPrompter{selected: []string{"owner/quiet"}, confirmed: true}
	_, tracker, err := executeProjectsCommand(t, true, prompter, "projects")
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
	if len(tracker.selection) != 0 || !strings.Contains(output, "0 tracked, 2 untracked") {
		t.Fatalf("selection=%#v output=%q", tracker.selection, output)
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
			{Name: "active", Path: repository, GitHub: "owner/active", TrackingState: model.TrackingTracked},
			{Name: "quiet", Path: repository, GitHub: "owner/quiet", TrackingState: model.TrackingUntracked},
		},
		Groups: model.Groups{Ready: []string{}, Action: []string{}, Waiting: []string{}, Idle: []string{}, Untracked: []string{}},
		Lanes:  []model.Lane{}, Errors: []model.ScanError{}, Warnings: []model.ScanError{},
		Summary: model.Summary{Projects: 1, UntrackedProjects: 1},
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
	targets    []string
	setTracked bool
	selection  []string
}

func (t *recordingProjectTracker) Reconcile(snapshot model.Snapshot) (model.Snapshot, error) {
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

func (p *fakeProjectPrompter) SelectTrackedProjects(context.Context, []model.Project) ([]string, error) {
	return append([]string{}, p.selected...), nil
}

func (p *fakeProjectPrompter) Confirm(_ context.Context, title string) (bool, error) {
	p.confirmTitle = title
	return p.confirmed, nil
}
