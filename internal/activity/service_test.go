package activity

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestServiceObservesBeforeUnavailableAgentAndRefreshesOnlyMatchedStop(t *testing.T) {
	root := t.TempDir()
	repository := mkdir(t, filepath.Join(root, "repo"))
	store := Store{Path: filepath.Join(root, "activity.json"), LockPath: filepath.Join(root, "activity.lock")}
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	observed := 0
	unavailable := &fakeAgentClient{err: agent.ErrUnavailable}
	service := Service{Store: store, Agent: unavailable, Now: func() time.Time { return now }, Observe: func(string) error { observed++; return nil }}
	payload := `{"hook_event_name":"Stop","session_id":"raw-session","cwd":"` + repository + `","prompt":"secret prompt","transcript_path":"/secret/transcript"}`
	if err := service.Ingest(context.Background(), ProviderCodex, strings.NewReader(payload)); err != nil || observed != 1 {
		t.Fatalf("unavailable ingest observed=%d err=%v", observed, err)
	}
	if _, err := os.Stat(store.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unavailable agent wrote cache: %v", err)
	}
	client := &fakeAgentClient{snapshot: matchingSnapshot(repository, filepath.Join(repository, "missing-worktree"))}
	service.Agent = client
	if err := service.Ingest(context.Background(), ProviderCodex, strings.NewReader(payload)); err != nil {
		t.Fatal(err)
	}
	if got := client.requestTypes(); len(got) != 2 || got[0] != agent.RequestGetSnapshot || got[1] != agent.RequestRefreshProject {
		t.Fatalf("requests = %v", got)
	}
	client.requests = nil
	if err := service.Ingest(context.Background(), ProviderCodex, strings.NewReader(payload)); err != nil {
		t.Fatal(err)
	}
	if got := client.requestTypes(); len(got) != 1 || got[0] != agent.RequestGetSnapshot {
		t.Fatalf("duplicate Stop requests = %v", got)
	}
	contents, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"raw-session", "secret prompt", "transcript"} {
		if bytes.Contains(contents, []byte(forbidden)) {
			t.Fatalf("activity cache retained %q: %s", forbidden, contents)
		}
	}
	client.requests = nil
	working := strings.Replace(payload, `"Stop"`, `"UserPromptSubmit"`, 1)
	if err := service.Ingest(context.Background(), ProviderCodex, strings.NewReader(working)); err != nil {
		t.Fatal(err)
	}
	if got := client.requestTypes(); len(got) != 1 || got[0] != agent.RequestGetSnapshot {
		t.Fatalf("working requests = %v", got)
	}
	for _, test := range []struct {
		name     string
		provider string
		payload  string
	}{
		{name: "attention", provider: ProviderClaudeCode, payload: `{"hook_event_name":"PermissionRequest","session_id":"claude-session","cwd":"` + repository + `"}`},
		{name: "failure clear", provider: ProviderClaudeCode, payload: `{"hook_event_name":"StopFailure","session_id":"claude-session","cwd":"` + repository + `"}`},
		{name: "session end", provider: ProviderClaudeCode, payload: `{"hook_event_name":"SessionEnd","session_id":"claude-session","cwd":"` + repository + `"}`},
		{name: "unmatched", provider: ProviderCodex, payload: `{"hook_event_name":"UserPromptSubmit","session_id":"other","cwd":"` + root + `"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			client.mu.Lock()
			client.requests = nil
			client.mu.Unlock()
			if err := service.Ingest(context.Background(), test.provider, strings.NewReader(test.payload)); err != nil {
				t.Fatal(err)
			}
			if got := client.requestTypes(); len(got) != 1 || got[0] != agent.RequestGetSnapshot {
				t.Fatalf("excluded requests = %v", got)
			}
		})
	}
}

type fakeAgentClient struct {
	mu       sync.Mutex
	snapshot model.Snapshot
	err      error
	requests []agent.Request
}

func (f *fakeAgentClient) Request(_ context.Context, request agent.Request) (agent.Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, request)
	if f.err != nil {
		return agent.Event{}, f.err
	}
	if request.Type == agent.RequestGetSnapshot {
		return agent.Event{Snapshot: &f.snapshot}, nil
	}
	return agent.Event{}, nil
}

func (f *fakeAgentClient) requestTypes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.requests))
	for index, request := range f.requests {
		result[index] = request.Type
	}
	return result
}

func matchingSnapshot(repository, worktree string) model.Snapshot {
	return model.Snapshot{
		Projects: []model.Project{{Name: "repo", Path: repository, GitHub: "owner/repo", FollowState: model.FollowFollowing}},
		Lanes:    []model.Lane{{ID: "lane-31", GitHub: "owner/repo", Worktree: &model.Worktree{Path: worktree}}},
	}
}

func mkdir(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
