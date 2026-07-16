package cli

import (
	"fmt"
	"os"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/integrations"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/spf13/cobra"
)

func (a App) integrationsCommand(configPath *string) *cobra.Command {
	command := &cobra.Command{Use: "integrations", Short: "Manage structured provider hooks", Args: noArgs}
	command.AddCommand(
		a.integrationMutationCommand(configPath, "install"),
		a.integrationStatusCommand(configPath),
		a.integrationMutationCommand(configPath, "uninstall"),
	)
	return command
}

func (a App) integrationStatusCommand(configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "status <codex|claude-code>", Short: "Inspect a structured hook integration", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			manager, err := a.integrationManager(*configPath)
			if err != nil {
				return err
			}
			status := manager.Status(args[0])
			if jsonOutput {
				return output.JSONValue(a.Out, status)
			}
			if status.Message == "" {
				_, err = fmt.Fprintf(a.Out, "%s: %s (%s)\n", status.Provider, status.State, status.SettingsPath)
			} else {
				_, err = fmt.Fprintf(a.Out, "%s: %s (%s): %s\n", status.Provider, status.State, status.SettingsPath, status.Message)
			}
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	return command
}

func (a App) integrationMutationCommand(configPath *string, action string) *cobra.Command {
	var yes bool
	title := "Install"
	if action == "uninstall" {
		title = "Uninstall"
	}
	command := &cobra.Command{
		Use: action + " <codex|claude-code>", Short: title + " a structured hook integration", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := a.integrationManager(*configPath)
			if err != nil {
				return err
			}
			var plan integrations.Plan
			if action == "install" {
				plan, err = manager.PlanInstall(args[0])
			} else {
				plan, err = manager.PlanUninstall(args[0])
			}
			if err != nil {
				return err
			}
			if err := a.printIntegrationPlan(plan); err != nil {
				return err
			}
			if !plan.Changed {
				if action == "uninstall" {
					if err := manager.Apply(plan); err != nil {
						return err
					}
				}
				_, err = fmt.Fprintln(a.Out, "no changes required")
				return err
			}
			if !yes {
				confirmed, confirmErr := a.initPrompter().Confirm(cmd.Context(), "Apply the exact hook changes shown above?")
				if confirmErr != nil {
					return confirmErr
				}
				if !confirmed {
					_, err = fmt.Fprintln(a.Out, "cancelled")
					return err
				}
			}
			if err := manager.Apply(plan); err != nil {
				return err
			}
			_, err = fmt.Fprintf(a.Out, "%sed %s integration\n", action, args[0])
			return err
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "apply the previewed changes without an interactive prompt")
	return command
}

func (a App) integrationManager(configPath string) (integrations.Manager, error) {
	resolved, err := config.ResolvePath(configPath)
	if err != nil {
		return integrations.Manager{}, err
	}
	paths, err := agent.ResolvePaths(resolved)
	if err != nil {
		return integrations.Manager{}, err
	}
	executable, err := os.Executable()
	if err != nil {
		return integrations.Manager{}, fmt.Errorf("resolve Beacon executable: %w", err)
	}
	return integrations.Manager{Executable: executable, Health: integrations.HealthStore{Path: paths.IntegrationHealth}}, nil
}

func (a App) printIntegrationPlan(plan integrations.Plan) error {
	if _, err := fmt.Fprintf(a.Out, "Preview %s %s hooks in %s\n", plan.Action, plan.Provider, plan.SettingsPath); err != nil {
		return err
	}
	for _, change := range plan.Changes {
		matcher := ""
		if change.Matcher != "" {
			matcher = " matcher=" + change.Matcher
		}
		if _, err := fmt.Fprintf(a.Out, "  %s %s%s: %s\n", change.Operation, change.Event, matcher, change.Command); err != nil {
			return err
		}
	}
	if plan.BackupPath != "" {
		_, err := fmt.Fprintf(a.Out, "  backup 0600: %s\n", plan.BackupPath)
		return err
	}
	return nil
}
