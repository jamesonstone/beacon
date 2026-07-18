package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/jamesonstone/beacon/internal/agent"
)

func TestReorderCommandSendsCompleteLaneOrder(t *testing.T) {
	repository := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath, "version: 2\nrepositories:\n  - name: beacon\n    path: "+repository+"\n    github: owner/beacon\n")
	client := &recordingLaneAgent{event: agent.Event{Type: agent.EventWorkingSetChanged}}
	var output bytes.Buffer
	app := App{
		Out: &output, Err: &bytes.Buffer{}, autoStartAgent: false,
		agentClientSource: func(string) agentRequestClient { return client },
	}
	command := app.Root()
	command.SetArgs([]string{"--config", configPath, "reorder", "lane-b", "lane-a"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 1 || client.requests[0].Type != agent.RequestReorderLanes {
		t.Fatalf("requests = %#v", client.requests)
	}
	if actual := client.requests[0].LaneIDs; len(actual) != 2 || actual[0] != "lane-b" || actual[1] != "lane-a" {
		t.Fatalf("lane IDs = %#v", actual)
	}
	if output.String() != "reordered working-set lanes\n" {
		t.Fatalf("output = %q", output.String())
	}
}

type recordingLaneAgent struct {
	event    agent.Event
	requests []agent.Request
}

func (c *recordingLaneAgent) Request(_ context.Context, request agent.Request) (agent.Event, error) {
	c.requests = append(c.requests, request)
	event := c.event
	event.RequestID = request.RequestID
	event.ProtocolVersion = agent.ProtocolVersion
	return event, nil
}
