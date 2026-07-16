package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"text/tabwriter"

	"github.com/jamesonstone/beacon/internal/notes"
	"github.com/spf13/cobra"
)

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
