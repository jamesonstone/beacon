package githubapi

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestInspectRateLimitsRunsOneCommandAndReturnsStableBuckets(t *testing.T) {
	runner := &rateLimitRecordingRunner{output: []byte(`{
  "resources": {
    "core": {"limit": 5000, "used": 1250, "remaining": 3750, "reset": 1720000100},
    "graphql": {"limit": 5000, "used": 2500, "remaining": 2500, "reset": 1720000200},
    "search": {"limit": 30, "used": 3, "remaining": 27, "reset": 1720000300}
  }
}`)}
	checkedAt := time.Date(2026, 7, 14, 12, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	report, err := InspectRateLimits(context.Background(), runner, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 || runner.directory != "" || runner.name != "gh" || !reflect.DeepEqual(runner.args, []string{"api", "rate_limit"}) {
		t.Fatalf("command = calls:%d dir:%q name:%q args:%v", runner.calls, runner.directory, runner.name, runner.args)
	}
	if !report.CheckedAt.Equal(checkedAt) || report.CheckedAt.Location() != time.UTC {
		t.Fatalf("checked_at = %s", report.CheckedAt)
	}
	if len(report.Dependencies) != 1 || report.Dependencies[0].Name != "gh" {
		t.Fatalf("dependencies = %#v", report.Dependencies)
	}
	buckets := report.Dependencies[0].Buckets
	if got, want := []string{buckets[0].Name, buckets[1].Name, buckets[2].Name}, []string{"GraphQL", "REST Core", "Search"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("bucket order = %v, want %v", got, want)
	}
	if buckets[0].Used != 2500 || buckets[1].Remaining != 3750 || buckets[2].Limit != 30 {
		t.Fatalf("buckets = %#v", buckets)
	}
}

func TestInspectRateLimitsDerivesUsedForOlderResponses(t *testing.T) {
	runner := &rateLimitRecordingRunner{output: []byte(`{"resources":{"core":{"limit":5000,"remaining":4999},"graphql":{"limit":5000,"remaining":5000},"search":{"limit":30,"remaining":30}}}`)}
	report, err := InspectRateLimits(context.Background(), runner, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Dependencies[0].Buckets[1].Used; got != 1 {
		t.Fatalf("derived REST Core used = %d, want 1", got)
	}
}

type rateLimitRecordingRunner struct {
	output    []byte
	err       error
	calls     int
	directory string
	name      string
	args      []string
}

func (r *rateLimitRecordingRunner) Run(_ context.Context, directory, name string, args ...string) ([]byte, error) {
	r.calls++
	r.directory = directory
	r.name = name
	r.args = append([]string(nil), args...)
	if r.err != nil {
		return nil, fmt.Errorf("command failed: %w", r.err)
	}
	return append([]byte(nil), r.output...), nil
}
