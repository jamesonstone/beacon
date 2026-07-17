package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/notes"
	"github.com/spf13/cobra"
)

func (a App) notesCommand(configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "notes", Short: "Read and edit the local Markdown signal log", Args: noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return a.showNotes(notes.GeneralID, jsonOutput)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes document as JSON")
	command.AddCommand(
		a.notesShowCommand(),
		a.notesWriteCommand(configPath, "set", false),
		a.notesWriteCommand(configPath, "append", true),
		a.notesEditCommand(configPath),
		a.notesPathCommand(),
		a.notesListCommand(),
		a.notesNewCommand(configPath),
		a.notesOpenCommand(configPath),
		a.notesCloseCommand(configPath),
		a.notesDeleteCommand(configPath),
		a.notesPinCommand(configPath, true),
		a.notesPinCommand(configPath, false),
		a.notesReorderPinnedCommand(configPath),
	)
	return command
}

func (a App) notesShowCommand() *cobra.Command {
	var jsonOutput bool
	var selector string
	command := &cobra.Command{
		Use: "show", Short: "Print a Markdown signal note", Args: noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return a.showNotes(selector, jsonOutput)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes document as JSON")
	command.Flags().StringVar(&selector, "note", notes.GeneralID, "note ID or unique exact title")
	return command
}

func (a App) showNotes(selector string, jsonOutput bool) error {
	path, err := notes.ResolvePath()
	if err != nil {
		return err
	}
	document, err := (notes.FileStore{}).LoadNote(path, selector)
	if err != nil {
		return err
	}
	if jsonOutput {
		encoder := json.NewEncoder(a.Out)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(document)
	}
	_, err = io.WriteString(a.Out, document.Content)
	return err
}

func (a App) notesWriteCommand(configPath *string, name string, appendContent bool) *cobra.Command {
	var jsonOutput bool
	var selector string
	command := &cobra.Command{
		Use: name + " [markdown]", Short: commandTitle(name) + " a Markdown signal note", Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := a.noteInput(args)
			if err != nil {
				return err
			}
			document, err := a.writeNotes(cmd.Context(), *configPath, selector, content, appendContent)
			if err != nil {
				return err
			}
			if jsonOutput {
				return encodeJSON(a.Out, document)
			}
			_, err = fmt.Fprintf(a.Out, "saved signal note %s\n", document.ID)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the saved notes document as JSON")
	command.Flags().StringVar(&selector, "note", notes.GeneralID, "note ID or unique exact title")
	return command
}

func (a App) writeNotes(ctx context.Context, configPath, selector, content string, appendContent bool) (notes.Document, error) {
	requestType := agent.RequestSetNotes
	if appendContent {
		requestType = agent.RequestAppendNotes
	}
	useAgent := true
	var err error
	if selector != notes.GeneralID {
		useAgent, err = a.notesAgentSupportsWorkspace(ctx, configPath)
		if err != nil {
			return notes.Document{}, err
		}
	}
	if useAgent {
		noteID := selector
		if selector == notes.GeneralID {
			noteID = ""
		}
		event, requestErr := a.requestNotes(ctx, configPath, agent.Request{Type: requestType, NoteID: noteID, Content: content})
		if requestErr == nil {
			if event.Notes != nil {
				return *event.Notes, nil
			}
			if event.Message != "" {
				return notes.Document{}, errors.New(event.Message)
			}
			return notes.Document{}, errors.New("Beacon agent returned no notes document")
		}
		if !errors.Is(requestErr, agent.ErrUnavailable) {
			return notes.Document{}, requestErr
		}
	}
	return a.writeNotesDirect(selector, content, appendContent)
}

func (a App) writeNotesDirect(selector, content string, appendContent bool) (notes.Document, error) {
	path, err := notes.ResolvePath()
	if err != nil {
		return notes.Document{}, err
	}
	store := notes.FileStore{}
	selected, err := store.LoadNote(path, selector)
	if err != nil {
		return notes.Document{}, err
	}
	if appendContent {
		_, err = store.AppendNote(path, selected.ID, content)
	} else {
		_, err = store.WriteNote(path, selected.ID, content)
	}
	if err != nil {
		return notes.Document{}, err
	}
	return store.LoadNote(path, selected.ID)
}

func (a App) notesListCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "list", Short: "List General, New Tab, and detail signal notes", Args: noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			path, err := notes.ResolvePath()
			if err != nil {
				return err
			}
			workspace, err := (notes.FileStore{}).LoadWorkspace(path)
			if err != nil {
				return err
			}
			if jsonOutput {
				return encodeJSON(a.Out, workspace)
			}
			return printNotesWorkspace(a.Out, workspace)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes workspace as JSON")
	return command
}

func (a App) notesNewCommand(configPath *string) *cobra.Command {
	var fromLine int
	var jsonOutput bool
	command := &cobra.Command{
		Use: "new [markdown]", Short: "Create and open a detail signal note", Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := a.newNoteInput(args, fromLine)
			if err != nil {
				return err
			}
			workspace, err := a.mutateNotesWorkspace(cmd.Context(), *configPath, agent.Request{Type: agent.RequestCreateNote, Content: content})
			if err != nil {
				return err
			}
			if jsonOutput {
				return encodeJSON(a.Out, workspace)
			}
			_, err = fmt.Fprintf(a.Out, "opened signal note %s\n", workspace.ActiveID)
			return err
		},
	}
	command.Flags().IntVar(&fromLine, "from-line", 0, "copy a one-based line from General as the title")
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes workspace as JSON")
	return command
}

func (a App) newNoteInput(args []string, fromLine int) (string, error) {
	if fromLine < 0 {
		return "", usageError{fmt.Errorf("--from-line must be greater than zero")}
	}
	if fromLine > 0 {
		if len(args) > 0 {
			return "", usageError{fmt.Errorf("--from-line cannot be combined with Markdown arguments")}
		}
		path, err := notes.ResolvePath()
		if err != nil {
			return "", err
		}
		general, err := (notes.FileStore{}).LoadNote(path, notes.GeneralID)
		if err != nil {
			return "", err
		}
		lines := strings.Split(general.Content, "\n")
		if fromLine > len(lines) {
			return "", usageError{fmt.Errorf("General has no line %d", fromLine)}
		}
		title := strings.TrimSpace(lines[fromLine-1])
		if title == "" {
			return "", usageError{fmt.Errorf("General line %d is empty", fromLine)}
		}
		return title + "\n\n", nil
	}
	content, err := a.noteInput(args)
	if err != nil {
		return "", err
	}
	if len(args) > 0 && !strings.Contains(content, "\n") {
		content += "\n\n"
	}
	return content, nil
}
