package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jamesonstone/beacon/internal/githubapi"
)

func TestLimitsJSONUsesOneExplicitGitHubRequest(t *testing.T) {
	runner := &limitsCommandRunner{output: []byte(`{"resources":{"core":{"limit":5000,"used":1000,"remaining":4000},"graphql":{"limit":5000,"used":2500,"remaining":2500},"search":{"limit":30,"used":3,"remaining":27}}}`)}
	var stdout bytes.Buffer
	app := App{Out: &stdout, Err: &bytes.Buffer{}, Runner: runner}
	root := app.Root()
	root.SetArgs([]string{"limits", "--json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 || runner.name != "gh" || !reflect.DeepEqual(runner.args, []string{"api", "rate_limit"}) {
		t.Fatalf("command = calls:%d name:%q args:%v", runner.calls, runner.name, runner.args)
	}
	var report githubapi.RateLimitReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if len(report.Dependencies) != 1 || len(report.Dependencies[0].Buckets) != 3 {
		t.Fatalf("report = %#v", report)
	}
}

type limitsCommandRunner struct {
	output []byte
	calls  int
	name   string
	args   []string
}

func (r *limitsCommandRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.calls++
	r.name = name
	r.args = append([]string(nil), args...)
	return append([]byte(nil), r.output...), nil
}
