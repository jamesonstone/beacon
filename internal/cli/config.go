package cli

import (
	"fmt"
	"os"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/spf13/cobra"
)

func (a App) configCommand(configPath *string) *cobra.Command {
	command := &cobra.Command{Use: "config", Short: "Manage Beacon configuration"}
	command.AddCommand(
		a.initCommandWithUse(configPath, "init"),
		&cobra.Command{
			Use: "path", Short: "Print the resolved configuration path", Args: noArgs,
			RunE: func(_ *cobra.Command, _ []string) error {
				path, err := config.ResolvePath(*configPath)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(a.Out, path)
				return err
			},
		},
		&cobra.Command{
			Use: "validate", Short: "Validate the configuration", Args: noArgs,
			RunE: func(_ *cobra.Command, _ []string) error {
				cfg, err := config.Load(*configPath)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(
					a.Out,
					"valid configuration: %s (%d projects, %d sources, %d repositories)\n",
					cfg.Path,
					len(cfg.Projects),
					len(cfg.Sources),
					len(cfg.Repositories),
				)
				return err
			},
		},
		&cobra.Command{
			Use: "open", Short: "Open the configuration file", Args: noArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				path, err := config.ResolvePath(*configPath)
				if err != nil {
					return err
				}
				if _, err := os.Stat(path); err != nil {
					return fmt.Errorf("open config: %w; run beacon config init first", err)
				}
				return a.openTarget(cmd.Context(), path)
			},
		},
	)
	return command
}
