package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type App struct {
	In                    io.Reader
	Out                   io.Writer
	Err                   io.Writer
	Runner                command.Runner
	InputIsTTY            func() bool
	OutputIsTTY           func() bool
	TerminalWidth         func() int
	scannerSource         snapshotScanner
	trackerSource         projectTracker
	prompter              initPrompter
	projectPrompterSource projectPrompter
	syncPrompterSource    repositorySyncPrompter
	agentClientSource     func(string) agentRequestClient
	agentStarterSource    func(agent.Paths) agentStarter
	autoStartAgent        bool
}

type agentStarter interface {
	Start(context.Context) error
}

type snapshotScanner interface {
	Scan(context.Context, config.Config, string, bool) (model.Snapshot, error)
}

type projectTracker interface {
	Reconcile(model.Snapshot) (model.Snapshot, error)
	ReconcilePartial(model.Snapshot) (model.Snapshot, error)
	SetTracked(model.Snapshot, []string, bool) (model.Snapshot, error)
	SetSelection(model.Snapshot, []string) (model.Snapshot, error)
}

func New() *cobra.Command {
	app := App{
		In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Runner: command.ExecRunner{},
		InputIsTTY:  func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
		OutputIsTTY: func() bool { return term.IsTerminal(int(os.Stdout.Fd())) },
		TerminalWidth: func() int {
			width, _, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				return 120
			}
			return width
		},
		autoStartAgent: true,
	}
	return app.Root()
}

func (a App) Root() *cobra.Command {
	var configPath string
	var colorMode string
	var includeIdle bool
	root := &cobra.Command{
		Use:           "beacon",
		Short:         "Working-set memory for agent-driven Git work",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.runAgentDashboard(cmd.Context(), configPath, colorMode, includeIdle)
		},
	}
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if a.autoStartAgent && runtime.GOOS == "darwin" && shouldAutoStartAgent(cmd) {
			a.startAgentBestEffort(cmd.Context(), configPath)
		}
		return nil
	}
	root.PersistentFlags().StringVar(&configPath, "config", "", "configuration file path")
	root.PersistentFlags().StringVar(&colorMode, "color", "auto", "color output: auto, always, or never")
	root.Flags().BoolVar(&includeIdle, "include-idle", false, "show projects with only idle work")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	root.AddCommand(
		a.initCommand(&configPath),
		a.scanCommand(&configPath),
		a.projectsCommand(&configPath),
		a.lanesCommand(&configPath),
		a.laneAttentionCommand(&configPath, "pin", agent.RequestSetLanePinned),
		a.laneReorderCommand(&configPath),
		a.laneAttentionCommand(&configPath, "park", agent.RequestSetLaneAttention),
		a.laneAttentionCommand(&configPath, "resume", agent.RequestSetLaneAttention),
		a.laneNoteCommand(&configPath),
		a.notesCommand(&configPath),
		a.laneTagCommand(&configPath, "tag", agent.RequestAddLaneTag),
		a.laneTagCommand(&configPath, "untag", agent.RequestRemoveLaneTag),
		a.laneAddCommand(&configPath),
		a.laneSeenCommand(&configPath),
		a.selectCommand(&configPath),
		a.rootFollowingCommand(&configPath, true),
		a.rootFollowingCommand(&configPath, false),
		a.rootTrackingCommand(&configPath, true),
		a.rootTrackingCommand(&configPath, false),
		a.refreshCommand(&configPath),
		a.syncCommand(&configPath),
		a.limitsCommand(),
		a.integrationsCommand(&configPath),
		a.activityCommand(&configPath),
		a.agentCommand(&configPath),
		a.doctorCommand(&configPath),
		a.openCommand(&configPath),
		a.openNextCommand(&configPath),
		a.configCommand(&configPath),
		versionCommand(a.Out),
	)
	return root
}

func shouldAutoStartAgent(command *cobra.Command) bool {
	if command.Name() == "init" {
		return false
	}
	top := command
	for top.Parent() != nil && top.Parent().Parent() != nil {
		top = top.Parent()
	}
	switch top.Name() {
	case "activity", "agent", "doctor", "init", "integrations", "version":
		return false
	default:
		return true
	}
}

func (a App) startAgentBestEffort(ctx context.Context, configPath string) {
	_, paths, err := a.agentConfig(configPath)
	if err != nil {
		return
	}
	starter := agentStarter(agent.Lifecycle{Paths: paths, Runner: a.Runner})
	if a.agentStarterSource != nil {
		starter = a.agentStarterSource(paths)
	}
	_ = starter.Start(ctx)
}

func (a App) resolveColor(mode string) (bool, error) {
	switch mode {
	case "", "auto":
		return a.outputIsTTY() && os.Getenv("NO_COLOR") == "", nil
	case "always":
		return true, nil
	case "never":
		return false, nil
	default:
		return false, usageError{fmt.Errorf("--color must be auto, always, or never: %q", mode)}
	}
}

func (a App) input() io.Reader {
	if a.In != nil {
		return a.In
	}
	return os.Stdin
}

func (a App) initPrompter() initPrompter {
	if a.prompter != nil {
		return a.prompter
	}
	return huhPrompter{input: a.input(), output: a.Out}
}

func (a App) inputIsTTY() bool {
	return a.InputIsTTY != nil && a.InputIsTTY()
}

func (a App) outputIsTTY() bool {
	return a.OutputIsTTY != nil && a.OutputIsTTY()
}

func (a App) terminalWidth() int {
	if a.TerminalWidth != nil {
		return a.TerminalWidth()
	}
	return 120
}

func defaultConfigPath(explicit string) string {
	path, err := config.ResolvePath(explicit)
	if err != nil {
		return "$HOME/.config/beacon/config.yaml"
	}
	return path
}

type usageError struct{ error }

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var usage usageError
	if errors.As(err, &usage) || strings.Contains(err.Error(), "unknown command") {
		return 2
	}
	return 1
}

func noArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return usageError{fmt.Errorf("%s accepts no arguments", cmd.CommandPath())}
	}
	return nil
}

func exactArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != count {
			return usageError{fmt.Errorf("%s requires exactly %d argument(s)", cmd.CommandPath(), count)}
		}
		return nil
	}
}
