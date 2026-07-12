package githubapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunnerCachesIdenticalGitHubCommands(t *testing.T) {
	delegate := &fakeRunner{rate: rateJSON(5000, 30, 5000), output: []byte(`[{"number":1}]`)}
	runner := NewRunner(delegate, 5*time.Minute)

	first, err := runner.Run(context.Background(), "", "gh", "pr", "list", "--repo", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	first[0] = 'x'
	second, err := runner.Run(context.Background(), "", "gh", "pr", "list", "--repo", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != `[{"number":1}]` || delegate.commandCalls != 1 || delegate.rateCalls != 1 {
		t.Fatalf("second=%q commandCalls=%d rateCalls=%d", second, delegate.commandCalls, delegate.rateCalls)
	}
}

func TestRunnerProtectsGitHubBucketReserve(t *testing.T) {
	delegate := &fakeRunner{rate: rateJSON(1000, 30, 5000), output: []byte("[]")}
	runner := NewRunner(delegate, time.Minute)

	_, err := runner.Run(context.Background(), "", "gh", "issue", "list", "--repo", "owner/repo")
	var budget *BudgetError
	if !errors.As(err, &budget) {
		t.Fatalf("error = %v", err)
	}
	if budget.Bucket != "graphql" || budget.Reserve != 1000 || delegate.commandCalls != 0 || !strings.Contains(err.Error(), "resets at") {
		t.Fatalf("budget=%#v commandCalls=%d", budget, delegate.commandCalls)
	}
}

func TestRunnerServesStaleSuccessWhenBudgetIsProtected(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	delegate := &fakeRunner{rate: rateJSON(5000, 30, 5000), output: []byte("cached")}
	runner := NewRunner(delegate, time.Minute)
	runner.now = func() time.Time { return now }
	args := []string{"pr", "list", "--repo", "owner/repo"}
	if _, err := runner.Run(context.Background(), "", "gh", args...); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	delegate.rate = rateJSON(1000, 30, 5000)
	output, err := runner.Run(context.Background(), "", "gh", args...)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "cached" || delegate.commandCalls != 1 || delegate.rateCalls != 2 {
		t.Fatalf("output=%q commandCalls=%d rateCalls=%d", output, delegate.commandCalls, delegate.rateCalls)
	}
}

func TestRunnerPassesNonGitHubCommandsThrough(t *testing.T) {
	delegate := &fakeRunner{output: []byte("main")}
	runner := NewRunner(delegate, time.Minute)
	output, err := runner.Run(context.Background(), "/repo", "git", "branch", "--show-current")
	if err != nil || string(output) != "main" || delegate.commandCalls != 1 || delegate.rateCalls != 0 {
		t.Fatalf("output=%q err=%v commandCalls=%d rateCalls=%d", output, err, delegate.commandCalls, delegate.rateCalls)
	}
}

type fakeRunner struct {
	mutex        sync.Mutex
	rate         []byte
	output       []byte
	err          error
	rateCalls    int
	commandCalls int
}

func (r *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if name == "gh" && len(args) == 2 && args[0] == "api" && args[1] == "rate_limit" {
		r.rateCalls++
		return append([]byte(nil), r.rate...), nil
	}
	r.commandCalls++
	return append([]byte(nil), r.output...), r.err
}

func rateJSON(graphql, search, core int) []byte {
	return []byte(fmt.Sprintf(`{"resources":{"graphql":{"limit":5000,"remaining":%d,"reset":2000000000},"search":{"limit":30,"remaining":%d,"reset":2000000000},"core":{"limit":5000,"remaining":%d,"reset":2000000000}}}`, graphql, search, core))
}
