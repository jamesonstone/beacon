package githubscan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

const pullRequestFields = "number,title,url,headRefName,headRefOid,baseRefName,isDraft,updatedAt,reviewDecision,statusCheckRollup,mergeStateStatus,mergeable,comments,reviews,closingIssuesReferences"

type rawActor struct {
	Login string `json:"login"`
}

type rawLabel struct {
	Name string `json:"name"`
}

type rawRepository struct {
	NameWithOwner string `json:"nameWithOwner"`
}

type rawIssue struct {
	Number     int           `json:"number"`
	Title      string        `json:"title"`
	URL        string        `json:"url"`
	UpdatedAt  time.Time     `json:"updatedAt"`
	Labels     []rawLabel    `json:"labels"`
	Assignees  []rawActor    `json:"assignees"`
	Repository rawRepository `json:"repository"`
}

type rawReview struct {
	State       string    `json:"state"`
	Author      rawActor  `json:"author"`
	SubmittedAt time.Time `json:"submittedAt"`
}

type rawPullRequest struct {
	Number                  int               `json:"number"`
	Title                   string            `json:"title"`
	URL                     string            `json:"url"`
	HeadRefName             string            `json:"headRefName"`
	HeadRefOID              string            `json:"headRefOid"`
	BaseRefName             string            `json:"baseRefName"`
	IsDraft                 bool              `json:"isDraft"`
	UpdatedAt               time.Time         `json:"updatedAt"`
	ReviewDecision          string            `json:"reviewDecision"`
	StatusCheckRollup       []map[string]any  `json:"statusCheckRollup"`
	MergeStateStatus        string            `json:"mergeStateStatus"`
	Mergeable               string            `json:"mergeable"`
	Comments                []json.RawMessage `json:"comments"`
	Reviews                 []rawReview       `json:"reviews"`
	ClosingIssuesReferences []rawIssue        `json:"closingIssuesReferences"`
}

type rawSearchItem struct {
	Number     int           `json:"number"`
	Repository rawRepository `json:"repository"`
}

type rawThreadResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					TotalCount int `json:"totalCount"`
					Nodes      []struct {
						IsResolved bool `json:"isResolved"`
					} `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
}

func parsePullRequests(output []byte) ([]model.PullRequest, error) {
	var raw []rawPullRequest
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("decode gh pull requests: %w", err)
	}
	result := make([]model.PullRequest, 0, len(raw))
	for _, pullRequest := range raw {
		result = append(result, normalizePullRequest(pullRequest))
	}
	sortPullRequests(result)
	return result, nil
}

func normalizePullRequest(pullRequest rawPullRequest) model.PullRequest {
	ci, checks := normalizeChecks(pullRequest.StatusCheckRollup)
	feedback := model.Feedback{Comments: len(pullRequest.Comments), Reviews: len(pullRequest.Reviews)}
	latestReviews := make(map[string]rawReview, len(pullRequest.Reviews))
	for index, review := range pullRequest.Reviews {
		key := review.Author.Login
		if key == "" {
			key = fmt.Sprintf("anonymous-%d", index)
		}
		previous, exists := latestReviews[key]
		if !exists || previous.SubmittedAt.IsZero() || review.SubmittedAt.After(previous.SubmittedAt) {
			latestReviews[key] = review
		}
	}
	for _, review := range latestReviews {
		switch strings.ToUpper(review.State) {
		case "APPROVED":
			feedback.Approvals++
		case "CHANGES_REQUESTED":
			feedback.ChangesRequested++
		}
	}
	issues := make([]model.Issue, 0, len(pullRequest.ClosingIssuesReferences))
	for _, issue := range pullRequest.ClosingIssuesReferences {
		issues = append(issues, normalizeIssue(issue))
	}
	return model.PullRequest{
		Number: pullRequest.Number, Title: pullRequest.Title, URL: pullRequest.URL,
		HeadRefName: pullRequest.HeadRefName, HeadRefOID: pullRequest.HeadRefOID,
		BaseRefName: pullRequest.BaseRefName, IsDraft: pullRequest.IsDraft,
		UpdatedAt: pullRequest.UpdatedAt, ReviewDecision: pullRequest.ReviewDecision,
		MergeState: pullRequest.MergeStateStatus, Mergeable: pullRequest.Mergeable,
		CI: ci, Checks: checks, Feedback: feedback, ClosingIssues: issues,
	}
}

func parseIssues(output []byte) ([]model.Issue, error) {
	var raw []rawIssue
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("decode gh issues: %w", err)
	}
	result := make([]model.Issue, 0, len(raw))
	for _, issue := range raw {
		result = append(result, normalizeIssue(issue))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Number < result[j].Number })
	return result, nil
}

func parseSearchIssues(output []byte) (map[string][]model.Issue, int, error) {
	var raw []rawIssue
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, 0, fmt.Errorf("decode gh issue search: %w", err)
	}
	result := make(map[string][]model.Issue)
	for _, issue := range raw {
		result[issue.Repository.NameWithOwner] = append(result[issue.Repository.NameWithOwner], normalizeIssue(issue))
	}
	for repository, issues := range result {
		sort.Slice(issues, func(i, j int) bool { return issues[i].Number < issues[j].Number })
		result[repository] = issues
	}
	return result, len(raw), nil
}

func normalizeIssue(issue rawIssue) model.Issue {
	labels := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labels = append(labels, label.Name)
	}
	sort.Strings(labels)
	assignees := make([]string, 0, len(issue.Assignees))
	for _, assignee := range issue.Assignees {
		assignees = append(assignees, assignee.Login)
	}
	sort.Strings(assignees)
	return model.Issue{Number: issue.Number, Title: issue.Title, URL: issue.URL, Labels: labels, Assignees: assignees, UpdatedAt: issue.UpdatedAt}
}

func normalizeCI(checks []map[string]any) model.CIState {
	state, _ := normalizeChecks(checks)
	return state
}

func normalizeChecks(checks []map[string]any) (model.CIState, model.CheckSummary) {
	summary := model.CheckSummary{Total: len(checks)}
	if len(checks) == 0 {
		return model.CINone, summary
	}
	for _, check := range checks {
		state := upperString(check["state"])
		status := upperString(check["status"])
		conclusion := upperString(check["conclusion"])
		switch {
		case isFailure(state) || isFailure(conclusion):
			summary.Failure++
		case state == "PENDING" || state == "EXPECTED" || status == "QUEUED" || status == "IN_PROGRESS" || status == "PENDING" || status == "WAITING" || status == "REQUESTED":
			summary.Pending++
		case conclusion == "SKIPPED" || conclusion == "NEUTRAL":
			summary.Skipped++
		case state == "SUCCESS" || conclusion == "SUCCESS":
			summary.Success++
		default:
			summary.Unknown++
		}
	}
	switch {
	case summary.Failure > 0:
		return model.CIFailure, summary
	case summary.Pending > 0:
		return model.CIPending, summary
	case summary.Unknown > 0:
		return model.CIUnknown, summary
	default:
		return model.CISuccess, summary
	}
}

func isFailure(value string) bool {
	switch value {
	case "FAILURE", "ERROR", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED", "STARTUP_FAILURE", "STALE":
		return true
	default:
		return false
	}
}

func upperString(value any) string {
	text, _ := value.(string)
	return strings.ToUpper(text)
}
