package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/notes"
)

func TestNotesWorkspaceMutationsUseRunningAgent(t *testing.T) {
	workspace := notes.Workspace{Version: 1, ActiveID: "detail", OpenIDs: []string{notes.GeneralID, "detail"}}
	client := &notesAgentClient{event: agent.Event{Type: agent.EventWorkspaceUpdated, NotesWorkspace: &workspace}}
	app := App{
		In: bytes.NewBuffer(nil), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &notesRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
		agentClientSource: func(string) agentRequestClient { return client },
	}
	command := app.Root()
	command.SetArgs([]string{"notes", "open", "detail", "--json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 2 ||
		client.requests[0].Type != agent.RequestGetNotesWorkspace ||
		client.requests[1].Type != agent.RequestOpenNote || client.requests[1].NoteID != "detail" {
		t.Fatalf("requests = %#v", client.requests)
	}
}

func TestNotesWorkspaceMutationsFallbackWhenAgentLacksWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)
	client := &notesAgentClient{event: agent.Event{
		Type: agent.EventProjectFailed, Message: "unknown agent request: " + agent.RequestGetNotesWorkspace,
	}}
	var output bytes.Buffer
	app := App{
		In: bytes.NewBuffer(nil), Out: &output, Err: &bytes.Buffer{}, Runner: &notesRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
		agentClientSource: func(string) agentRequestClient { return client },
	}
	command := app.Root()
	command.SetArgs([]string{"notes", "new", "Legacy detail", "--json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	var workspace notes.Workspace
	if err := json.Unmarshal(output.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	if workspace.Active == nil || workspace.Active.Content != "Legacy detail\n\n" {
		t.Fatalf("workspace = %#v", workspace)
	}
	if len(client.requests) != 1 || client.requests[0].Type != agent.RequestGetNotesWorkspace {
		t.Fatalf("requests = %#v", client.requests)
	}
}

func TestNotesDeleteFallsBackWhenCapableAgentLacksDeleteRequest(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)
	path := filepath.Join(root, "beacon", "notes.md")
	created, err := (notes.FileStore{}).CreateNote(path, "Legacy deletion\nbody")
	if err != nil {
		t.Fatal(err)
	}
	client := &sequencedNotesAgentClient{events: []agent.Event{
		{Type: agent.EventNotesWorkspace, NotesWorkspace: &created},
		{Type: agent.EventProjectFailed, Message: "unknown agent request: " + agent.RequestDeleteNote},
	}}
	var output bytes.Buffer
	app := App{
		In: bytes.NewBuffer(nil), Out: &output, Err: &bytes.Buffer{}, Runner: &notesRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
		agentClientSource: func(string) agentRequestClient { return client },
	}
	command := app.Root()
	command.SetArgs([]string{"notes", "delete", created.ActiveID, "--json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	var workspace notes.Workspace
	if err := json.Unmarshal(output.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	if containsString(workspace.OpenIDs, created.ActiveID) {
		t.Fatalf("workspace = %#v", workspace)
	}
	if len(client.requests) != 2 || client.requests[1].Type != agent.RequestDeleteNote {
		t.Fatalf("requests = %#v", client.requests)
	}
	if _, err := os.Lstat(created.Active.Path); !os.IsNotExist(err) {
		t.Fatalf("deleted note still exists: %v", err)
	}
}

func TestNotesDetailWriteFallbackProbesOlderAgentBeforeMutation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)
	path := filepath.Join(root, "beacon", "notes.md")
	workspace, err := (notes.FileStore{}).CreateNote(path, "Original\nbody")
	if err != nil {
		t.Fatal(err)
	}
	client := &notesAgentClient{event: agent.Event{
		Type: agent.EventProjectFailed, Message: "unknown agent request: " + agent.RequestGetNotesWorkspace,
	}}
	app := App{
		In: bytes.NewBuffer(nil), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &notesRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
		agentClientSource: func(string) agentRequestClient { return client },
	}
	command := app.Root()
	command.SetArgs([]string{"notes", "set", "Updated", "--note", workspace.ActiveID})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	document, err := (notes.FileStore{}).LoadNote(path, workspace.ActiveID)
	if err != nil {
		t.Fatal(err)
	}
	if document.Content != "Updated" {
		t.Fatalf("content = %q", document.Content)
	}
	if len(client.requests) != 1 || client.requests[0].Type != agent.RequestGetNotesWorkspace {
		t.Fatalf("requests = %#v", client.requests)
	}
}

func TestNotesWorkspaceMutationDoesNotHideSupportedAgentFailure(t *testing.T) {
	client := &notesAgentClient{event: agent.Event{Type: agent.EventProjectFailed, Message: "workspace failed"}}
	app := App{
		In: bytes.NewBuffer(nil), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &notesRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
		agentClientSource: func(string) agentRequestClient { return client },
	}
	command := app.Root()
	command.SetArgs([]string{"notes", "new", "must not persist"})
	if err := command.ExecuteContext(context.Background()); err == nil || !strings.Contains(err.Error(), "workspace failed") {
		t.Fatalf("error = %v", err)
	}
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func unavailableNotesAgent(string) agentRequestClient {
	return &notesAgentClient{err: fmt.Errorf("%w: test socket", agent.ErrUnavailable)}
}

type notesAgentClient struct {
	event    agent.Event
	err      error
	requests []agent.Request
}

type sequencedNotesAgentClient struct {
	events   []agent.Event
	requests []agent.Request
}

func (c *sequencedNotesAgentClient) Request(_ context.Context, request agent.Request) (agent.Event, error) {
	c.requests = append(c.requests, request)
	if len(c.events) == 0 {
		return agent.Event{}, errors.New("unexpected agent request")
	}
	event := c.events[0]
	c.events = c.events[1:]
	return event, nil
}

func (c *notesAgentClient) Request(_ context.Context, request agent.Request) (agent.Event, error) {
	c.requests = append(c.requests, request)
	return c.event, c.err
}

type notesRunner struct {
	name string
	args []string
}

func (r *notesRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.name = name
	r.args = append([]string{}, args...)
	return nil, nil
}
