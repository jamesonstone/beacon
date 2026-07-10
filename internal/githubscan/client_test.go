package githubscan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/jamesonstone/beacon/internal/config"
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

func TestCollectMineFiltersRepositoriesAndEnrichesEvidence(t *testing.T) {
	runner := &fixtureRunner{responses: map[string][]byte{
		"gh search prs":    []byte(`[{"number":2,"repository":{"nameWithOwner":"owner/beacon"}},{"number":9,"repository":{"nameWithOwner":"other/repo"}}]`),
		"gh pr view":       []byte(`[REPLACED]`),
		"gh api graphql":   []byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"totalCount":2,"nodes":[{"isResolved":false},{"isResolved":true}]}}}}}`),
		"gh search issues": []byte(`[{"number":1,"title":"Build Beacon","url":"https://github.com/owner/beacon/issues/1","updatedAt":"2026-07-10T12:00:00Z","labels":[{"name":"feature"}],"assignees":[{"login":"me"}],"repository":{"nameWithOwner":"owner/beacon"}}]`),
	}}
	runner.responses["gh pr view"] = []byte(`{"number":2,"title":"Feature","url":"https://github.com/owner/beacon/pull/2","headRefName":"GH-1","headRefOid":"abc","baseRefName":"main","isDraft":false,"updatedAt":"2026-07-10T12:00:00Z","reviewDecision":"","statusCheckRollup":[{"status":"COMPLETED","conclusion":"SUCCESS"}],"mergeStateStatus":"CLEAN","mergeable":"MERGEABLE","comments":[{}],"reviews":[],"closingIssuesReferences":[{"number":1,"title":"Build Beacon","url":"https://github.com/owner/beacon/issues/1","updatedAt":"2026-07-10T12:00:00Z","labels":[],"assignees":[]}]}`)

	collection := (Client{Runner: runner}).Collect(context.Background(), []config.Repository{{Name: "beacon", GitHub: "owner/beacon"}}, "mine", "@me", 2)
	evidence := collection.Repositories["owner/beacon"]
	if len(collection.Errors) != 0 || len(collection.Warnings) != 0 || len(evidence.Errors) != 0 || len(evidence.Warnings) != 0 {
		t.Fatalf("diagnostics = %#v / %#v", collection, evidence)
	}
	if len(evidence.PullRequests) != 1 || evidence.PullRequests[0].Number != 2 || evidence.PullRequests[0].Feedback.UnresolvedThreads != 1 || evidence.PullRequests[0].Checks.Success != 1 {
		t.Fatalf("pull requests = %#v", evidence.PullRequests)
	}
	if len(evidence.Issues) != 1 || evidence.Issues[0].Number != 1 {
		t.Fatalf("issues = %#v", evidence.Issues)
	}
	if runner.count("gh pr view") != 1 {
		t.Fatalf("PR detail calls = %d", runner.count("gh pr view"))
	}
}

func TestCollectAllKeepsIssueEvidenceWhenPullRequestsFail(t *testing.T) {
	runner := &fixtureRunner{
		responses: map[string][]byte{
			"gh issue list": []byte(`[{"number":7,"title":"Queued","url":"https://github.com/owner/repo/issues/7","updatedAt":"2026-07-10T12:00:00Z","labels":[],"assignees":[]}]`),
		},
		failures: map[string]error{"gh pr list": fmt.Errorf("pull requests unavailable")},
	}
	collection := (Client{Runner: runner}).Collect(context.Background(), []config.Repository{{Name: "repo", GitHub: "owner/repo"}}, "all", "@me", 1)
	evidence := collection.Repositories["owner/repo"]
	if len(evidence.Issues) != 1 || len(evidence.Errors) != 1 || evidence.Errors[0].Stage != "github-prs" {
		t.Fatalf("evidence = %#v", evidence)
	}
}

func TestCollectMineWarnsWhenIssueSearchHitsCap(t *testing.T) {
	issues := make([]rawIssue, searchLimit)
	for index := range issues {
		issues[index] = rawIssue{Number: index + 1, Repository: rawRepository{NameWithOwner: "owner/beacon"}}
	}
	issueJSON, err := json.Marshal(issues)
	if err != nil {
		t.Fatal(err)
	}
	runner := &fixtureRunner{responses: map[string][]byte{
		"gh search prs":    []byte(`[]`),
		"gh search issues": issueJSON,
	}}
	collection := (Client{Runner: runner}).Collect(context.Background(), []config.Repository{{Name: "beacon", GitHub: "owner/beacon"}}, "mine", "@me", 2)
	if len(collection.Errors) != 0 || len(collection.Warnings) != 1 || collection.Warnings[0].Stage != "github-search-issues" || !strings.Contains(collection.Warnings[0].Message, "truncated") {
		t.Fatalf("diagnostics = %#v", collection)
	}
}

type fixtureRunner struct {
	mutex     sync.Mutex
	responses map[string][]byte
	failures  map[string]error
	calls     []string
}

func (r *fixtureRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	command := strings.Join(append([]string{name}, args...), " ")
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for prefix, failure := range r.failures {
		if strings.HasPrefix(command, prefix) {
			r.calls = append(r.calls, prefix)
			return nil, failure
		}
	}
	for prefix, response := range r.responses {
		if strings.HasPrefix(command, prefix) {
			r.calls = append(r.calls, prefix)
			return append([]byte(nil), response...), nil
		}
	}
	return nil, fmt.Errorf("unexpected command: %s", command)
}

func (r *fixtureRunner) count(prefix string) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	count := 0
	for _, call := range r.calls {
		if call == prefix {
			count++
		}
	}
	return count
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

func TestNormalizePullRequestUsesLatestReviewStatePerAuthor(t *testing.T) {
	input := []byte(`[{
		"number":42,"title":"Feature","url":"https://example.test/42",
		"headRefName":"feature","headRefOid":"abc","baseRefName":"main",
		"updatedAt":"2026-07-09T12:00:00Z","statusCheckRollup":[],
		"reviews":[
			{"state":"CHANGES_REQUESTED","author":{"login":"reviewer"},"submittedAt":"2026-07-09T10:00:00Z"},
			{"state":"APPROVED","author":{"login":"reviewer"},"submittedAt":"2026-07-09T11:00:00Z"},
			{"state":"CHANGES_REQUESTED","author":{"login":"second"},"submittedAt":"2026-07-09T11:30:00Z"}
		]
	}]`)
	pullRequests, err := parsePullRequests(input)
	if err != nil {
		t.Fatal(err)
	}
	feedback := pullRequests[0].Feedback
	if feedback.Reviews != 3 || feedback.Approvals != 1 || feedback.ChangesRequested != 1 {
		t.Fatalf("feedback = %#v", feedback)
	}
}
