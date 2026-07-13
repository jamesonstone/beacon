package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestBareDashboardFallsBackToForcedForegroundScanWhenAgentIsUnavailable(t *testing.T) {
	configPath := writeBareDashboardConfig(t)
	scanner := &recordingSnapshotScanner{snapshot: model.Snapshot{SchemaVersion: model.SchemaVersion}}
	app := App{
		Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &recordingRunner{}, scannerSource: scanner,
		OutputIsTTY: func() bool { return false }, TerminalWidth: func() int { return 120 },
		agentClientSource: func(string) agentRequestClient {
			return &scriptedAgentClient{results: []agentClientResult{{err: fmt.Errorf("%w: test socket", agent.ErrUnavailable)}}}
		},
	}
	if err := app.runAgentDashboard(context.Background(), configPath, "never", false); err != nil {
		t.Fatal(err)
	}
	if scanner.calls != 1 || !scanner.refresh {
		t.Fatalf("foreground scan calls=%d refresh=%t", scanner.calls, scanner.refresh)
	}
}

func TestBareDashboardFallsBackWhenAgentStopsDuringManualRefresh(t *testing.T) {
	configPath := writeBareDashboardConfig(t)
	for _, test := range []struct {
		name    string
		results []agentClientResult
	}{
		{
			name: "status polling fails",
			results: []agentClientResult{
				{event: agent.Event{Type: agent.EventProjectQueued, ScanID: "scan-one"}},
				{err: fmt.Errorf("%w: disconnected", agent.ErrUnavailable)},
			},
		},
		{
			name: "final snapshot fails",
			results: []agentClientResult{
				{event: agent.Event{Type: agent.EventProjectQueued, ScanID: "scan-two"}},
				{event: agent.Event{Type: agent.EventAgentStatus, Status: &agent.Status{Running: true, Refreshing: false}}},
				{err: fmt.Errorf("%w: disconnected", agent.ErrUnavailable)},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			scanner := &recordingSnapshotScanner{snapshot: model.Snapshot{SchemaVersion: model.SchemaVersion}}
			client := &scriptedAgentClient{results: test.results}
			app := App{
				Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &recordingRunner{}, scannerSource: scanner,
				OutputIsTTY: func() bool { return false }, TerminalWidth: func() int { return 120 },
				agentClientSource: func(string) agentRequestClient { return client },
			}
			if err := app.runAgentDashboard(context.Background(), configPath, "never", false); err != nil {
				t.Fatal(err)
			}
			if scanner.calls != 1 || !scanner.refresh {
				t.Fatalf("foreground scan calls=%d refresh=%t", scanner.calls, scanner.refresh)
			}
		})
	}
}

func writeBareDashboardConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	configPath := filepath.Join(home, ".config", "beacon", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	writeTestConfig(t, configPath, `version: 2
repositories:
  - name: beacon
    path: `+t.TempDir()+`
    github: owner/beacon
`)
	return configPath
}

type agentClientResult struct {
	event agent.Event
	err   error
}

type scriptedAgentClient struct {
	mutex   sync.Mutex
	results []agentClientResult
}

func (c *scriptedAgentClient) Request(_ context.Context, _ agent.Request) (agent.Event, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(c.results) == 0 {
		return agent.Event{}, errors.New("unexpected agent request")
	}
	result := c.results[0]
	c.results = c.results[1:]
	return result.event, result.err
}
