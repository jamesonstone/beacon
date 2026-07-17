package cli

import (
	"fmt"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/spf13/cobra"
)

func (a App) notesPinCommand(configPath *string, pinned bool) *cobra.Command {
	name := "pin"
	short := "Pin a detail note before unpinned tabs"
	if !pinned {
		name = "unpin"
		short = "Unpin a detail note without closing it"
	}
	var jsonOutput bool
	command := &cobra.Command{
		Use: name + " <id-or-exact-title>", Short: short, Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := a.mutateNotesWorkspace(cmd.Context(), *configPath, agent.Request{
				Type: agent.RequestSetNotePinned, NoteID: args[0], Pinned: pinned,
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				return encodeJSON(a.Out, workspace)
			}
			verb := "pinned"
			if !pinned {
				verb = "unpinned"
			}
			_, err = fmt.Fprintf(a.Out, "%s Notes tab %s\n", verb, args[0])
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes workspace as JSON")
	return command
}

func (a App) notesReorderPinnedCommand(configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "reorder-pinned <id-or-exact-title>...",
		Short: "Replace the complete pinned detail order",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := a.mutateNotesWorkspace(cmd.Context(), *configPath, agent.Request{
				Type: agent.RequestReorderPinned, NoteIDs: args,
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				return encodeJSON(a.Out, workspace)
			}
			_, err = fmt.Fprintln(a.Out, "reordered pinned Notes tabs")
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes workspace as JSON")
	return command
}
