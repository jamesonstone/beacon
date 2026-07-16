package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jamesonstone/beacon/internal/model"
)

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

func TestRelativeAgeUsesStableHumanUnits(t *testing.T) {
	now := time.Date(2026, 7, 12, 16, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name    string
		updated time.Time
		want    string
	}{
		{name: "unknown", want: "activity unknown"},
		{name: "now", updated: now.Add(-20 * time.Second), want: "active now"},
		{name: "minutes", updated: now.Add(-12 * time.Minute), want: "active 12m ago"},
		{name: "hours", updated: now.Add(-5 * time.Hour), want: "active 5h ago"},
		{name: "days", updated: now.Add(-72 * time.Hour), want: "active 3d ago"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := relativeAge(now, test.updated); got != test.want {
				t.Fatalf("relativeAge() = %q, want %q", got, test.want)
			}
		})
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
