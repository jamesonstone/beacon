package githubapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
)

const rateLimitInspectionTimeout = 20 * time.Second

// RateLimitReport is an explicit snapshot of allowances for Beacon's external
// dependencies. Beacon does not collect this report in the background.
type RateLimitReport struct {
	CheckedAt    time.Time             `json:"checked_at"`
	Dependencies []RateLimitDependency `json:"dependencies"`
}

type RateLimitDependency struct {
	Name    string            `json:"name"`
	Buckets []RateLimitBucket `json:"buckets"`
}

type RateLimitBucket struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Limit     int       `json:"limit"`
	Used      int       `json:"used"`
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"reset_at"`
}

// InspectRateLimits runs exactly one bounded gh request and returns the three
// GitHub buckets Beacon consumes. Callers must invoke it only from an explicit
// user action.
func InspectRateLimits(ctx context.Context, runner command.Runner, now func() time.Time) (RateLimitReport, error) {
	if runner == nil {
		return RateLimitReport{}, fmt.Errorf("inspect dependency limits: command runner is unavailable")
	}
	if now == nil {
		now = time.Now
	}
	requestContext, cancel := context.WithTimeout(ctx, rateLimitInspectionTimeout)
	defer cancel()

	output, err := runner.Run(requestContext, "", "gh", "api", "rate_limit")
	if err != nil {
		return RateLimitReport{}, fmt.Errorf("inspect GitHub API limits: %w", err)
	}
	var response rateResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return RateLimitReport{}, fmt.Errorf("decode GitHub API limits: %w", err)
	}

	return RateLimitReport{
		CheckedAt: now().UTC(),
		Dependencies: []RateLimitDependency{{
			Name: "gh",
			Buckets: []RateLimitBucket{
				newRateLimitBucket("graphql", "GraphQL", response.Resources.GraphQL),
				newRateLimitBucket("core", "REST Core", response.Resources.Core),
				newRateLimitBucket("search", "Search", response.Resources.Search),
			},
		}},
	}, nil
}

func newRateLimitBucket(id, name string, state rateState) RateLimitBucket {
	used := state.Used
	if derived := state.Limit - state.Remaining; derived > used {
		used = derived
	}
	if used < 0 {
		used = 0
	}
	if state.Limit > 0 && used > state.Limit {
		used = state.Limit
	}
	resetAt := time.Time{}
	if state.Reset > 0 {
		resetAt = time.Unix(state.Reset, 0).UTC()
	}
	return RateLimitBucket{
		ID: id, Name: name, Limit: state.Limit, Used: used,
		Remaining: state.Remaining, ResetAt: resetAt,
	}
}
