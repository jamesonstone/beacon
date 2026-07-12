package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/githubapi"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/jamesonstone/beacon/internal/tracking"
	"github.com/jamesonstone/beacon/internal/workset"
	"github.com/spf13/cobra"
)

func (a App) agentCommand(configPath *string) *cobra.Command {
	command := &cobra.Command{Use: "agent", Short: "Manage the Beacon background agent", Args: noArgs}
	command.AddCommand(
		a.agentServeCommand(configPath),
		a.agentInstallCommand(configPath),
		a.agentStatusCommand(configPath),
		a.agentStopCommand(configPath),
		a.agentUninstallCommand(configPath),
	)
	return command
}

func (a App) agentServeCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "serve", Short: "Run the Beacon agent in the foreground", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			engine, paths, err := a.newAgentEngine(cmd.Context(), *configPath)
			if err != nil {
				return err
			}
			if err := agent.RotateLogs(paths, 5<<20); err != nil {
				return err
			}
			serveContext, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return (&agent.Server{Paths: paths, Engine: engine}).Serve(serveContext)
		},
	}
}

func (a App) agentInstallCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "install", Short: "Install and start the user LaunchAgent", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, paths, err := a.agentConfig(*configPath)
			if err != nil {
				return err
			}
			if err := (agent.Lifecycle{Paths: paths, Runner: a.Runner}).Install(cmd.Context()); err != nil {
				return err
			}
			_, err = fmt.Fprintf(a.Out, "installed Beacon agent at %s\n", paths.LaunchAgent)
			return err
		},
	}
}

func (a App) agentStatusCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "status", Short: "Show background agent status", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, paths, err := a.agentConfig(*configPath)
			if err != nil {
				return err
			}
			status, err := (agent.Lifecycle{Paths: paths, Runner: a.Runner}).Status(cmd.Context())
			if err != nil {
				return fmt.Errorf("Beacon agent is not running: %w", err)
			}
			_, err = fmt.Fprintf(a.Out, "running pid=%d projects=%d refreshing=%t socket=%s\n", status.PID, status.ProjectCount, status.Refreshing, status.Socket)
			return err
		},
	}
}

func (a App) agentStopCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "stop", Short: "Stop the Beacon background agent", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, paths, err := a.agentConfig(*configPath)
			if err != nil {
				return err
			}
			if err := (agent.Lifecycle{Paths: paths, Runner: a.Runner}).Stop(cmd.Context()); err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.Out, "stopped Beacon agent")
			return err
		},
	}
}

func (a App) agentUninstallCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "uninstall", Short: "Remove the Beacon user LaunchAgent", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, paths, err := a.agentConfig(*configPath)
			if err != nil {
				return err
			}
			if err := (agent.Lifecycle{Paths: paths, Runner: a.Runner}).Uninstall(cmd.Context()); err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.Out, "uninstalled Beacon agent")
			return err
		},
	}
}

func (a App) refreshCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "refresh [project]", Short: "Request a background refresh", Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return usageError{fmt.Errorf("%s accepts at most one project", cmd.CommandPath())}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, paths, err := a.agentConfig(*configPath)
			if err != nil {
				return err
			}
			request := agent.Request{Type: agent.RequestRefreshAll}
			if len(args) == 1 {
				request.Type, request.ProjectID = agent.RequestRefreshProject, args[0]
				if snapshotEvent, snapshotErr := (agent.Client{Socket: paths.Socket}).Request(cmd.Context(), agent.Request{Type: agent.RequestGetSnapshot}); snapshotErr == nil && snapshotEvent.Snapshot != nil {
					for _, lane := range snapshotEvent.Snapshot.Lanes {
						if lane.ID != args[0] {
							continue
						}
						if lane.GitHub == "" {
							return fmt.Errorf("manual lane cannot be refreshed: %s", lane.ID)
						}
						request.ProjectID = lane.GitHub
						break
					}
				}
			}
			event, err := (agent.Client{Socket: paths.Socket}).Request(cmd.Context(), request)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(a.Out, "queued refresh %s\n", event.ScanID)
			return err
		},
	}
}

func (a App) rootTrackingCommand(configPath *string, tracked bool) *cobra.Command {
	verb := "untrack"
	label := "Untracked"
	if tracked {
		verb = "track"
		label = "Tracked"
	}
	return &cobra.Command{
		Use: verb + " <project>...", Short: "Move projects to " + label,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageError{fmt.Errorf("%s requires at least one project", cmd.CommandPath())}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.setProjects(cmd.Context(), *configPath, args, tracked); err != nil {
				return err
			}
			_, err := fmt.Fprintf(a.Out, "%s %d project%s\n", verbPastTense(tracked), len(args), pluralSuffix(len(args)))
			return err
		},
	}
}

func (a App) runAgentDashboard(ctx context.Context, configPath, colorMode string, includeIdle, noWatch bool) error {
	color, err := a.resolveColor(colorMode)
	if err != nil {
		return err
	}
	_, paths, err := a.agentConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return a.runHumanScan(ctx, configPath, "", true, colorMode, includeIdle, true, true)
		}
		return err
	}
	// Injected scanners are used by command tests and do not have an agent
	// process. Production commands must not hide an unavailable agent behind a
	// slow foreground scan.
	if a.scannerSource != nil {
		return a.runHumanScan(ctx, configPath, "", true, colorMode, includeIdle, true, true)
	}
	client := agent.Client{Socket: paths.Socket}
	event, err := client.Request(ctx, agent.Request{Type: agent.RequestGetSnapshot})
	if err != nil {
		return fmt.Errorf("Beacon background agent is unavailable: %w; run beacon agent install, or use beacon scan for a blocking scan", err)
	}
	if event.Snapshot == nil {
		return errors.New("agent returned no cached snapshot")
	}
	if err := output.TerminalWithOptions(a.Out, *event.Snapshot, output.TerminalOptions{Color: color, Width: a.terminalWidth(), IncludeIdle: includeIdle, WorkingSet: true}); err != nil {
		return err
	}
	_ = noWatch
	return nil
}

func (a App) agentConfig(path string) (config.Config, agent.Paths, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, agent.Paths{}, err
	}
	paths, err := agent.ResolvePaths(cfg.Path)
	return cfg, paths, err
}

func (a App) newAgentEngine(ctx context.Context, path string) (*agent.Engine, agent.Paths, error) {
	cfg, paths, err := a.agentConfig(path)
	if err != nil {
		return nil, agent.Paths{}, err
	}
	if err := paths.EnsureRuntime(); err != nil {
		return nil, agent.Paths{}, err
	}
	githubRunner := githubapi.NewRunnerWithOptions(a.Runner, githubapi.Options{
		CacheTTL: cfg.Settings.RemoteRefreshInterval, CacheDirectory: filepath.Join(paths.CacheRoot, "github"),
	})
	scanner := a.scannerComponentsWithRunner(githubRunner)
	tracker := tracking.Manager{Store: tracking.FileStore{}, Now: time.Now}
	repositories := func(repositoryContext context.Context) ([]config.Repository, error) {
		values, scanErrors, _ := scanner.Repositories(repositoryContext, cfg)
		if len(values) == 0 {
			if len(scanErrors) > 0 {
				return nil, fmt.Errorf("discover repositories: %s", scanErrors[0].Message)
			}
			return nil, errors.New("configuration resolved no repositories")
		}
		return values, nil
	}
	projectScanner := func(scanContext context.Context, repository config.Repository, refresh bool, stage func(string)) (model.Snapshot, error) {
		snapshot, scanErr := scanner.ScanOne(scanContext, cfg, repository, refresh, stage)
		if scanErr != nil {
			return model.Snapshot{}, scanErr
		}
		return tracker.Reconcile(snapshot)
	}
	cache := agent.Cache{Directory: paths.Projects, Now: time.Now}
	prober := agent.Prober{Runner: githubRunner, Remote: scanner.GitHub}
	engine := agent.NewEngine(cfg, paths, cache, repositories, projectScanner, prober, tracker)
	workingSet := workset.Manager{Store: workset.FileStore{}, Now: time.Now}
	engine.WorkingSet = &workingSet
	engine.ScanBatch = func(
		scanContext context.Context,
		repositories []config.Repository,
		refresh bool,
		stage func(string, string),
	) (map[string]model.Snapshot, error) {
		snapshots, scanErr := scanner.ScanMany(scanContext, cfg, repositories, refresh, stage)
		if scanErr != nil {
			return nil, scanErr
		}
		for projectID, snapshot := range snapshots {
			reconciled, reconcileErr := tracker.Reconcile(snapshot)
			if reconcileErr != nil {
				return nil, reconcileErr
			}
			snapshots[projectID] = reconciled
		}
		return snapshots, nil
	}
	engine.ProbeBatch = prober.ProbeMany
	_ = ctx
	return engine, paths, nil
}
