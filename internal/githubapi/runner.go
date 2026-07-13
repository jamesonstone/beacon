package githubapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
)

const (
	defaultCacheTTL      = 5 * time.Minute
	repositoryCacheTTL   = 7 * 24 * time.Hour
	rateRefreshInterval  = 30 * time.Second
	rateRefreshCallCount = 5
)

type bucket string

const (
	bucketCore    bucket = "core"
	bucketGraphQL bucket = "graphql"
	bucketSearch  bucket = "search"
)

var reserves = map[bucket]int{
	bucketCore:    1500,
	bucketGraphQL: 2500,
	bucketSearch:  15,
}

var estimatedCosts = map[bucket]int{
	bucketCore:    1,
	bucketGraphQL: 25,
	bucketSearch:  1,
}

type cacheEntry struct {
	output    []byte
	updatedAt time.Time
}

type rateState struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}

type rateResponse struct {
	Resources struct {
		Core    rateState `json:"core"`
		GraphQL rateState `json:"graphql"`
		Search  rateState `json:"search"`
	} `json:"resources"`
}

// BudgetError reports that Beacon intentionally preserved the caller's GitHub
// allowance instead of issuing another background request.
type BudgetError struct {
	Bucket    string
	Remaining int
	Reserve   int
	ResetAt   time.Time
}

func (e *BudgetError) Error() string {
	reset := "an unknown time"
	if !e.ResetAt.IsZero() {
		reset = e.ResetAt.Local().Format(time.RFC3339)
	}
	return fmt.Sprintf("Beacon paused GitHub %s requests: %d remaining with %d reserved; resets at %s", e.Bucket, e.Remaining, e.Reserve, reset)
}

// Runner caches read-only gh results and protects a reserve in each GitHub API
// rate bucket. A Runner is safe for concurrent use and should be shared by all
// GitHub collection paths in one Beacon process.
type Runner struct {
	delegate       command.Runner
	cacheTTL       time.Duration
	cacheDirectory string
	now            func() time.Time

	cacheMutex sync.Mutex
	cache      map[string]cacheEntry

	rateMutex     sync.Mutex
	rates         map[bucket]rateState
	checkedAt     time.Time
	callsSinceRun int
	blockedUntil  map[bucket]time.Time
	bucketMutexes map[bucket]*sync.Mutex
}

func NewRunner(delegate command.Runner, cacheTTL time.Duration) *Runner {
	return NewRunnerWithOptions(delegate, Options{CacheTTL: cacheTTL})
}

type Options struct {
	CacheTTL       time.Duration
	CacheDirectory string
}

func NewRunnerWithOptions(delegate command.Runner, options Options) *Runner {
	cacheTTL := options.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}
	cacheDirectory := options.CacheDirectory
	if cacheDirectory == "" {
		cacheDirectory = DefaultCacheDirectory()
	}
	return &Runner{
		delegate: delegate, cacheTTL: cacheTTL, cacheDirectory: cacheDirectory,
		now: time.Now, cache: make(map[string]cacheEntry), rates: make(map[bucket]rateState),
		blockedUntil: make(map[bucket]time.Time),
		bucketMutexes: map[bucket]*sync.Mutex{
			bucketCore: {}, bucketGraphQL: {}, bucketSearch: {},
		},
	}
}

func (r *Runner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	apiBucket, guarded := commandBucket(name, args)
	if !guarded {
		return r.delegate.Run(ctx, dir, name, args...)
	}
	key := cacheKey(dir, name, args)
	entry, found := r.cached(key, args)
	if found && r.now().Sub(entry.updatedAt) < r.ttl(args) {
		return clone(entry.output), nil
	}
	bucketMutex := r.bucketMutexes[apiBucket]
	bucketMutex.Lock()
	defer bucketMutex.Unlock()
	entry, found = r.cached(key, args)
	if found && r.now().Sub(entry.updatedAt) < r.ttl(args) {
		return clone(entry.output), nil
	}
	if err := r.reserve(ctx, apiBucket); err != nil {
		if found {
			return clone(entry.output), nil
		}
		return nil, err
	}
	output, err := r.delegate.Run(ctx, dir, name, args...)
	if err != nil {
		if isRateLimitError(err) {
			r.noteRateLimit(ctx, apiBucket)
		}
		if found && isRateLimitError(err) {
			return clone(entry.output), nil
		}
		return output, err
	}
	r.store(key, output)
	return clone(output), nil
}

func (r *Runner) reserve(ctx context.Context, apiBucket bucket) error {
	r.rateMutex.Lock()
	defer r.rateMutex.Unlock()
	now := r.now()
	if reset := r.blockedUntil[apiBucket]; now.Before(reset) {
		state := r.rates[apiBucket]
		return budgetError(apiBucket, state, reset)
	}
	if r.checkedAt.IsZero() || now.Sub(r.checkedAt) >= rateRefreshInterval || r.callsSinceRun >= rateRefreshCallCount {
		if err := r.refreshRates(ctx); err != nil {
			return err
		}
	}
	state, found := r.rates[apiBucket]
	if !found {
		return fmt.Errorf("GitHub rate response omitted %s capacity", apiBucket)
	}
	estimatedCost := estimatedCosts[apiBucket]
	if state.Remaining-estimatedCost < reserves[apiBucket] {
		return budgetError(apiBucket, state, time.Unix(state.Reset, 0))
	}
	state.Remaining -= estimatedCost
	r.rates[apiBucket] = state
	r.callsSinceRun++
	return nil
}

func (r *Runner) refreshRates(ctx context.Context) error {
	output, err := r.delegate.Run(ctx, "", "gh", "api", "rate_limit")
	if err != nil {
		return fmt.Errorf("inspect GitHub API budget: %w", err)
	}
	var response rateResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return fmt.Errorf("decode GitHub API budget: %w", err)
	}
	r.rates[bucketCore] = response.Resources.Core
	r.rates[bucketGraphQL] = response.Resources.GraphQL
	r.rates[bucketSearch] = response.Resources.Search
	r.checkedAt = r.now()
	r.callsSinceRun = 0
	for apiBucket, reset := range r.blockedUntil {
		if !r.checkedAt.Before(reset) {
			delete(r.blockedUntil, apiBucket)
		}
	}
	return nil
}

func (r *Runner) noteRateLimit(ctx context.Context, apiBucket bucket) {
	r.rateMutex.Lock()
	defer r.rateMutex.Unlock()
	r.checkedAt = time.Time{}
	if err := r.refreshRates(ctx); err == nil {
		state := r.rates[apiBucket]
		r.blockedUntil[apiBucket] = time.Unix(state.Reset, 0)
		return
	}
	r.blockedUntil[apiBucket] = r.now().Add(time.Minute)
}

func (r *Runner) ttl(args []string) time.Duration {
	if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
		return repositoryCacheTTL
	}
	return r.cacheTTL
}

func commandBucket(name string, args []string) (bucket, bool) {
	if name != "gh" || len(args) == 0 {
		return "", false
	}
	switch args[0] {
	case "repo", "pr", "issue":
		return bucketGraphQL, true
	case "search":
		return bucketSearch, true
	case "api":
		if len(args) > 1 && args[1] == "graphql" {
			return bucketGraphQL, true
		}
		return bucketCore, true
	default:
		return "", false
	}
}

func cacheKey(dir, name string, args []string) string {
	return dir + "\x00" + name + "\x00" + strings.Join(args, "\x00")
}

func budgetError(apiBucket bucket, state rateState, reset time.Time) error {
	return &BudgetError{
		Bucket: string(apiBucket), Remaining: state.Remaining,
		Reserve: reserves[apiBucket], ResetAt: reset,
	}
}

func isRateLimitError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "rate limit") || strings.Contains(message, "secondary rate")
}

func clone(value []byte) []byte {
	return append([]byte(nil), value...)
}
