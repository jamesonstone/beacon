package githubscan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/model"
)

const githubTimeout = 20 * time.Second

type Client struct {
	Runner command.Runner
}

type rawPullRequest struct {
	Number            int              `json:"number"`
	Title             string           `json:"title"`
	URL               string           `json:"url"`
	HeadRefName       string           `json:"headRefName"`
	HeadRefOID        string           `json:"headRefOid"`
	BaseRefName       string           `json:"baseRefName"`
	IsDraft           bool             `json:"isDraft"`
	UpdatedAt         time.Time        `json:"updatedAt"`
	ReviewDecision    string           `json:"reviewDecision"`
	StatusCheckRollup []map[string]any `json:"statusCheckRollup"`
	MergeStateStatus  string           `json:"mergeStateStatus"`
	Mergeable         string           `json:"mergeable"`
}

func (c Client) ListOpen(ctx context.Context, repository, author string) ([]model.PullRequest, error) {
	commandContext, cancel := context.WithTimeout(ctx, githubTimeout)
	defer cancel()
	fields := "number,title,url,headRefName,headRefOid,baseRefName,isDraft,updatedAt,reviewDecision,statusCheckRollup,mergeStateStatus,mergeable"
	output, err := c.Runner.Run(commandContext, "", "gh", "pr", "list", "--repo", repository, "--author", author, "--state", "open", "--limit", "100", "--json", fields)
	if err != nil {
		return nil, err
	}
	return parsePullRequests(output)
}

func parsePullRequests(output []byte) ([]model.PullRequest, error) {
	var raw []rawPullRequest
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("decode gh pull requests: %w", err)
	}
	result := make([]model.PullRequest, 0, len(raw))
	for _, pullRequest := range raw {
		result = append(result, model.PullRequest{
			Number: pullRequest.Number, Title: pullRequest.Title, URL: pullRequest.URL,
			HeadRefName: pullRequest.HeadRefName, HeadRefOID: pullRequest.HeadRefOID,
			BaseRefName: pullRequest.BaseRefName, IsDraft: pullRequest.IsDraft,
			UpdatedAt: pullRequest.UpdatedAt, ReviewDecision: pullRequest.ReviewDecision,
			MergeState: pullRequest.MergeStateStatus, Mergeable: pullRequest.Mergeable,
			CI: normalizeCI(pullRequest.StatusCheckRollup),
		})
	}
	return result, nil
}

func normalizeCI(checks []map[string]any) model.CIState {
	if len(checks) == 0 {
		return model.CINone
	}
	known := false
	pending := false
	unknown := false
	for _, check := range checks {
		state := upperString(check["state"])
		status := upperString(check["status"])
		conclusion := upperString(check["conclusion"])
		if isFailure(state) || isFailure(conclusion) {
			return model.CIFailure
		}
		if state == "PENDING" || state == "EXPECTED" || status == "QUEUED" || status == "IN_PROGRESS" || status == "PENDING" || status == "WAITING" || status == "REQUESTED" {
			pending = true
			known = true
			continue
		}
		if state == "SUCCESS" || conclusion == "SUCCESS" || conclusion == "NEUTRAL" || conclusion == "SKIPPED" {
			known = true
			continue
		}
		if status == "COMPLETED" && conclusion == "" {
			unknown = true
			continue
		}
		unknown = true
	}
	if pending {
		return model.CIPending
	}
	if unknown || !known {
		return model.CIUnknown
	}
	return model.CISuccess
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
