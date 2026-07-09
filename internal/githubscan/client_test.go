package githubscan

import (
	"testing"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestNormalizeCI(t *testing.T) {
	tests := []struct {
		name     string
		checks   []map[string]any
		expected model.CIState
	}{
		{"none", nil, model.CINone},
		{"success", []map[string]any{{"status": "COMPLETED", "conclusion": "SUCCESS"}}, model.CISuccess},
		{"pending", []map[string]any{{"status": "IN_PROGRESS", "conclusion": ""}}, model.CIPending},
		{"failure wins", []map[string]any{{"state": "PENDING"}, {"state": "FAILURE"}}, model.CIFailure},
		{"unknown", []map[string]any{{"status": "COMPLETED"}}, model.CIUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := normalizeCI(test.checks); actual != test.expected {
				t.Fatalf("CI = %q, want %q", actual, test.expected)
			}
		})
	}
}

func TestParsePullRequests(t *testing.T) {
	input := []byte(`[{"number":42,"title":"Feature","url":"https://example.test/42","headRefName":"feature","headRefOid":"abc","baseRefName":"main","isDraft":false,"updatedAt":"2026-07-09T12:00:00Z","reviewDecision":"REVIEW_REQUIRED","statusCheckRollup":[],"mergeStateStatus":"CLEAN","mergeable":"MERGEABLE"}]`)
	pullRequests, err := parsePullRequests(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(pullRequests) != 1 || pullRequests[0].Number != 42 || pullRequests[0].CI != model.CINone {
		t.Fatalf("pull requests = %#v", pullRequests)
	}
}
