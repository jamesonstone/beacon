package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestJSONEmitsVersionedSnapshot(t *testing.T) {
	snapshot := model.Snapshot{SchemaVersion: model.SchemaVersion, GeneratedAt: time.Now(), Groups: model.Groups{}, Projects: []model.Project{}, Lanes: []model.Lane{}, Errors: []model.ScanError{}, Warnings: []model.ScanError{}}
	var buffer bytes.Buffer
	if err := JSON(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["schema_version"] != float64(model.SchemaVersion) {
		t.Fatalf("JSON = %s", buffer.String())
	}
}

func TestJSONEmitsFollowingStateWithoutZeroActivityTimestamp(t *testing.T) {
	snapshot := model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		Projects: []model.Project{{
			Name: "repo", GitHub: "owner/repo", FollowState: model.FollowFollowing,
		}},
	}
	var buffer bytes.Buffer
	if err := JSON(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buffer.String(), `"follow_state": "following"`) || strings.Contains(buffer.String(), "last_activity_at") {
		t.Fatalf("JSON = %s", buffer.String())
	}
}

func TestJSONEmitsEmptyCollectionsAsArrays(t *testing.T) {
	snapshot := model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		Refresh:       []model.Refresh{},
		Tracking:      model.Tracking{AutoReactivated: []string{}},
		Groups:        model.Groups{Ready: []string{}, Action: []string{}, Waiting: []string{}, Idle: []string{}, Untracked: []string{}},
		Projects:      []model.Project{},
		Lanes:         []model.Lane{},
		Errors:        []model.ScanError{},
		Warnings:      []model.ScanError{},
	}
	var buffer bytes.Buffer
	if err := JSON(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"refresh": []`, `"ready": []`, `"untracked": []`, `"auto_reactivated": []`, `"projects": []`, `"lanes": []`, `"errors": []`, `"warnings": []`} {
		if !strings.Contains(buffer.String(), expected) {
			t.Fatalf("JSON missing %s: %s", expected, buffer.String())
		}
	}
}

func TestTerminalReplacesOutsideInventoryWithFollowingHints(t *testing.T) {
	snapshot := model.Snapshot{
		GeneratedAt: time.Now(),
		Summary:     model.Summary{UntrackedProjects: 2, RecentProjects: 1, QuietProjects: 1, Warnings: 1, Errors: 2},
		Groups:      model.Groups{Untracked: []string{"one", "two"}},
		Projects: []model.Project{
			{Name: "one", GitHub: "owner/one", TrackingState: model.TrackingUntracked, FollowState: model.FollowRecent},
			{Name: "two", GitHub: "owner/two", TrackingState: model.TrackingUntracked, FollowState: model.FollowQuiet},
		},
		Lanes: []model.Lane{
			{ID: "one", Repository: "one", NextAction: model.ActionFixCI},
			{ID: "two", Repository: "two", NextAction: model.ActionNone},
		},
		Errors: []model.ScanError{
			{Repository: "one", Stage: "github", Message: "hidden project error"},
			{Stage: "github-search", Message: "visible global error"},
		},
		Warnings: []model.ScanError{{Repository: "owner/two", Stage: "progress", Message: "hidden warning"}},
	}
	var buffer bytes.Buffer
	if err := TerminalWithOptions(&buffer, snapshot, TerminalOptions{Width: 120}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buffer.String(), "1 recently updated project outside Following · run beacon projects --recent to view") || !strings.Contains(buffer.String(), "1 quiet project outside Following · run beacon projects --quiet to view") {
		t.Fatalf("terminal output = %q", buffer.String())
	}
	if strings.Contains(buffer.String(), "one\t") || strings.Contains(buffer.String(), "two\t") {
		t.Fatalf("terminal exposed untracked lanes: %q", buffer.String())
	}
	if strings.Contains(buffer.String(), "hidden project error") || strings.Contains(buffer.String(), "hidden warning") || strings.Contains(buffer.String(), "1 warnings") {
		t.Fatalf("terminal exposed untracked diagnostics: %q", buffer.String())
	}
	if !strings.Contains(buffer.String(), "visible global error") {
		t.Fatalf("terminal hid a global diagnostic: %q", buffer.String())
	}
}

func TestTerminalGroupsLanes(t *testing.T) {
	snapshot := model.Snapshot{
		GeneratedAt: time.Now(), Groups: model.Groups{Ready: []string{"lane"}},
		Lanes: []model.Lane{{ID: "lane", Repository: "example", Branch: "feature", ReviewReady: true, NextAction: model.ActionReviewPR}},
	}
	var buffer bytes.Buffer
	if err := Terminal(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buffer.String(), "Ready for Review") || !strings.Contains(buffer.String(), "review manually") {
		t.Fatalf("terminal output = %q", buffer.String())
	}
}

func TestTerminalHidesIdleFollowingProjectsUntilRequested(t *testing.T) {
	snapshot := quietProjectSnapshot()

	var defaultOutput bytes.Buffer
	if err := TerminalWithOptions(&defaultOutput, snapshot, TerminalOptions{Width: 120}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(defaultOutput.String(), "1 idle following project hidden · use --include-idle to show") {
		t.Fatalf("default terminal output = %q", defaultOutput.String())
	}
	if strings.Contains(defaultOutput.String(), "quiet-main") || strings.Contains(defaultOutput.String(), "active-main") || strings.Contains(defaultOutput.String(), "Idle Following Projects") {
		t.Fatalf("default terminal exposed idle inventory: %q", defaultOutput.String())
	}

	var completeOutput bytes.Buffer
	if err := TerminalWithOptions(&completeOutput, snapshot, TerminalOptions{Width: 120, IncludeIdle: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(completeOutput.String(), "Idle Following Projects") || !strings.Contains(completeOutput.String(), "quiet-main") {
		t.Fatalf("included terminal output = %q", completeOutput.String())
	}
	if strings.Contains(completeOutput.String(), "active-main") {
		t.Fatalf("included terminal exposed redundant active-project base lane: %q", completeOutput.String())
	}
}

func TestJSONPreservesIdleLanesHiddenFromHumanOutput(t *testing.T) {
	snapshot := quietProjectSnapshot()
	var buffer bytes.Buffer
	if err := JSON(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	for _, id := range snapshot.Groups.Idle {
		if !strings.Contains(buffer.String(), id) {
			t.Fatalf("JSON omitted idle lane %q: %s", id, buffer.String())
		}
	}
}

func TestTerminalSummarizesWarningsWithoutRenderingDiagnosticFlood(t *testing.T) {
	snapshot := model.Snapshot{
		GeneratedAt: time.Now(), Summary: model.Summary{Warnings: 2},
		Groups: model.Groups{}, Warnings: []model.ScanError{
			{Repository: "first", Stage: "progress-warning", Message: "legacy progress document"},
			{Repository: "second", Stage: "discovery-inspect", Message: "repository unavailable"},
		},
	}
	var buffer bytes.Buffer
	if err := Terminal(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buffer.String(), "2 warnings") {
		t.Fatalf("terminal output = %q", buffer.String())
	}
	if strings.Contains(buffer.String(), "legacy progress document") || strings.Contains(buffer.String(), "repository unavailable") || strings.Contains(buffer.String(), "Errors") {
		t.Fatalf("terminal rendered non-blocking diagnostics as errors: %q", buffer.String())
	}
}
