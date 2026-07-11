package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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

func (p *fakeProjectPrompter) SelectTrackedProjects(context.Context, []model.Project) ([]string, error) {
	return append([]string{}, p.selected...), nil
}

func (p *fakeProjectPrompter) Confirm(_ context.Context, title string) (bool, error) {
	p.confirmTitle = title
	return p.confirmed, nil
}
