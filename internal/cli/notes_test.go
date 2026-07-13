package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/notes"
)

func TestNotesCommandsShareMarkdownDocument(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	executeNotesCommand(t, "", "notes", "set", "# Signal Log\n\nFirst thought")
	executeNotesCommand(t, "- another thought", "notes", "append")
	output := executeNotesCommand(t, "", "notes")
	want := "# Signal Log\n\nFirst thought\n- another thought\n"
	if output != want {
		t.Fatalf("notes output = %q, want %q", output, want)
	}

	jsonOutput := executeNotesCommand(t, "", "notes", "show", "--json")
	var document notes.Document
	if err := json.Unmarshal([]byte(jsonOutput), &document); err != nil {
		t.Fatal(err)
	}
	if document.Content != want || document.Path != filepath.Join(root, "beacon", "notes.md") || document.UpdatedAt.IsZero() {
		t.Fatalf("document = %#v", document)
	}
}

func TestNotesPathAndEditUseLocalMarkdownFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)
	path := filepath.Join(root, "beacon", "notes.md")
	runner := &notesRunner{}
	var output bytes.Buffer
	app := App{
		In: bytes.NewBuffer(nil), Out: &output, Err: &bytes.Buffer{}, Runner: runner,
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
		agentClientSource: unavailableNotesAgent,
	}
	command := app.Root()
	command.SetArgs([]string{"notes", "edit"})
	editErr := command.ExecuteContext(context.Background())
	if runtime.GOOS == "darwin" {
		if editErr != nil {
			t.Fatal(editErr)
		}
		if runner.name != "open" || len(runner.args) != 3 || runner.args[0] != "-W" || runner.args[1] != "-t" || runner.args[2] != path {
			t.Fatalf("editor command = %s %v", runner.name, runner.args)
		}
	} else {
		if editErr == nil || !strings.Contains(editErr.Error(), "unsupported") {
			t.Fatalf("edit error = %v", editErr)
		}
		if runner.name != "" {
			t.Fatalf("unsupported editor invoked %s %v", runner.name, runner.args)
		}
	}

	output.Reset()
	command = app.Root()
	command.SetArgs([]string{"notes", "path"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if output.String() != path+"\n" {
		t.Fatalf("path output = %q", output.String())
	}
}

func TestNotesSetWithoutArgumentsRequiresTextInTTY(t *testing.T) {
	app := App{
		In: bytes.NewBuffer(nil), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &notesRunner{},
		InputIsTTY: func() bool { return true }, OutputIsTTY: func() bool { return true },
		agentClientSource: unavailableNotesAgent,
	}
	command := app.Root()
	command.SetArgs([]string{"notes", "set"})
	if err := command.ExecuteContext(context.Background()); err == nil || ExitCode(err) != 2 {
		t.Fatalf("error = %v", err)
	}
}

func executeNotesCommand(t *testing.T, input string, args ...string) string {
	t.Helper()
	var output bytes.Buffer
	app := App{
		In: bytes.NewBufferString(input), Out: &output, Err: &bytes.Buffer{}, Runner: &notesRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
		agentClientSource: unavailableNotesAgent,
	}
	command := app.Root()
	command.SetArgs(args)
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(args) > 1 && (args[1] == "set" || args[1] == "append") {
		return ""
	}
	return output.String()
}

func TestNotesWritesThroughRunningAgent(t *testing.T) {
	document := notes.Document{Content: "# Through the agent", Path: "/tmp/notes.md"}
	client := &notesAgentClient{event: agent.Event{Type: agent.EventNotesUpdated, Notes: &document}}
	var output bytes.Buffer
	app := App{
		In: bytes.NewBuffer(nil), Out: &output, Err: &bytes.Buffer{}, Runner: &notesRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return false },
		agentClientSource: func(string) agentRequestClient { return client },
	}
	command := app.Root()
	command.SetArgs([]string{"notes", "set", "# Through the agent"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 1 || client.requests[0].Type != agent.RequestSetNotes || client.requests[0].Content != document.Content {
		t.Fatalf("requests = %#v", client.requests)
	}
}

func unavailableNotesAgent(string) agentRequestClient {
	return &notesAgentClient{err: fmt.Errorf("%w: test socket", agent.ErrUnavailable)}
}

type notesAgentClient struct {
	event    agent.Event
	err      error
	requests []agent.Request
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
