package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/spf13/cobra"
)

func (a App) configCommand(configPath *string) *cobra.Command {
	command := &cobra.Command{Use: "config", Short: "Manage Beacon configuration"}
	command.AddCommand(
		&cobra.Command{
			Use: "init", Short: "Create an example configuration", Args: noArgs,
			RunE: func(_ *cobra.Command, _ []string) error { return initConfig(*configPath, a.Out) },
		},
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
				_, err = fmt.Fprintf(a.Out, "valid configuration: %s (%d repositories)\n", cfg.Path, len(cfg.Repositories))
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
				commandContext, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
				defer cancel()
				_, err = a.Runner.Run(commandContext, "", "open", path)
				return err
			},
		},
	)
	return command
}

func initConfig(explicit string, writer interface{ Write([]byte) (int, error) }) error {
	path, err := config.ResolvePath(explicit)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create config %s: %w", path, err)
	}
	if _, err := file.WriteString(config.Example()); err != nil {
		file.Close()
		return fmt.Errorf("write config %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close config %s: %w", path, err)
	}
	_, err = fmt.Fprintf(writer, "created %s\n", path)
	return err
}
