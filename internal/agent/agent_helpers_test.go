package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

type mutableProber struct {
	mutex  sync.Mutex
	result ProbeResult
}

type countingProber struct{ calls atomic.Int32 }

func (p *countingProber) Probe(context.Context, config.Repository, string) (ProbeResult, error) {
	p.calls.Add(1)
	return ProbeResult{}, nil
}

type probeCommandRunner struct{ commands []string }

type countingRemoteCollector struct {
	calls int
	size  int
}

func (c *countingRemoteCollector) Collect(_ context.Context, repositories []config.Repository, _, _ string, _ int) model.RemoteCollection {
	c.calls++
	c.size = len(repositories)
	collection := model.RemoteCollection{
		Repositories: make(map[string]model.RemoteEvidence, len(repositories)),
		Errors:       []model.ScanError{}, Warnings: []model.ScanError{},
	}
	for _, repository := range repositories {
		collection.Repositories[repository.GitHub] = model.RemoteEvidence{
			PullRequests: []model.PullRequest{}, Issues: []model.Issue{},
			Errors: []model.ScanError{}, Warnings: []model.ScanError{},
		}
	}
	return collection
}

func (r *probeCommandRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.commands = append(r.commands, strings.TrimSpace(name+" "+strings.Join(args, " ")))
	switch fmt.Sprint(append([]string{name}, args...)) {
	case "[git rev-parse HEAD]":
		return []byte("head\n"), nil
	case "[git status --porcelain=v2 --branch -z]":
		return []byte("# branch.head main\x00"), nil
	case "[git for-each-ref --format=%(refname:short)%00%(objectname) refs/heads]":
		return []byte("main\x00head\n"), nil
	default:
		if name == "gh" {
			return []byte("[]"), nil
		}
		return nil, fmt.Errorf("unexpected command: %s %v", name, args)
	}
}

type lifecycleCommandRunner struct{ commands []string }

func (r *lifecycleCommandRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.commands = append(r.commands, name+" "+strings.Join(args, " "))
	return nil, nil
}

func (p *mutableProber) Probe(context.Context, config.Repository, string) (ProbeResult, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.result, nil
}

func (p *mutableProber) set(result ProbeResult) {
	p.mutex.Lock()
	p.result = result
	p.mutex.Unlock()
}

func cachedRecord(id string, revision uint64, state model.TrackingState) ProjectRecord {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	project := model.Project{Name: "repo", Path: "/repo", GitHub: id, Base: "main", Remote: "origin", TrackingState: state, LaneIDs: []string{"git:" + id + "@main"}, Errors: []model.ScanError{}, Warnings: []model.ScanError{}}
	lane := model.Lane{ID: project.LaneIDs[0], Repository: project.Name, GitHub: id, Base: "main", Branch: "main", Worktree: &model.Worktree{Path: "/repo", HeadOID: "head", StatusHash: "status", UpdatedAt: now}, Signals: model.Signals{Worktree: model.WorktreeClean, Publication: model.PublicationBase, Freshness: model.FreshnessCurrent}, NextAction: model.ActionNone, UpdatedAt: now, Reasons: []string{}, Warnings: []string{}, Blockers: []string{}}
	snapshot := model.Snapshot{SchemaVersion: model.SchemaVersion, GeneratedAt: now, ConfigPath: "/config.yaml", Projects: []model.Project{project}, Lanes: []model.Lane{lane}, Refresh: []model.Refresh{}, Errors: []model.ScanError{}, Warnings: []model.ScanError{}}
	return ProjectRecord{Version: CacheVersion, ProjectID: id, Revision: revision, Stage: "ready", UpdatedAt: now, Snapshot: snapshot}
}

func testPaths(root string) Paths {
	return Paths{Config: filepath.Join(root, "config.yaml"), State: filepath.Join(root, "state", "beacon", "tracking.json"), Notes: filepath.Join(root, "data", "beacon", "notes.md"), CacheRoot: filepath.Join(root, "cache"), Projects: filepath.Join(root, "cache", "projects"), Socket: filepath.Join(root, "cache", "agent.sock"), PID: filepath.Join(root, "cache", "agent.pid"), LaunchAgent: filepath.Join(root, "LaunchAgents", "agent.plist"), Logs: filepath.Join(root, "logs"), StandardLog: filepath.Join(root, "logs", "agent.log"), ErrorLog: filepath.Join(root, "logs", "agent-error.log")}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("file did not appear: %s", path)
}

func waitForRefresh(t *testing.T, engine *Engine) {
	t.Helper()
	waitForRefreshWithin(t, engine, 2*time.Second)
}

func waitForRefreshWithin(t *testing.T, engine *Engine, timeout time.Duration) {
	t.Helper()
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); {
		if !engine.Status().Refreshing {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("refresh did not complete")
}
