package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
	"github.com/jamesonstone/beacon/internal/githubapi"
	"github.com/jamesonstone/beacon/internal/githubscan"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/jamesonstone/beacon/internal/scan"
	"github.com/jamesonstone/beacon/internal/tracking"
	"github.com/jamesonstone/beacon/internal/workscan"
	"github.com/spf13/cobra"
)

func (a App) scanner() snapshotScanner {
	if a.scannerSource != nil {
		return a.scannerSource
	}
	return a.scannerComponents()
}

func (a App) scannerComponents() scan.Scanner {
	return a.scannerComponentsWithRunner(githubapi.NewRunner(a.Runner, 5*time.Minute))
}

func (a App) scannerComponentsWithRunner(runner command.Runner) scan.Scanner {
	git := gitscan.Scanner{Runner: runner, Now: time.Now}
	github := githubscan.Client{Runner: runner}
	discoverer := discovery.Discoverer{Runner: runner}
	return scan.Scanner{Git: git, GitHub: github, Discovery: discoverer, Now: time.Now}
}

func (a App) workScanner() workSnapshotScanner {
	if a.workScannerSource != nil {
		return a.workScannerSource
	}
	return workscan.Scanner{
		Git:       gitscan.Scanner{Runner: a.Runner, Now: time.Now},
		GitHub:    githubscan.Client{Runner: a.Runner},
		Discovery: discovery.Discoverer{Runner: a.Runner},
		Now:       time.Now,
	}
}

func (a App) tracker() projectTracker {
	return a.trackerFor(0)
}

func (a App) trackerFor(recentWindow time.Duration) projectTracker {
	if a.trackerSource != nil {
		return a.trackerSource
	}
	return tracking.Manager{
		Store: tracking.FileStore{}, Now: time.Now,
		RecentWindow: recentWindow,
	}
}

func (a App) scanSnapshot(ctx context.Context, cfg config.Config, repository string, refresh bool) (model.Snapshot, error) {
	snapshot, err := a.scanner().Scan(ctx, cfg, repository, refresh)
	if err != nil {
		return model.Snapshot{}, err
	}
	if snapshot.ConfigPath == "" {
		snapshot.ConfigPath = cfg.Path
	}
	tracker := a.trackerFor(cfg.Settings.StaleAfter)
	if repository != "" {
		return tracker.ReconcilePartial(snapshot)
	}
	return tracker.Reconcile(snapshot)
}

func (a App) scanCommand(configPath *string) *cobra.Command {
	var repository string
	var jsonOutput bool
	var noRefresh bool
	var includeIdle bool
	command := &cobra.Command{
		Use:   "scan [path...]",
		Short: "Scan configured repositories or supplied paths",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			colorMode, _ := cmd.Flags().GetString("color")
			if _, err := a.resolveColor(colorMode); err != nil {
				return err
			}
			if len(args) > 0 {
				if repository != "" {
					return usageError{errors.New("--repo cannot be used with path arguments")}
				}
				if *configPath != "" {
					return usageError{errors.New("--config cannot be used with path arguments")}
				}
				return a.runPathScan(cmd.Context(), args, !noRefresh, includeIdle, jsonOutput, colorMode)
			}
			if repository != "" {
				if jsonOutput {
					cfg, err := config.Load(*configPath)
					if err != nil {
						return err
					}
					snapshot, err := a.scanSnapshot(cmd.Context(), cfg, repository, !noRefresh)
					if err != nil {
						return err
					}
					return output.JSON(a.Out, snapshot)
				}
				return a.runHumanScan(cmd.Context(), *configPath, repository, !noRefresh, colorMode, true, false, false, false)
			}
			cfg, err := config.Load(*configPath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("%w; run beacon projects to select projects", err)
				}
				return err
			}
			return a.runWorkScan(cmd.Context(), cfg, !noRefresh, includeIdle, jsonOutput, colorMode)
		},
	}
	command.Flags().StringVar(&repository, "repo", "", "scan one configured repository using the legacy projection")
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	command.Flags().BoolVar(&noRefresh, "no-refresh", false, "skip git fetch")
	command.Flags().BoolVar(&includeIdle, "include-idle", false, "show projects with only idle work")
	return command
}

func (a App) runWorkScan(
	ctx context.Context,
	cfg config.Config,
	refresh bool,
	includeIdle bool,
	jsonOutput bool,
	colorMode string,
) error {
	result, err := a.workScanner().Scan(ctx, cfg, refresh, includeIdle)
	if err != nil {
		return err
	}
	if jsonOutput {
		return output.JSONValue(a.Out, result)
	}
	color, err := a.resolveColor(colorMode)
	if err != nil {
		return err
	}
	return output.WorkTerminal(a.Out, result, output.TerminalOptions{
		Color: color, Width: a.terminalWidth(), IncludeIdle: includeIdle,
	})
}

func (a App) runPathScan(
	ctx context.Context,
	paths []string,
	refresh bool,
	includeIdle bool,
	jsonOutput bool,
	colorMode string,
) error {
	cfg, err := config.ForSources(paths)
	if err != nil {
		return err
	}
	return a.runWorkScan(ctx, cfg, refresh, includeIdle, jsonOutput, colorMode)
}

func (a App) runHumanScan(ctx context.Context, path, repository string, refresh bool, colorMode string, includeIdle, offerInit, showLoader, workingSet bool) error {
	color, err := a.resolveColor(colorMode)
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		if !offerInit || !a.inputIsTTY() {
			return fmt.Errorf("%w; run beacon init to create %s", err, defaultConfigPath(path))
		}
		start, promptErr := a.initPrompter().Confirm(ctx, "Beacon is not configured. Run beacon init now?")
		if promptErr != nil {
			return fmt.Errorf("offer initialization: %w", promptErr)
		}
		if !start {
			return errors.New("Beacon configuration is required; run beacon init")
		}
		if err := a.newInitService(path, initOptions{}).run(ctx); err != nil {
			return err
		}
		cfg, err = config.Load(path)
	}
	if err != nil {
		return err
	}
	loader := startScanLoader(a.Out, showLoader && a.outputIsTTY(), color, a.terminalWidth())
	defer loader.Stop(false)
	snapshot, err := a.scanSnapshot(ctx, cfg, repository, refresh)
	loader.Stop(err == nil)
	if err != nil {
		return err
	}
	return output.TerminalWithOptions(a.Out, snapshot, output.TerminalOptions{
		Color: color, Width: a.terminalWidth(), IncludeIdle: includeIdle, WorkingSet: workingSet,
	})
}
