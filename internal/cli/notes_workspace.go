package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/notes"
	"github.com/spf13/cobra"
)

func (a App) notesOpenCommand(configPath *string) *cobra.Command {
	return a.notesLifecycleCommand(configPath, "open", "Open or activate a signal note", agent.RequestOpenNote)
}

func (a App) notesCloseCommand(configPath *string) *cobra.Command {
	return a.notesLifecycleCommand(configPath, "close", "Close a signal note without deleting it", agent.RequestCloseNote)
}

func (a App) notesDeleteCommand(configPath *string) *cobra.Command {
	return a.notesLifecycleCommand(configPath, "delete", "Permanently delete a detail signal note", agent.RequestDeleteNote)
}

func (a App) notesLifecycleCommand(configPath *string, name, short, requestType string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: name + " <id-or-exact-title>", Short: short, Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := a.mutateNotesWorkspace(cmd.Context(), *configPath, agent.Request{Type: requestType, NoteID: args[0]})
			if err != nil {
				return err
			}
			if jsonOutput {
				return encodeJSON(a.Out, workspace)
			}
			pastTense := map[string]string{
				"open": "opened", "close": "closed", "delete": "deleted",
			}[name]
			if pastTense == "" {
				pastTense = name + "d"
			}
			_, err = fmt.Fprintf(a.Out, "%s signal note %s\n", pastTense, args[0])
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes workspace as JSON")
	return command
}

func (a App) mutateNotesWorkspace(ctx context.Context, configPath string, request agent.Request) (notes.Workspace, error) {
	useAgent, err := a.notesAgentSupportsWorkspace(ctx, configPath)
	if err != nil {
		return notes.Workspace{}, err
	}
	if useAgent {
		event, requestErr := a.requestNotes(ctx, configPath, request)
		if requestErr == nil {
			if event.NotesWorkspace != nil {
				return *event.NotesWorkspace, nil
			}
			if unsupportedNotesRequest(event, request.Type) {
				return a.mutateNotesWorkspaceDirect(request)
			}
			if event.Message != "" {
				return notes.Workspace{}, errors.New(event.Message)
			}
			return notes.Workspace{}, errors.New("Beacon agent returned no notes workspace")
		}
		if !errors.Is(requestErr, agent.ErrUnavailable) {
			return notes.Workspace{}, requestErr
		}
	}
	return a.mutateNotesWorkspaceDirect(request)
}

func (a App) mutateNotesWorkspaceDirect(request agent.Request) (notes.Workspace, error) {
	path, err := notes.ResolvePath()
	if err != nil {
		return notes.Workspace{}, err
	}
	store := notes.FileStore{}
	switch request.Type {
	case agent.RequestCreateNote:
		return store.CreateNote(path, request.Content)
	case agent.RequestOpenNote:
		return store.OpenNote(path, request.NoteID)
	case agent.RequestCloseNote:
		return store.CloseNote(path, request.NoteID)
	case agent.RequestDeleteNote:
		return store.DeleteNote(path, request.NoteID)
	case agent.RequestSetNotePinned:
		return store.SetNotePinned(path, request.NoteID, request.Pinned)
	case agent.RequestReorderPinned:
		return store.ReorderPinnedNotes(path, request.NoteIDs)
	default:
		return notes.Workspace{}, fmt.Errorf("unsupported notes workspace mutation: %s", request.Type)
	}
}

func (a App) notesAgentSupportsWorkspace(ctx context.Context, configPath string) (bool, error) {
	event, err := a.requestNotes(ctx, configPath, agent.Request{Type: agent.RequestGetNotesWorkspace})
	if errors.Is(err, agent.ErrUnavailable) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if event.NotesWorkspace != nil {
		return true, nil
	}
	if unsupportedNotesRequest(event, agent.RequestGetNotesWorkspace) {
		return false, nil
	}
	if event.Message != "" {
		return false, errors.New(event.Message)
	}
	return false, errors.New("Beacon agent returned no notes workspace")
}

func unsupportedNotesRequest(event agent.Event, requestType string) bool {
	return event.Type == agent.EventProjectFailed && event.Message == "unknown agent request: "+requestType
}

func (a App) requestNotes(ctx context.Context, configPath string, request agent.Request) (agent.Event, error) {
	resolvedConfig, err := config.ResolvePath(configPath)
	if err != nil {
		return agent.Event{}, err
	}
	paths, err := agent.ResolvePaths(resolvedConfig)
	if err != nil {
		return agent.Event{}, err
	}
	return a.agentClient(paths.Socket).Request(ctx, request)
}

func (a App) noteInput(args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	if a.inputIsTTY() {
		return "", usageError{fmt.Errorf("provide Markdown text or pipe it on standard input")}
	}
	contents, err := io.ReadAll(io.LimitReader(a.input(), notes.MaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read Markdown notes from standard input: %w", err)
	}
	if len(contents) > notes.MaxBytes {
		return "", fmt.Errorf("Beacon notes exceed the %d-byte limit", notes.MaxBytes)
	}
	return string(contents), nil
}
