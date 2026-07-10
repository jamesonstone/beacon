package progress

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingSummaryReturnsNoEvidence(t *testing.T) {
	result := Load(t.TempDir())
	if len(result.Features) != 0 || result.Selected != nil || len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestLoadParsesFeaturesSpecsAndSelectsHighestActive(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary(
		"| 0001 | shipped | `docs/specs/0001-shipped` | deliver | no | 2026-01-01 | Shipped. |\n"+
			"| 0002 | paused | `docs/specs/0002-paused` | implement | yes | 2026-01-02 | Paused. |\n"+
			"| 0003 | active | `docs/specs/0003-active` | validate | no | 2026-01-03 | Active. |",
		featureSummary("shipped", "deliver", "no")+featureSummary("paused", "implement", "yes")+featureSummary("active", "validate", "no"),
	))
	writeSpec(t, root, "0001-shipped", "0001", "shipped", "deliver", "https://github.com/acme/project/issues/7")
	writeSpec(t, root, "0002-paused", "0002", "paused", "implement", "https://github.com/acme/project/issues/8")
	writeSpec(t, root, "0003-active", "0003", "active", "validate", "https://github.com/acme/project/issues/7")

	result := Load(root)
	if result.Selected == nil || result.Selected.ID != "0003" {
		t.Fatalf("selected = %#v", result.Selected)
	}
	if len(result.Features) != 3 || result.Features[2].Intent != "Intent for active." {
		t.Fatalf("features = %#v", result.Features)
	}
	if len(result.IssueLinks) != 2 || strings.Join(result.IssueLinks[0].FeatureIDs, ",") != "0001,0003" {
		t.Fatalf("issue links = %#v", result.IssueLinks)
	}
	if matches := result.FeaturesForIssueURL("https://github.com/acme/project/issues/7"); len(matches) != 2 {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestLoadFallsBackToHighestDeliveredFeature(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary(
		"| 0001 | old | `docs/specs/0001-old` | deliver | no | 2026-01-01 | Old. |\n"+
			"| 0002 | latest | `docs/specs/0002-latest` | deliver | no | 2026-01-02 | Latest. |",
		featureSummary("old", "deliver", "no")+featureSummary("latest", "deliver", "no"),
	))
	writeSpec(t, root, "0001-old", "0001", "old", "deliver", "")
	writeSpec(t, root, "0002-latest", "0002", "latest", "deliver", "")

	result := Load(root)
	if result.Selected == nil || result.Selected.ID != "0002" {
		t.Fatalf("selected = %#v", result.Selected)
	}
}

func TestLoadIgnoresPausedActiveFeatureWhenSelectingFallback(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary(
		"| 0001 | delivered | `docs/specs/0001-delivered` | deliver | no | 2026-01-01 | Done. |\n"+
			"| 0002 | paused | `docs/specs/0002-paused` | implement | yes | 2026-01-02 | Paused. |",
		featureSummary("delivered", "deliver", "no")+featureSummary("paused", "implement", "yes"),
	))
	writeSpec(t, root, "0001-delivered", "0001", "delivered", "deliver", "")
	writeSpec(t, root, "0002-paused", "0002", "paused", "implement", "")

	result := Load(root)
	if result.Selected == nil || result.Selected.ID != "0001" {
		t.Fatalf("selected = %#v", result.Selected)
	}
}

func TestLoadUsesCanonicalSpecPhaseAndReportsStaleness(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary(
		"| 0001 | feature | `docs/specs/0001-feature` | implement | no | 2026-01-01 | Work. |",
		featureSummary("feature", "implement", "no"),
	))
	writeSpec(t, root, "0001-feature", "0001", "feature", "deliver", "https://github.com/acme/project/issues/1")

	result := Load(root)
	if result.Features[0].Phase != "deliver" {
		t.Fatalf("phase = %q", result.Features[0].Phase)
	}
	assertDiagnosticContains(t, result.Diagnostics, "using SPEC")
}

func TestLoadUsesSummaryPhaseWhenCurrentSpecOmitsPhase(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary(
		"| 0035 | loop-review | `docs/specs/0035-loop-review` | reflect | no | 2026-01-01 | Reflected. |",
		featureSummary("loop-review", "reflect", "no"),
	))
	writeProgressFile(t, root, "docs/specs/0035-loop-review/SPEC.md", `---
kit_metadata_version: 1
artifact: spec
feature:
  id: 0035
  slug: loop-review
  dir: 0035-loop-review
references:
  - ../../legacy-reference.md
  - type: github-issue
    target: https://github.com/acme/project/issues/35
---
# Feature
`)

	result := Load(root)
	if len(result.Diagnostics) != 0 || len(result.Features) != 1 || result.Features[0].Phase != "reflect" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Features[0].IssueURLs) != 1 || result.Features[0].IssueURLs[0] != "https://github.com/acme/project/issues/35" {
		t.Fatalf("issue URLs = %#v", result.Features[0].IssueURLs)
	}
}

func TestLoadAcceptsEmptyCanonicalProgressTable(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary("", "none"))
	result := Load(root)
	if len(result.Features) != 0 || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestLoadDoesNotWarnForRemovedFeatureWithoutSpec(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary(
		"| 0036 | removed-feature | `docs/specs/0036-removed-feature` | removed | no | 2026-01-01 | Removed. |",
		featureSummary("removed-feature", "removed", "no"),
	))
	result := Load(root)
	if len(result.Diagnostics) != 0 || result.Selected != nil {
		t.Fatalf("result = %#v", result)
	}
}

func TestLoadRetainsValidEvidenceWhenDocumentsAreMalformed(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary(
		"| nope | broken | `../../outside` | implement | maybe | today | Broken. |\n"+
			"| 0002 | valid | `docs/specs/0002-valid` | implement | no | 2026-01-02 | Valid. |",
		featureSummary("valid", "implement", "no"),
	))
	writeProgressFile(t, root, "docs/specs/0002-valid/SPEC.md", "---\nartifact: spec\nfeature: [\n---\n")

	result := Load(root)
	if len(result.Features) != 1 || result.Features[0].ID != "0002" {
		t.Fatalf("features = %#v", result.Features)
	}
	if len(result.Diagnostics) < 2 {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func TestLoadRejectsUnsafeSummaryPathWithoutDiscardingFeature(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary(
		"| 0001 | unsafe | `../../outside` | implement | no | 2026-01-01 | Unsafe. |",
		featureSummary("unsafe", "implement", "no"),
	))

	result := Load(root)
	if len(result.Features) != 1 || result.Features[0].SpecPath != "" {
		t.Fatalf("features = %#v", result.Features)
	}
	assertDiagnosticContains(t, result.Diagnostics, "unsafe path")
}

func TestLoadAddsUnlistedSpecAndRejectsMalformedIssueURL(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, projectSummary(
		"| 0001 | listed | `docs/specs/0001-listed` | deliver | no | 2026-01-01 | Listed. |",
		featureSummary("listed", "deliver", "no"),
	))
	writeSpec(t, root, "0001-listed", "0001", "listed", "deliver", "")
	writeSpec(t, root, "0002-unlisted", "0002", "unlisted", "implement", "https://github.com/acme/project/issues/2?query=yes")

	result := Load(root)
	if len(result.Features) != 2 || result.Selected == nil || result.Selected.ID != "0002" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Features[1].IssueURLs) != 0 {
		t.Fatalf("issue URLs = %#v", result.Features[1].IssueURLs)
	}
	assertDiagnosticContains(t, result.Diagnostics, "not listed")
	assertDiagnosticContains(t, result.Diagnostics, "malformed GitHub issue URL")
}

func TestLoadMalformedSummaryReturnsScopedError(t *testing.T) {
	root := t.TempDir()
	writeProgressFile(t, root, SummaryPath, "# no progress table\n")
	writeProgressFile(t, root, "docs/specs/0001-invalid/SPEC.md", "not front matter\n")
	result := Load(root)
	if !result.HasErrors() || len(result.Features) != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func projectSummary(rows, summaries string) string {
	return "# PROJECT PROGRESS SUMMARY\n\n## FEATURE PROGRESS TABLE\n\n" +
		"| ID | FEATURE | PATH | PHASE | PAUSED | CREATED | SUMMARY |\n" +
		"| -- | ------- | ---- | ----- | ------ | ------- | ------- |\n" + rows + "\n\n## FEATURE SUMMARIES\n\n" + summaries + "\n## LAST UPDATED\n\n2026-01-01 UTC\n"
}

func featureSummary(slug, status, paused string) string {
	return "### " + slug + "\n\n" +
		"- **STATUS**: " + status + "\n" +
		"- **PAUSED**: " + paused + "\n" +
		"- **INTENT**: Intent for " + slug + ".\n" +
		"- **APPROACH**: Approach for " + slug + ".\n" +
		"- **OPEN ITEMS**: Open items for " + slug + ".\n\n"
}

func writeSpec(t *testing.T, root, dir, id, slug, phase, issueURL string) {
	t.Helper()
	reference := ""
	if issueURL != "" {
		reference = "references:\n  - type: github-issue\n    target: " + issueURL + "\n"
	}
	contents := "---\nartifact: spec\nphase: " + phase + "\nfeature:\n  id: \"" + id + "\"\n  slug: " + slug + "\n  dir: " + dir + "\n" + reference + "---\n\n# Feature\n"
	writeProgressFile(t, root, filepath.Join("docs/specs", dir, "SPEC.md"), contents)
}

func writeProgressFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertDiagnosticContains(t *testing.T, diagnostics []Diagnostic, substring string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, substring) {
			return
		}
	}
	t.Fatalf("diagnostics %#v do not contain %q", diagnostics, substring)
}
