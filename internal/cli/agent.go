package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/spf13/cobra"
)

type agentRequestClient interface {
	Request(context.Context, agent.Request) (agent.Event, error)
}

func (a App) agentClient(socket string) agentRequestClient {
	if a.agentClientSource != nil {
		return a.agentClientSource(socket)
	}
	return agent.Client{Socket: socket}
}

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
	return a.rootProjectMutationCommand(configPath, verb, label, tracked, true)
}

func (a App) rootFollowingCommand(configPath *string, followed bool) *cobra.Command {
	verb := "unfollow"
	label := "outside Following"
	if followed {
		verb = "follow"
		label = "Following"
	}
	return a.rootProjectMutationCommand(configPath, verb, label, followed, false)
}

func (a App) rootProjectMutationCommand(configPath *string, verb, label string, followed, compatibility bool) *cobra.Command {
	short := "Move projects to " + label
	if compatibility {
		short += " (compatibility alias)"
	}
	return &cobra.Command{
		Use: verb + " <project>...", Short: short,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageError{fmt.Errorf("%s requires at least one project", cmd.CommandPath())}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.setProjects(cmd.Context(), *configPath, args, followed); err != nil {
				return err
			}
			past := "stopped following"
			if followed {
				past = "followed"
			}
			if compatibility {
				past = verbPastTense(followed)
			}
			_, err := fmt.Fprintf(a.Out, "%s %d project%s\n", past, len(args), pluralSuffix(len(args)))
			return err
		},
	}
}

func (a App) runAgentDashboard(ctx context.Context, configPath, colorMode string, includeIdle bool) error {
	color, err := a.resolveColor(colorMode)
	if err != nil {
		return err
	}
	_, paths, err := a.agentConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return a.runHumanScan(ctx, configPath, "", true, colorMode, includeIdle, true, true, !includeIdle)
		}
		return err
	}
	// Scanner injection is a test boundary. Tests that exercise the agent path
	// provide both an injected scanner and an injected agent client.
	if a.scannerSource != nil && a.agentClientSource == nil {
		return a.runHumanScan(ctx, configPath, "", true, colorMode, includeIdle, true, true, !includeIdle)
	}
	client := a.agentClient(paths.Socket)
	loader := startScanLoader(a.Out, a.outputIsTTY(), color, a.terminalWidth())
	event, err := client.Request(ctx, agent.Request{Type: agent.RequestRefreshAll})
	if err != nil {
		loader.Stop(false)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return a.runHumanScan(ctx, configPath, "", true, colorMode, includeIdle, true, true, !includeIdle)
	}
	if event.ScanID == "" {
		loader.Stop(false)
		return errors.New("agent returned no scan id for manual refresh")
	}
	if err := waitForAgentRefresh(ctx, client); err != nil {
		loader.Stop(false)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return a.runHumanScan(ctx, configPath, "", true, colorMode, includeIdle, true, true, !includeIdle)
	}
	event, err = client.Request(ctx, agent.Request{Type: agent.RequestGetSnapshot})
	loader.Stop(err == nil && event.Snapshot != nil)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return a.runHumanScan(ctx, configPath, "", true, colorMode, includeIdle, true, true, !includeIdle)
	}
	if event.Snapshot == nil {
		return a.runHumanScan(ctx, configPath, "", true, colorMode, includeIdle, true, true, !includeIdle)
	}
	return output.TerminalWithOptions(a.Out, *event.Snapshot, output.TerminalOptions{
		Color: color, Width: a.terminalWidth(), IncludeIdle: includeIdle, WorkingSet: !includeIdle,
	})
}

func waitForAgentRefresh(ctx context.Context, client agentRequestClient) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		event, err := client.Request(ctx, agent.Request{Type: agent.RequestGetAgentStatus})
		if err != nil {
			return err
		}
		if event.Status == nil {
			return errors.New("agent returned no status during manual refresh")
		}
		if !event.Status.Refreshing {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
