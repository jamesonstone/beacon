package progress

import "strings"

const SummaryPath = "docs/PROJECT_PROGRESS_SUMMARY.md"

type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Diagnostic describes progress evidence that could not be trusted. Progress
// documents are optional, so callers can surface diagnostics without failing a
// repository scan.
type Diagnostic struct {
	Severity Severity
	Path     string
	Message  string
}

// Feature is the neutral, evidence-backed representation of one Kit feature.
// Fields not present in the repository's progress documents remain empty.
type Feature struct {
	ID        string
	Slug      string
	Path      string
	Phase     string
	Paused    bool
	Created   string
	Summary   string
	Intent    string
	Approach  string
	OpenItems string
	SpecPath  string
	IssueURLs []string
	Listed    bool
}

// IssueLink retains the many-to-many relationship between GitHub issues and
// features. Multiple features may intentionally share one delivery issue.
type IssueLink struct {
	URL        string
	FeatureIDs []string
}

// Result contains all usable progress evidence and any scoped diagnostics.
// Selected is the highest-ID non-paused active feature, or the highest-ID
// delivered feature when no active feature exists.
type Result struct {
	Features    []Feature
	Selected    *Feature
	IssueLinks  []IssueLink
	Diagnostics []Diagnostic
}

func (r Result) FeaturesForIssueURL(issueURL string) []Feature {
	var matches []Feature
	for _, feature := range r.Features {
		for _, candidate := range feature.IssueURLs {
			if candidate == issueURL {
				matches = append(matches, feature)
				break
			}
		}
	}
	return matches
}

func (r Result) HasErrors() bool {
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}

func isDelivered(phase string) bool {
	return strings.EqualFold(strings.TrimSpace(phase), "deliver")
}

func isRemoved(phase string) bool {
	return strings.EqualFold(strings.TrimSpace(phase), "removed")
}
