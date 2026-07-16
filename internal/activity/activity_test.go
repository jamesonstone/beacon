package activity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestDecodeNormalizesDocumentedProviderEvents(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		provider     string
		event        string
		notification string
		action       Action
		state        string
	}{
		{name: "codex prompt", provider: ProviderCodex, event: "UserPromptSubmit", action: ActionUpsert, state: StateWorking},
		{name: "codex permission", provider: ProviderCodex, event: "PermissionRequest", action: ActionUpsert, state: StateNeedsAttention},
		{name: "codex stop", provider: ProviderCodex, event: "Stop", action: ActionUpsert, state: StateTurnFinished},
		{name: "claude permission notification", provider: ProviderClaudeCode, event: "Notification", notification: "permission_prompt", action: ActionUpsert, state: StateNeedsAttention},
		{name: "claude idle notification", provider: ProviderClaudeCode, event: "Notification", notification: "idle_prompt", action: ActionUpsert, state: StateNeedsAttention},
		{name: "claude elicitation notification", provider: ProviderClaudeCode, event: "Notification", notification: "elicitation_dialog", action: ActionUpsert, state: StateNeedsAttention},
		{name: "claude input notification", provider: ProviderClaudeCode, event: "Notification", notification: "agent_needs_input", action: ActionUpsert, state: StateNeedsAttention},
		{name: "claude ignored notification", provider: ProviderClaudeCode, event: "Notification", notification: "auth_success", action: ActionNone},
		{name: "claude stop failure", provider: ProviderClaudeCode, event: "StopFailure", action: ActionRemove},
		{name: "claude session end", provider: ProviderClaudeCode, event: "SessionEnd", action: ActionRemove},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := map[string]any{
				"hook_event_name": test.event, "session_id": "opaque-session", "cwd": "/tmp/repo",
				"notification_type": test.notification, "prompt": "never retain me", "transcript_path": "/secret/transcript",
			}
			contents, _ := json.Marshal(payload)
			event, err := Decode(test.provider, bytes.NewReader(contents), now)
			if err != nil {
				t.Fatal(err)
			}
			if event.Action != test.action || event.State != test.state || !event.WellFormed {
				t.Fatalf("event = %#v", event)
			}
			if event.SessionKey == "opaque-session" || len(event.SessionKey) != 64 {
				t.Fatalf("session key = %q", event.SessionKey)
			}
			encoded, _ := json.Marshal(event)
			for _, forbidden := range []string{"never retain me", "transcript", "opaque-session"} {
				if bytes.Contains(encoded, []byte(forbidden)) {
					t.Fatalf("normalized event retained %q: %s", forbidden, encoded)
				}
			}
		})
	}
}

func TestDecodeRejectsMalformedAndOversizedInput(t *testing.T) {
	if _, err := Decode(ProviderCodex, strings.NewReader(`{"hook_event_name":"Stop"}`), time.Now()); err == nil {
		t.Fatal("missing required fields accepted")
	}
	if _, err := Decode(ProviderCodex, strings.NewReader(strings.Repeat("x", MaxHookBytes+1)), time.Now()); err == nil {
		t.Fatal("oversized input accepted")
	}
	if _, err := Decode(ProviderCodex, strings.NewReader(`{"hook_event_name":"SessionEnd","session_id":"s","cwd":"/tmp"}`), time.Now()); err == nil {
		t.Fatal("undocumented Codex event accepted")
	}
}

func TestMatchUsesLongestWorktreeThenRepositoryFallback(t *testing.T) {
	root := t.TempDir()
	repository := mkdir(t, filepath.Join(root, "repo"))
	worktree := mkdir(t, filepath.Join(repository, "worktrees", "GH-31"))
	nested := mkdir(t, filepath.Join(worktree, "nested"))
	snapshot := matchingSnapshot(repository, worktree)
	target, err := Match(snapshot, nested)
	if err != nil || target.LaneID != "lane-31" || target.ProjectID != "owner/repo" {
		t.Fatalf("worktree target = %#v, %v", target, err)
	}
	other := mkdir(t, filepath.Join(repository, "docs"))
	target, err = Match(snapshot, other)
	if err != nil || target.LaneID != "" || target.ProjectID != "owner/repo" {
		t.Fatalf("repository target = %#v, %v", target, err)
	}
}

func TestMatchRefusesAmbiguousBoundaryMissingAndNonFollowedPaths(t *testing.T) {
	root := t.TempDir()
	repository := mkdir(t, filepath.Join(root, "repo"))
	worktree := mkdir(t, filepath.Join(repository, "worktree"))
	snapshot := matchingSnapshot(repository, worktree)
	snapshot.Lanes = append(snapshot.Lanes, model.Lane{ID: "duplicate", GitHub: "owner/repo", Worktree: &model.Worktree{Path: worktree}})
	if _, err := Match(snapshot, worktree); !errors.Is(err, ErrUnmatched) {
		t.Fatalf("duplicate worktree error = %v", err)
	}
	boundary := mkdir(t, filepath.Join(root, "repo-other"))
	if _, err := Match(matchingSnapshot(repository, worktree), boundary); !errors.Is(err, ErrUnmatched) {
		t.Fatalf("boundary error = %v", err)
	}
	if _, err := Match(matchingSnapshot(repository, worktree), filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing cwd accepted")
	}
	nonFollowed := matchingSnapshot(repository, worktree)
	nonFollowed.Projects[0].FollowState = model.FollowQuiet
	if _, err := Match(nonFollowed, worktree); !errors.Is(err, ErrUnmatched) {
		t.Fatalf("non-followed error = %v", err)
	}
	second := mkdir(t, filepath.Join(repository, "nested-repository"))
	ambiguous := matchingSnapshot(repository, worktree)
	ambiguous.Projects = append(ambiguous.Projects, model.Project{GitHub: "owner/nested", Path: second, FollowState: model.FollowFollowing})
	if _, err := Match(ambiguous, second); !errors.Is(err, ErrUnmatched) {
		t.Fatalf("repository ambiguity error = %v", err)
	}
}

func TestStoreSupersedesSessionsPrunesPhysicallyAndCoalescesRefresh(t *testing.T) {
	directory := t.TempDir()
	store := Store{Path: filepath.Join(directory, "activity.json"), LockPath: filepath.Join(directory, "activity.lock")}
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	target := Target{ProjectID: "owner/repo", LaneID: "lane-31"}
	working := Event{Provider: ProviderCodex, SessionKey: "hash-a", State: StateWorking, Action: ActionUpsert, ObservedAt: now}
	snapshot, refresh, err := store.Apply(context.Background(), working, target, now)
	if err != nil || refresh || len(snapshot.Records) != 1 || snapshot.NextExpiry != now.Add(2*time.Hour) {
		t.Fatalf("working apply = %#v, refresh=%t, err=%v", snapshot, refresh, err)
	}
	attention := working
	attention.State, attention.ObservedAt = StateNeedsAttention, now.Add(time.Minute)
	snapshot, _, err = store.Apply(context.Background(), attention, target, now.Add(time.Minute))
	if err != nil || len(snapshot.Records) != 1 || snapshot.Records[0].State != StateNeedsAttention {
		t.Fatalf("attention supersession = %#v, %v", snapshot, err)
	}
	stale := working
	stale.ObservedAt = now.Add(30 * time.Second)
	snapshot, _, err = store.Apply(context.Background(), stale, target, now.Add(2*time.Minute))
	if err != nil || len(snapshot.Records) != 1 || snapshot.Records[0].State != StateNeedsAttention {
		t.Fatalf("out-of-order event replaced latest state = %#v, %v", snapshot, err)
	}
	second := Event{Provider: ProviderClaudeCode, SessionKey: "hash-b", State: StateWorking, Action: ActionUpsert, ObservedAt: now.Add(2 * time.Minute)}
	snapshot, _, _ = store.Apply(context.Background(), second, target, now.Add(2*time.Minute))
	if len(snapshot.Records) != 2 {
		t.Fatalf("concurrent records = %#v", snapshot.Records)
	}
	stop := working
	stop.State, stop.ObservedAt = StateTurnFinished, now.Add(3*time.Minute)
	_, refresh, _ = store.Apply(context.Background(), stop, target, now.Add(3*time.Minute))
	if !refresh {
		t.Fatal("first Stop did not request refresh")
	}
	_, refresh, _ = store.Apply(context.Background(), stop, target, now.Add(3*time.Minute+time.Second))
	if refresh {
		t.Fatal("duplicate Stop requested refresh")
	}
	snapshot, err = store.Prune(context.Background(), now.Add(4*time.Hour))
	if err != nil || len(snapshot.Records) != 0 || !snapshot.NextExpiry.IsZero() {
		t.Fatalf("pruned snapshot = %#v, %v", snapshot, err)
	}
	contents, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"hash-a", "hash-b", "needs_attention", "turn_finished"} {
		if bytes.Contains(contents, []byte(forbidden)) {
			t.Fatalf("expired cache retained %q: %s", forbidden, contents)
		}
	}
	if info, err := os.Stat(store.Path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("cache mode = %v, %v", info.Mode().Perm(), err)
	}
}

func TestStoreRemoveAndBoundedLockContention(t *testing.T) {
	directory := t.TempDir()
	store := Store{Path: filepath.Join(directory, "activity.json"), LockPath: filepath.Join(directory, "activity.lock"), LockWait: 25 * time.Millisecond}
	now := time.Now()
	event := Event{Provider: ProviderClaudeCode, SessionKey: "hash", State: StateWorking, Action: ActionUpsert, ObservedAt: now}
	_, _, _ = store.Apply(context.Background(), event, Target{ProjectID: "owner/repo"}, now)
	event.Action, event.State = ActionRemove, ""
	snapshot, refresh, err := store.Apply(context.Background(), event, Target{ProjectID: "owner/repo"}, now)
	if err != nil || refresh || len(snapshot.Records) != 0 {
		t.Fatalf("remove = %#v, refresh=%t, err=%v", snapshot, refresh, err)
	}
	lock, err := os.OpenFile(store.LockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	started := time.Now()
	if _, err := store.List(context.Background(), now); err == nil {
		t.Fatal("lock contention succeeded")
	}
	if elapsed := time.Since(started); elapsed > 150*time.Millisecond {
		t.Fatalf("lock contention took %s", elapsed)
	}
}

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
