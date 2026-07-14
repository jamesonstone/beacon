package cli

import (
	"time"

	"github.com/jamesonstone/beacon/internal/githubapi"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/spf13/cobra"
)

func (a App) limitsCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "limits",
		Short: "Inspect external dependency rate limits",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report, err := githubapi.InspectRateLimits(cmd.Context(), a.Runner, time.Now)
			if err != nil {
				return err
			}
			if jsonOutput {
				return output.DependencyLimitsJSON(a.Out, report)
			}
			colorMode, _ := cmd.Flags().GetString("color")
			color, err := a.resolveColor(colorMode)
			if err != nil {
				return err
			}
			return output.DependencyLimits(a.Out, report, output.TerminalOptions{Color: color, Width: a.terminalWidth()})
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	return command
}
