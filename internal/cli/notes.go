package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
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
	event, err := a.requestNotes(ctx, configPath, agent.Request{Type: requestType, NoteID: selector, Content: content})
	if err == nil {
		if event.Notes != nil {
			return *event.Notes, nil
		}
		if event.Message != "" {
			return notes.Document{}, errors.New(event.Message)
		}
		return notes.Document{}, errors.New("Beacon agent returned no notes document")
	}
	if !errors.Is(err, agent.ErrUnavailable) {
		return notes.Document{}, err
	}
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

func (a App) notesOpenCommand(configPath *string) *cobra.Command {
	return a.notesLifecycleCommand(configPath, "open", "Open or activate a signal note", agent.RequestOpenNote)
}

func (a App) notesCloseCommand(configPath *string) *cobra.Command {
	return a.notesLifecycleCommand(configPath, "close", "Close a signal note without deleting it", agent.RequestCloseNote)
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
			pastTense := "opened"
			if name == "close" {
				pastTense = "closed"
			}
			_, err = fmt.Fprintf(a.Out, "%s signal note %s\n", pastTense, args[0])
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes workspace as JSON")
	return command
}

func (a App) mutateNotesWorkspace(ctx context.Context, configPath string, request agent.Request) (notes.Workspace, error) {
	event, err := a.requestNotes(ctx, configPath, request)
	if err == nil {
		if event.NotesWorkspace != nil {
			return *event.NotesWorkspace, nil
		}
		if event.Message != "" {
			return notes.Workspace{}, errors.New(event.Message)
		}
		return notes.Workspace{}, errors.New("Beacon agent returned no notes workspace")
	}
	if !errors.Is(err, agent.ErrUnavailable) {
		return notes.Workspace{}, err
	}
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
	default:
		return notes.Workspace{}, fmt.Errorf("unsupported notes workspace mutation: %s", request.Type)
	}
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

func (a App) notesPathCommand() *cobra.Command {
	var selector string
	command := &cobra.Command{
		Use: "path", Short: "Print a local Markdown signal-note path", Args: noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			path, err := notes.ResolvePath()
			if err != nil {
				return err
			}
			document, err := (notes.FileStore{}).LoadNote(path, selector)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.Out, document.Path)
			return err
		},
	}
	command.Flags().StringVar(&selector, "note", notes.GeneralID, "note ID or unique exact title")
	return command
}

func (a App) notesEditCommand(configPath *string) *cobra.Command {
	var selector string
	command := &cobra.Command{
		Use: "edit", Short: "Open a Markdown signal note in an editor", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := notes.ResolvePath()
			if err != nil {
				return err
			}
			store := notes.FileStore{}
			document, err := store.LoadNote(path, selector)
			if err != nil {
				return err
			}
			if runtime.GOOS != "darwin" {
				return fmt.Errorf("beacon notes edit is unsupported on %s; edit %s directly", runtime.GOOS, document.Path)
			}
			if document.UpdatedAt.IsZero() {
				if _, err := store.WriteNote(path, document.ID, document.Content); err != nil {
					return err
				}
			}
			if _, err = a.Runner.Run(cmd.Context(), "", "open", "-W", "-t", document.Path); err != nil {
				return err
			}
			document, err = store.LoadNote(path, document.ID)
			if err != nil {
				return err
			}
			_, err = a.writeNotes(cmd.Context(), *configPath, document.ID, document.Content, false)
			return err
		},
	}
	command.Flags().StringVar(&selector, "note", notes.GeneralID, "note ID or unique exact title")
	return command
}

func printNotesWorkspace(writer io.Writer, workspace notes.Workspace) error {
	open := make(map[string]bool, len(workspace.OpenIDs))
	for _, id := range workspace.OpenIDs {
		open[id] = true
	}
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "ACTIVE\tOPEN\tID\tTITLE\tUPDATED\tPATH"); err != nil {
		return err
	}
	for _, tab := range workspace.Tabs {
		active := ""
		if tab.ID == workspace.ActiveID {
			active = "*"
		}
		opened := ""
		if open[tab.ID] {
			opened = "yes"
		}
		updated := ""
		if !tab.UpdatedAt.IsZero() {
			updated = tab.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n", active, opened, tab.ID, tab.Title, updated, tab.Path); err != nil {
			return err
		}
	}
	return table.Flush()
}

func encodeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
