package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
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

func TestTerminalReplacesUntrackedInventoryWithManagementHint(t *testing.T) {
	snapshot := model.Snapshot{
		GeneratedAt: time.Now(),
		Summary:     model.Summary{UntrackedProjects: 2, Warnings: 1, Errors: 2},
		Groups:      model.Groups{Untracked: []string{"one", "two"}},
		Projects: []model.Project{
			{Name: "one", GitHub: "owner/one", TrackingState: model.TrackingUntracked},
			{Name: "two", GitHub: "owner/two", TrackingState: model.TrackingUntracked},
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
	if !strings.Contains(buffer.String(), "2 untracked projects · run beacon projects --untracked to view") {
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

func TestTerminalHidesQuietProjectsUntilRequested(t *testing.T) {
	snapshot := quietProjectSnapshot()

	var defaultOutput bytes.Buffer
	if err := TerminalWithOptions(&defaultOutput, snapshot, TerminalOptions{Width: 120}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(defaultOutput.String(), "1 quiet project hidden · use --include-idle to show") {
		t.Fatalf("default terminal output = %q", defaultOutput.String())
	}
	if strings.Contains(defaultOutput.String(), "quiet-main") || strings.Contains(defaultOutput.String(), "active-main") || strings.Contains(defaultOutput.String(), "Quiet Projects") {
		t.Fatalf("default terminal exposed idle inventory: %q", defaultOutput.String())
	}

	var completeOutput bytes.Buffer
	if err := TerminalWithOptions(&completeOutput, snapshot, TerminalOptions{Width: 120, IncludeIdle: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(completeOutput.String(), "Quiet Projects") || !strings.Contains(completeOutput.String(), "quiet-main") {
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

func TestTerminalColorAndNarrowLayout(t *testing.T) {
	snapshot := model.Snapshot{
		GeneratedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		Summary:     model.Summary{Projects: 1, Total: 1, NeedsAction: 1},
		Groups:      model.Groups{Action: []string{"lane"}},
		Lanes: []model.Lane{{
			ID: "lane", Repository: "example", Branch: "feature", NextAction: model.ActionFixCI,
			Signals: model.Signals{Worktree: model.WorktreeClean, CI: model.CIFailure, Review: model.ReviewNone, Freshness: model.FreshnessCurrent},
		}},
	}
	var colored bytes.Buffer
	if err := TerminalWithOptions(&colored, snapshot, TerminalOptions{Color: true, Width: 120}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(colored.String(), "\x1b[") {
		t.Fatalf("colored terminal output has no ANSI escapes: %q", colored.String())
	}

	var narrow bytes.Buffer
	if err := TerminalWithOptions(&narrow, snapshot, TerminalOptions{Color: false, Width: 60}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(narrow.String(), "\x1b[") || !strings.Contains(narrow.String(), "Next: fix failing CI") || strings.Contains(narrow.String(), "PROJECT\t") {
		t.Fatalf("narrow terminal output = %q", narrow.String())
	}
}

func TestNarrowTerminalWrapsLongUnicodeAndANSIContent(t *testing.T) {
	snapshot := model.Snapshot{
		GeneratedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		Summary:     model.Summary{Projects: 1, Total: 1, NeedsAction: 1},
		Groups:      model.Groups{Action: []string{"lane"}},
		Lanes: []model.Lane{{
			ID: "lane", Repository: "project-測試-🛰️", Branch: "feature", NextAction: model.ActionAddressReview,
			Issue:    &model.Issue{Number: 42, Title: "A deliberately long Unicode work item 測試 that must wrap without splitting the terminal layout"},
			Progress: &model.Progress{Phase: "implement", Feature: "long-dashboard-feature", Summary: "Evidence with enough words to wrap safely across several visible terminal lines."},
			Signals:  model.Signals{Worktree: model.WorktreeClean, CI: model.CISuccess, Review: model.ReviewFeedbackPending, Freshness: model.FreshnessCurrent},
		}},
	}
	var buffer bytes.Buffer
	if err := TerminalWithOptions(&buffer, snapshot, TerminalOptions{Color: true, Width: 48}); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(buffer.String(), "\n"), "\n") {
		if width := lipgloss.Width(line); width > 48 {
			t.Fatalf("visible width %d exceeds 48: %q\n%s", width, line, buffer.String())
		}
	}
}

func TestVeryNarrowTerminalHonorsReportedWidth(t *testing.T) {
	snapshot := model.Snapshot{
		GeneratedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		Summary:     model.Summary{Projects: 1, Total: 1, NeedsAction: 1},
		Groups:      model.Groups{Action: []string{"lane"}},
		Lanes: []model.Lane{{
			ID: "lane", Repository: "long-project", Branch: "long-feature-branch", NextAction: model.ActionInspectLocal,
			Signals: model.Signals{Worktree: model.WorktreeDirty, CI: model.CINone, Review: model.ReviewNone, Freshness: model.FreshnessCurrent},
		}},
		Errors: []model.ScanError{{Repository: "long-project", Stage: "discovery", Message: "a long scoped diagnostic that must also wrap"}},
	}
	var buffer bytes.Buffer
	if err := TerminalWithOptions(&buffer, snapshot, TerminalOptions{Color: true, Width: 20}); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(buffer.String(), "\n"), "\n") {
		if width := lipgloss.Width(line); width > 20 {
			t.Fatalf("visible width %d exceeds 20: %q\n%s", width, line, buffer.String())
		}
	}
}

func TestJSONNeverContainsANSI(t *testing.T) {
	snapshot := model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		Projects:      []model.Project{},
		Groups:        model.Groups{Ready: []string{}, Action: []string{}, Waiting: []string{}, Idle: []string{}},
		Lanes:         []model.Lane{{ID: "\x1b[31munsafe"}},
		Refresh:       []model.Refresh{}, Errors: []model.ScanError{}, Warnings: []model.ScanError{},
	}
	var buffer bytes.Buffer
	if err := JSON(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buffer.String(), "\x1b[") {
		t.Fatalf("JSON contains literal ANSI escape: %q", buffer.String())
	}
}

func quietProjectSnapshot() model.Snapshot {
	return model.Snapshot{
		GeneratedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		Summary:     model.Summary{Projects: 2, Total: 3, NeedsAction: 1, Idle: 2},
		Groups: model.Groups{
			Action: []string{"active-work"},
			Idle:   []string{"active-base", "quiet-base"},
		},
		Lanes: []model.Lane{
			{ID: "active-work", Repository: "active", GitHub: "owner/active", Branch: "feature", NextAction: model.ActionFixCI},
			{ID: "active-base", Repository: "active", GitHub: "owner/active", Branch: "active-main", NextAction: model.ActionNone},
			{ID: "quiet-base", Repository: "quiet", GitHub: "owner/quiet", Branch: "quiet-main", NextAction: model.ActionNone},
		},
	}
}
