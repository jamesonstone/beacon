package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/jamesonstone/beacon/internal/reposync"
	"github.com/spf13/cobra"
)

type repositorySyncPrompter interface {
	SelectRepositoryUpdates(context.Context, []reposync.Repository) ([]string, error)
	Confirm(context.Context, string) (bool, error)
}

func (a App) syncCommand(configPath *string) *cobra.Command {
	command := &cobra.Command{
		Use: "sync", Short: "Check and safely update local default branches", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.runInteractiveRepositorySync(cmd.Context(), *configPath)
		},
	}
	command.AddCommand(a.syncCheckCommand(configPath), a.syncApplyCommand(configPath))
	return command
}

func (a App) syncCheckCommand(configPath *string) *cobra.Command {
	var noFetch bool
	var jsonOutput bool
	command := &cobra.Command{
		Use: "check [project...]", Short: "Check repositories for default-branch updates", Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, targets []string) error {
			repositories, service, err := a.repositorySyncInputs(cmd.Context(), *configPath, targets)
			if err != nil {
				return err
			}
			report := service.Check(cmd.Context(), repositories, !noFetch)
			return a.writeRepositorySync(cmd, report, jsonOutput)
		},
	}
	command.Flags().BoolVar(&noFetch, "no-fetch", false, "inspect existing local refs without network access")
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	return command
}

func (a App) syncApplyCommand(configPath *string) *cobra.Command {
	var yes bool
	var jsonOutput bool
	command := &cobra.Command{
		Use: "apply <project>...", Short: "Apply fast-forward-safe repository updates", Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageError{fmt.Errorf("%s requires at least one project", cmd.CommandPath())}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, targets []string) error {
			repositories, service, err := a.repositorySyncInputs(cmd.Context(), *configPath, targets)
			if err != nil {
				return err
			}
			if !yes {
				if !a.inputIsTTY() || jsonOutput {
					return usageError{errors.New("beacon sync apply requires --yes outside an interactive terminal")}
				}
				confirmed, confirmErr := a.repositorySyncPrompter().Confirm(cmd.Context(), fmt.Sprintf("Apply fast-forward-only updates to %d repository(s)?", len(repositories)))
				if confirmErr != nil {
					return fmt.Errorf("confirm repository sync: %w", confirmErr)
				}
				if !confirmed {
					return errors.New("repository sync cancelled")
				}
			}
			report := service.Apply(cmd.Context(), repositories, repositoryIDs(repositories))
			if err := a.writeRepositorySync(cmd, report, jsonOutput); err != nil {
				return err
			}
			return repositorySyncResultError(report)
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "confirm the requested fast-forward-only updates")
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	return command
}

func (a App) runInteractiveRepositorySync(ctx context.Context, configPath string) error {
	if !a.inputIsTTY() {
		return usageError{errors.New("beacon sync requires a TTY; use beacon sync check or beacon sync apply <project> --yes")}
	}
	repositories, service, err := a.repositorySyncInputs(ctx, configPath, nil)
	if err != nil {
		return err
	}
	report := service.Check(ctx, repositories, true)
	if err := output.RepositorySync(a.Out, report, output.TerminalOptions{Color: true, Width: a.terminalWidth()}); err != nil {
		return err
	}
	candidates := safeRepositoryUpdates(report)
	if len(candidates) == 0 {
		_, err := fmt.Fprintln(a.Out, "\nNo repositories can be updated automatically.")
		return err
	}
	selected, err := a.repositorySyncPrompter().SelectRepositoryUpdates(ctx, candidates)
	if err != nil {
		return fmt.Errorf("select repository updates: %w", err)
	}
	confirmed, err := a.repositorySyncPrompter().Confirm(ctx, fmt.Sprintf("Run the shown Git-only updates for %d repository(s)?", len(selected)))
	if err != nil {
		return fmt.Errorf("confirm repository sync: %w", err)
	}
	if !confirmed {
		return errors.New("repository sync cancelled")
	}
	applied := service.Apply(ctx, repositories, selected)
	if _, err := fmt.Fprintln(a.Out); err != nil {
		return err
	}
	if err := output.RepositorySync(a.Out, applied, output.TerminalOptions{Color: true, Width: a.terminalWidth()}); err != nil {
		return err
	}
	return repositorySyncResultError(applied)
}

func (a App) repositorySyncInputs(ctx context.Context, configPath string, targets []string) ([]config.Repository, reposync.Service, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, reposync.Service{}, err
	}
	repositories, scanErrors, _ := a.scannerComponents().Repositories(ctx, cfg)
	if len(scanErrors) > 0 {
		return nil, reposync.Service{}, fmt.Errorf("discover repositories: %s", scanErrors[0].Message)
	}
	if len(repositories) == 0 {
		return nil, reposync.Service{}, errors.New("configuration resolved no repositories")
	}
	if len(targets) > 0 {
		repositories, err = resolveRepositoryTargets(repositories, targets)
		if err != nil {
			return nil, reposync.Service{}, err
		}
	}
	return repositories, reposync.Service{Runner: a.Runner, MaxParallel: cfg.Settings.MaxParallel, Now: time.Now}, nil
}

func (a App) writeRepositorySync(cmd *cobra.Command, report reposync.Report, jsonOutput bool) error {
	if jsonOutput {
		return output.RepositorySyncJSON(a.Out, report)
	}
	colorMode, _ := cmd.Flags().GetString("color")
	color, err := a.resolveColor(colorMode)
	if err != nil {
		return err
	}
	return output.RepositorySync(a.Out, report, output.TerminalOptions{Color: color, Width: a.terminalWidth()})
}

func (a App) repositorySyncPrompter() repositorySyncPrompter {
	if a.syncPrompterSource != nil {
		return a.syncPrompterSource
	}
	return huhPrompter{input: a.input(), output: a.Out}
}

func resolveRepositoryTargets(repositories []config.Repository, targets []string) ([]config.Repository, error) {
	selected := make(map[string]config.Repository, len(targets))
	for _, target := range targets {
		var matches []config.Repository
		for _, repository := range repositories {
			if repository.GitHub == target || repository.Name == target {
				matches = append(matches, repository)
			}
		}
		if len(matches) == 0 {
			return nil, usageError{fmt.Errorf("unknown repository %q", target)}
		}
		if len(matches) > 1 {
			return nil, usageError{fmt.Errorf("repository name %q is ambiguous; use owner/repository", target)}
		}
		selected[matches[0].GitHub] = matches[0]
	}
	result := make([]config.Repository, 0, len(selected))
	for _, repository := range selected {
		result = append(result, repository)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].GitHub < result[j].GitHub })
	return result, nil
}

func safeRepositoryUpdates(report reposync.Report) []reposync.Repository {
	result := make([]reposync.Repository, 0)
	for _, repository := range report.Repositories {
		if repository.CanUpdate {
			result = append(result, repository)
		}
	}
	return result
}

func repositoryIDs(repositories []config.Repository) []string {
	result := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		result = append(result, repository.GitHub)
	}
	return result
}

func repositorySyncResultError(report reposync.Report) error {
	var failures []string
	for _, repository := range report.Repositories {
		if repository.Error != "" || (repository.NeedsUpdate && !repository.Updated) {
			failures = append(failures, repository.ProjectID)
		}
	}
	if len(failures) == 0 {
		return nil
	}
	return fmt.Errorf("repository sync needs manual attention: %s", strings.Join(failures, ", "))
}
