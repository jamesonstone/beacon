package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jamesonstone/beacon/internal/activity"
	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/integrations"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/spf13/cobra"
)

func (a App) activityCommand(configPath *string) *cobra.Command {
	command := &cobra.Command{Use: "activity", Hidden: true, Args: noArgs}
	command.AddCommand(
		a.activityIngestCommand(configPath),
		a.activityReadCommand(configPath, "list"),
		a.activityReadCommand(configPath, "prune"),
	)
	return command
}

func (a App) activityIngestCommand(configPath *string) *cobra.Command {
	var provider string
	var hook bool
	command := &cobra.Command{
		Use: "ingest", Hidden: true, Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !hook {
				return usageError{fmt.Errorf("%s requires --hook", cmd.CommandPath())}
			}
			// Provider hooks are a fail-open boundary: no malformed callback or
			// unavailable Beacon component may affect the provider process.
			_ = a.ingestHook(cmd.Context(), *configPath, provider)
			return nil
		},
	}
	command.Flags().BoolVar(&hook, "hook", false, "accept a provider hook callback")
	command.Flags().StringVar(&provider, "provider", "", "hook provider")
	return command
}

func (a App) ingestHook(ctx context.Context, configPath, provider string) error {
	resolved, err := config.ResolvePath(configPath)
	if err != nil {
		return err
	}
	paths, err := agent.ResolvePaths(resolved)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	manager := integrations.Manager{Executable: executable, Health: integrations.HealthStore{Path: paths.IntegrationHealth}}
	service := activity.Service{
		Store:    activity.Store{Path: paths.Activity, LockPath: paths.ActivityLock},
		Agent:    agent.Client{Socket: paths.Socket, Timeout: 150 * time.Millisecond},
		Observe:  manager.Observe,
		Deadline: 450 * time.Millisecond,
	}
	bounded, cancel := context.WithTimeout(ctx, 475*time.Millisecond)
	defer cancel()
	completed := make(chan error, 1)
	go func() { completed <- service.Ingest(bounded, provider, a.input()) }()
	select {
	case err := <-completed:
		return err
	case <-bounded.Done():
		return bounded.Err()
	}
}

func (a App) activityReadCommand(configPath *string, action string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: action, Hidden: true, Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolved, err := config.ResolvePath(*configPath)
			if err != nil {
				return err
			}
			paths, err := agent.ResolvePaths(resolved)
			if err != nil {
				return err
			}
			store := activity.Store{Path: paths.Activity, LockPath: paths.ActivityLock}
			var snapshot activity.Snapshot
			if action == "prune" {
				snapshot, err = store.Prune(cmd.Context(), time.Now())
			} else {
				snapshot, err = store.List(cmd.Context(), time.Now())
			}
			if err != nil {
				return err
			}
			if jsonOutput {
				return output.JSONValue(a.Out, snapshot)
			}
			_, err = fmt.Fprintf(a.Out, "%d current activity record(s)\n", len(snapshot.Records))
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	return command
}
