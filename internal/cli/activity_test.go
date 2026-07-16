package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
)

func TestHookCommandFailsOpenWithoutStartingAgentOrLeakingPayload(t *testing.T) {
	home := t.TempDir()
	cache := filepath.Join(home, "cache")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	repository := filepath.Join(home, "repo")
	if err := os.MkdirAll(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	payload := `{"hook_event_name":"Stop","session_id":"raw-session","cwd":"` + repository + `","prompt":"secret prompt"}`
	output := &bytes.Buffer{}
	errors := &bytes.Buffer{}
	starter := &recordingAgentStarter{}
	app := App{
		In: strings.NewReader(payload), Out: output, Err: errors,
		autoStartAgent:     true,
		agentStarterSource: func(agent.Paths) agentStarter { return starter },
	}
	command := app.Root()
	command.SetArgs([]string{"activity", "ingest", "--hook", "--provider", "codex"})
	started := time.Now()
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("hook command took %s", elapsed)
	}
	if starter.calls != 0 {
		t.Fatalf("agent starts = %d", starter.calls)
	}
	if output.Len() != 0 || errors.Len() != 0 {
		t.Fatalf("hook output stdout=%q stderr=%q", output.String(), errors.String())
	}
	health, err := os.ReadFile(filepath.Join(cache, "beacon", "integration-health.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"raw-session", "secret prompt", repository} {
		if bytes.Contains(health, []byte(forbidden)) {
			t.Fatalf("health retained %q: %s", forbidden, health)
		}
	}
	if _, err := os.Stat(filepath.Join(cache, "beacon", "activity.json")); !os.IsNotExist(err) {
		t.Fatalf("unavailable agent wrote activity: %v", err)
	}
}

func TestHookCommandReturnsSuccessForMalformedAndOversizedInput(t *testing.T) {
	for _, input := range []string{"{", strings.Repeat("x", (32<<10)+1)} {
		app := App{In: strings.NewReader(input), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
		command := app.Root()
		command.SetArgs([]string{"activity", "ingest", "--hook", "--provider", "codex"})
		if err := command.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("input length %d: %v", len(input), err)
		}
	}
}
