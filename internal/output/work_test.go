package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestWorkTerminalRendersConciseActiveSummaryAndDiagnostics(t *testing.T) {
	scan := model.WorkScan{
		SchemaVersion: model.WorkScanSchemaVersion,
		GeneratedAt:   time.Date(2026, 7, 24, 14, 0, 0, 0, time.UTC),
		Summary: model.WorkScanSummary{
			Projects: 3, ActiveProjects: 2, WorkItems: 2, IdleProjects: 1,
		},
		Items: []model.WorkItem{
			{
				Repository: "beacon", Branch: "GH-76", State: model.WorkDirty,
				NextAction: model.ActionInspectLocal,
			},
			{
				Repository: "kit", Branch: "GH-64", State: model.WorkPullRequest,
				NextAction: model.ActionReviewPR,
				PullRequest: &model.WorkPullRequestSummary{
					Number: 65, Title: "Feature", URL: "https://example.test/65",
				},
			},
		},
		Warnings: []model.ScanError{{Repository: "kit", Stage: "fetch", Message: "offline"}},
		Errors:   []model.ScanError{{Repository: "beacon", Stage: "github", Message: "unavailable"}},
	}
	var output bytes.Buffer
	if err := WorkTerminal(&output, scan, TerminalOptions{Width: 120}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, expected := range []string{
		"Beacon v2", "3 projects · 2 active · 2 work items", "beacon", "GH-76",
		"local changes", "inspect local changes", "PR #65 · GH-64", "open PR",
		"1 idle project hidden", "Warnings", "kit: fetch: offline",
		"Errors", "beacon: github: unavailable",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("output missing %q:\n%s", expected, text)
		}
	}
}

func TestWorkTerminalShowsIdleProjectsWhenIncluded(t *testing.T) {
	scan := model.WorkScan{
		Summary: model.WorkScanSummary{Projects: 1, IdleProjects: 1},
		Items: []model.WorkItem{{
			Repository: "quiet", Branch: "main", Base: "main", State: model.WorkIdle,
		}},
	}
	var output bytes.Buffer
	if err := WorkTerminal(&output, scan, TerminalOptions{Width: 60, IncludeIdle: true}); err != nil {
		t.Fatal(err)
	}
	if text := output.String(); !strings.Contains(text, "quiet") || !strings.Contains(text, "main · idle") ||
		strings.Contains(text, "hidden") {
		t.Fatalf("output = %q", text)
	}
}
