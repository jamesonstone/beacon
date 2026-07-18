package githubscan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

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
		"gh search prs": []byte(`[{"number":2,"updatedAt":"2099-07-11T22:00:00Z","repository":{"nameWithOwner":"owner/beacon"}},{"number":9,"updatedAt":"2099-07-11T22:00:00Z","repository":{"nameWithOwner":"other/repo"}}]`),
		"gh pr view":    []byte(`[REPLACED]`),
		"gh api graphql": []byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"totalCount":2,"nodes":[
			{"id":"thread-1","isResolved":false,"path":"internal/work.go","line":42,"comments":{"totalCount":1,"nodes":[{"id":"comment-1","author":{"login":"reviewer"},"body":"Handle the retry before returning.","url":"https://github.com/owner/beacon/pull/2#discussion_r1","createdAt":"2026-07-10T10:00:00Z","updatedAt":"2026-07-10T10:00:00Z"}]}},
			{"id":"thread-2","isResolved":true,"path":"README.md","comments":{"totalCount":0,"nodes":[]}}
		]}}}}}`),
		"gh search issues": []byte(`[{"number":1,"title":"Build Beacon","body":"Issue acceptance criteria","url":"https://github.com/owner/beacon/issues/1","updatedAt":"2026-07-10T12:00:00Z","labels":[{"name":"feature"}],"assignees":[{"login":"me"}],"repository":{"nameWithOwner":"owner/beacon"}}]`),
	}}
	runner.responses["gh pr view"] = []byte(`{"number":2,"title":"Feature","body":"Pull request description","url":"https://github.com/owner/beacon/pull/2","headRefName":"GH-1","headRefOid":"abc","baseRefName":"main","isDraft":false,"updatedAt":"2026-07-10T12:00:00Z","reviewDecision":"","statusCheckRollup":[{"status":"COMPLETED","conclusion":"SUCCESS"}],"mergeStateStatus":"CLEAN","mergeable":"MERGEABLE","comments":[{}],"reviews":[],"closingIssuesReferences":[{"number":1,"title":"Build Beacon","body":"Linked issue description","url":"https://github.com/owner/beacon/issues/1","updatedAt":"2026-07-10T12:00:00Z","labels":[],"assignees":[]}]}`)

	collection := (Client{Runner: runner, Now: func() time.Time { return time.Date(2099, 7, 12, 0, 0, 0, 0, time.UTC) }}).Collect(context.Background(), []config.Repository{
		{Name: "beacon", GitHub: "owner/beacon"},
		{Name: "other", GitHub: "owner/other"},
	}, "mine", "@me", 2)
	evidence := collection.Repositories["owner/beacon"]
	if len(collection.Errors) != 0 || len(collection.Warnings) != 0 || len(evidence.Errors) != 0 || len(evidence.Warnings) != 0 {
		t.Fatalf("diagnostics = %#v / %#v", collection, evidence)
	}
	if len(evidence.PullRequests) != 1 || evidence.PullRequests[0].Number != 2 || evidence.PullRequests[0].Body != "Pull request description" || evidence.PullRequests[0].Feedback.UnresolvedThreads != 1 || evidence.PullRequests[0].Checks.Success != 1 {
		t.Fatalf("pull requests = %#v", evidence.PullRequests)
	}
	threads := evidence.PullRequests[0].Feedback.Threads
	if len(threads) != 1 || threads[0].Path != "internal/work.go" || len(threads[0].Comments) != 1 || threads[0].Comments[0].Author != "reviewer" {
		t.Fatalf("review threads = %#v", threads)
	}
	if len(evidence.Issues) != 1 || evidence.Issues[0].Number != 1 || evidence.Issues[0].Body != "Issue acceptance criteria" {
		t.Fatalf("issues = %#v", evidence.Issues)
	}
	if runner.count("gh pr view") != 1 || runner.count("gh api graphql") != 1 {
		t.Fatalf("PR detail calls = view:%d threads:%d", runner.count("gh pr view"), runner.count("gh api graphql"))
	}
}

func TestCollectMineBatchesEightyRepositoriesIntoTwoSearches(t *testing.T) {
	runner := &fixtureRunner{responses: map[string][]byte{
		"gh search prs":    []byte(`[]`),
		"gh search issues": []byte(`[]`),
	}}
	repositories := make([]config.Repository, 80)
	for index := range repositories {
		repositories[index] = config.Repository{
			Name:   fmt.Sprintf("repo-%02d", index),
			GitHub: fmt.Sprintf("owner/repo-%02d", index),
		}
	}

	collection := (Client{Runner: runner}).Collect(context.Background(), repositories, "mine", "@me", 4)
	if len(collection.Errors) != 0 || len(collection.Repositories) != len(repositories) {
		t.Fatalf("collection = %#v", collection)
	}
	if runner.count("gh search prs") != 1 || runner.count("gh search issues") != 1 {
		t.Fatalf("calls = %v", runner.calls)
	}
	for _, call := range runner.calls {
		if strings.HasPrefix(call, "gh pr list") || strings.HasPrefix(call, "gh issue list") {
			t.Fatalf("batch collection used repository-scoped list: %s", call)
		}
	}
}

func TestCollectMineForOneRepositoryStillUsesGlobalSearches(t *testing.T) {
	runner := &fixtureRunner{responses: map[string][]byte{
		"gh search prs":    []byte(`[]`),
		"gh search issues": []byte(`[]`),
	}}
	collection := (Client{Runner: runner}).Collect(context.Background(), []config.Repository{{Name: "beacon", GitHub: "owner/beacon"}}, "mine", "@me", 2)
	if len(collection.Errors) != 0 || len(collection.Repositories["owner/beacon"].Errors) != 0 {
		t.Fatalf("collection = %#v", collection)
	}
	joined := strings.Join(runner.calls, "\n")
	if !strings.Contains(joined, "gh search prs --author @me") || !strings.Contains(joined, "gh search issues --assignee @me") {
		t.Fatalf("global calls = %v", runner.calls)
	}
}

func TestCollectMineSkipsInactivePullRequestEnrichmentUnlessExplicit(t *testing.T) {
	now := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	responses := map[string][]byte{
		"gh search prs":    []byte(`[{"number":2,"updatedAt":"2026-06-01T12:00:00Z","repository":{"nameWithOwner":"owner/beacon"}}]`),
		"gh search issues": []byte(`[]`),
		"gh pr view":       []byte(`{"number":2,"title":"Old","url":"https://github.com/owner/beacon/pull/2","headRefName":"old","headRefOid":"abc","baseRefName":"main","isDraft":false,"updatedAt":"2026-06-01T12:00:00Z","reviewDecision":"","statusCheckRollup":[],"mergeStateStatus":"CLEAN","mergeable":"MERGEABLE","comments":[],"reviews":[],"closingIssuesReferences":[]}`),
		"gh api graphql":   []byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"totalCount":0,"nodes":[]}}}}}`),
	}
	runner := &fixtureRunner{responses: responses}
	client := Client{Runner: runner, Now: func() time.Time { return now }}
	repositories := []config.Repository{{Name: "beacon", GitHub: "owner/beacon"}}
	collection := client.Collect(context.Background(), repositories, "mine", "@me", 2)
	if len(collection.Repositories["owner/beacon"].PullRequests) != 0 || runner.count("gh pr view") != 0 {
		t.Fatalf("inactive PR was enriched: %#v / %v", collection, runner.calls)
	}

	runner.calls = nil
	collection = client.Collect(WithInactivePullRequests(context.Background()), repositories, "mine", "@me", 2)
	if len(collection.Repositories["owner/beacon"].PullRequests) != 1 || runner.count("gh pr view") != 1 {
		t.Fatalf("explicit refresh omitted inactive PR: %#v / %v", collection, runner.calls)
	}
}

func TestCollectMineOnlyEnrichesInactivePullRequestsForFollowedRepositories(t *testing.T) {
	now := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	runner := &fixtureRunner{responses: map[string][]byte{
		"gh search prs": []byte(`[
			{"number":2,"updatedAt":"2026-06-01T12:00:00Z","repository":{"nameWithOwner":"owner/followed"}},
			{"number":3,"updatedAt":"2026-06-01T12:00:00Z","repository":{"nameWithOwner":"owner/quiet"}}
		]`),
		"gh search issues": []byte(`[]`),
		"gh pr view":       []byte(`{"number":2,"title":"Old","url":"https://github.com/owner/followed/pull/2","headRefName":"old","headRefOid":"abc","baseRefName":"main","isDraft":false,"updatedAt":"2026-06-01T12:00:00Z","reviewDecision":"","statusCheckRollup":[],"mergeStateStatus":"CLEAN","mergeable":"MERGEABLE","comments":[],"reviews":[],"closingIssuesReferences":[]}`),
		"gh api graphql":   []byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"totalCount":0,"nodes":[]}}}}}`),
	}}
	client := Client{Runner: runner, Now: func() time.Time { return now }}
	repositories := []config.Repository{
		{Name: "followed", GitHub: "owner/followed"},
		{Name: "quiet", GitHub: "owner/quiet"},
	}
	ctx := WithInactivePullRequestRepositories(context.Background(), []string{"owner/followed"})
	collection := client.Collect(ctx, repositories, "mine", "@me", 2)
	if len(collection.Repositories["owner/followed"].PullRequests) != 1 || len(collection.Repositories["owner/quiet"].PullRequests) != 0 {
		t.Fatalf("collection = %#v", collection.Repositories)
	}
	if runner.count("gh pr view") != 1 || runner.count("gh api graphql") != 1 {
		t.Fatalf("inactive enrichment calls = %v", runner.calls)
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
	collection := (Client{Runner: runner}).Collect(context.Background(), []config.Repository{
		{Name: "beacon", GitHub: "owner/beacon"},
		{Name: "other", GitHub: "owner/other"},
	}, "mine", "@me", 2)
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
			r.calls = append(r.calls, command)
			return nil, failure
		}
	}
	for prefix, response := range r.responses {
		if strings.HasPrefix(command, prefix) {
			r.calls = append(r.calls, command)
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
		if strings.HasPrefix(call, prefix) {
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

func TestReviewThreadDetailsSortsAndMarksTruncation(t *testing.T) {
	runner := &fixtureRunner{responses: map[string][]byte{
		"gh api graphql": []byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"totalCount":3,"nodes":[
			{"id":"z","isResolved":false,"isOutdated":true,"path":"z.go","originalLine":12,"comments":{"totalCount":1,"nodes":[{"id":"later","author":{"login":"zoe"},"body":"later","url":"https://example.test/later","createdAt":"2026-07-10T12:00:00Z","updatedAt":"2026-07-10T12:00:00Z"}]}},
			{"id":"a","isResolved":false,"path":"a.go","line":7,"comments":{"totalCount":3,"nodes":[{"id":"second","author":{"login":"sam"},"body":"second","url":"https://example.test/second","createdAt":"2026-07-10T11:00:00Z","updatedAt":"2026-07-10T11:00:00Z"},{"id":"first","author":{"login":"amy"},"body":"first","url":"https://example.test/first","createdAt":"2026-07-10T10:00:00Z","updatedAt":"2026-07-10T10:00:00Z"}]}}
		]}}}}}`),
	}}
	threads, count, truncated, err := (Client{Runner: runner}).reviewThreadDetails(context.Background(), "owner/beacon", 39)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || !truncated || len(threads) != 2 || threads[0].ID != "a" || threads[1].ID != "z" {
		t.Fatalf("threads=%#v count=%d truncated=%v", threads, count, truncated)
	}
	if !threads[0].CommentsTruncated || threads[0].Comments[0].ID != "first" || !threads[1].Outdated {
		t.Fatalf("thread details=%#v", threads)
	}
}

func TestTruncateGitHubBodyPreservesUTF8(t *testing.T) {
	body := strings.Repeat("x", maxGitHubBodyBytes-1) + "🦞more"
	truncated, didTruncate := truncateGitHubBody(body)
	if !didTruncate || !strings.HasSuffix(truncated, "x") || !utf8.ValidString(truncated) {
		t.Fatalf("truncated bytes=%d valid=%v did_truncate=%v", len(truncated), utf8.ValidString(truncated), didTruncate)
	}
}
