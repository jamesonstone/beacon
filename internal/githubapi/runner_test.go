package githubapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerCachesIdenticalGitHubCommands(t *testing.T) {
	delegate := &fakeRunner{rate: rateJSON(5000, 30, 5000), output: []byte(`[{"number":1}]`)}
	runner := newTestRunner(t, delegate, 5*time.Minute)

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

func TestExplicitRefreshBypassesCacheButKeepsBudgetGuard(t *testing.T) {
	delegate := &fakeRunner{rate: rateJSON(5000, 30, 5000), output: []byte(`[{"number":1}]`)}
	runner := newTestRunner(t, delegate, time.Hour)
	args := []string{"pr", "list", "--repo", "owner/repo"}
	if _, err := runner.Run(context.Background(), "", "gh", args...); err != nil {
		t.Fatal(err)
	}
	delegate.output = []byte(`[{"number":2}]`)
	output, err := runner.Run(WithFreshEvidence(context.Background()), "", "gh", args...)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != `[{"number":2}]` || delegate.commandCalls != 2 || delegate.rateCalls != 1 {
		t.Fatalf("output=%q commandCalls=%d rateCalls=%d", output, delegate.commandCalls, delegate.rateCalls)
	}
}

func TestRunnerProtectsGitHubBucketReserve(t *testing.T) {
	delegate := &fakeRunner{rate: rateJSON(2520, 30, 5000), output: []byte("[]")}
	runner := newTestRunner(t, delegate, time.Minute)

	_, err := runner.Run(context.Background(), "", "gh", "issue", "list", "--repo", "owner/repo")
	var budget *BudgetError
	if !errors.As(err, &budget) {
		t.Fatalf("error = %v", err)
	}
	if budget.Bucket != "graphql" || budget.Reserve != 2500 || delegate.commandCalls != 0 || !strings.Contains(err.Error(), "resets at") {
		t.Fatalf("budget=%#v commandCalls=%d", budget, delegate.commandCalls)
	}
}

func TestRunnerServesStaleSuccessWhenBudgetIsProtected(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	cacheDirectory := t.TempDir()
	delegate := &fakeRunner{rate: rateJSON(5000, 30, 5000), output: []byte("cached")}
	runner := NewRunnerWithOptions(delegate, Options{CacheTTL: time.Minute, CacheDirectory: cacheDirectory})
	runner.now = func() time.Time { return now }
	args := []string{"pr", "list", "--repo", "owner/repo"}
	if _, err := runner.Run(context.Background(), "", "gh", args...); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(cacheDirectory, "*.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("cache files=%v err=%v", files, err)
	}
	if info, err := os.Stat(files[0]); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("cache mode info=%v err=%v", info, err)
	}
	now = now.Add(2 * time.Minute)
	delegate.rate = rateJSON(2500, 30, 5000)
	restarted := NewRunnerWithOptions(delegate, Options{CacheTTL: time.Minute, CacheDirectory: cacheDirectory})
	restarted.now = func() time.Time { return now }
	output, err := restarted.Run(context.Background(), "", "gh", args...)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "cached" || delegate.commandCalls != 1 || delegate.rateCalls != 2 {
		t.Fatalf("output=%q commandCalls=%d rateCalls=%d", output, delegate.commandCalls, delegate.rateCalls)
	}
}

func TestRunnerPassesNonGitHubCommandsThrough(t *testing.T) {
	delegate := &fakeRunner{output: []byte("main")}
	runner := newTestRunner(t, delegate, time.Minute)
	output, err := runner.Run(context.Background(), "/repo", "git", "branch", "--show-current")
	if err != nil || string(output) != "main" || delegate.commandCalls != 1 || delegate.rateCalls != 0 {
		t.Fatalf("output=%q err=%v commandCalls=%d rateCalls=%d", output, err, delegate.commandCalls, delegate.rateCalls)
	}
}

func TestRunnerSerializesConcurrentGraphQLCacheMisses(t *testing.T) {
	delegate := &concurrencyRunner{rate: rateJSON(5000, 30, 5000)}
	runner := NewRunnerWithOptions(delegate, Options{CacheTTL: time.Minute, CacheDirectory: t.TempDir()})
	var waitGroup sync.WaitGroup
	for index := 0; index < 12; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			if _, err := runner.Run(context.Background(), "", "gh", "pr", "list", "--repo", fmt.Sprintf("owner/repo-%d", index)); err != nil {
				t.Errorf("run %d: %v", index, err)
			}
		}(index)
	}
	waitGroup.Wait()
	if delegate.maximum.Load() != 1 || delegate.commandCalls.Load() != 12 {
		t.Fatalf("maximum=%d calls=%d", delegate.maximum.Load(), delegate.commandCalls.Load())
	}
}

func newTestRunner(t *testing.T, delegate *fakeRunner, ttl time.Duration) *Runner {
	t.Helper()
	return NewRunnerWithOptions(delegate, Options{CacheTTL: ttl, CacheDirectory: t.TempDir()})
}

type fakeRunner struct {
	mutex        sync.Mutex
	rate         []byte
	output       []byte
	err          error
	rateCalls    int
	commandCalls int
}

type concurrencyRunner struct {
	rate         []byte
	active       atomic.Int32
	maximum      atomic.Int32
	commandCalls atomic.Int32
}

func (r *concurrencyRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	if name == "gh" && len(args) == 2 && args[0] == "api" && args[1] == "rate_limit" {
		return append([]byte(nil), r.rate...), nil
	}
	current := r.active.Add(1)
	for {
		maximum := r.maximum.Load()
		if current <= maximum || r.maximum.CompareAndSwap(maximum, current) {
			break
		}
	}
	r.commandCalls.Add(1)
	time.Sleep(5 * time.Millisecond)
	r.active.Add(-1)
	return []byte("[]"), nil
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
