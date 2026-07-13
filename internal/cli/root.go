package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
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
}

type snapshotScanner interface {
	Scan(context.Context, config.Config, string, bool) (model.Snapshot, error)
}

type projectTracker interface {
	Reconcile(model.Snapshot) (model.Snapshot, error)
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
	}
	return app.Root()
}

func (a App) Root() *cobra.Command {
	var configPath string
	var colorMode string
	var includeIdle bool
	var noWatch bool
	root := &cobra.Command{
		Use:           "beacon",
		Short:         "Working-set memory for agent-driven Git work",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.runAgentDashboard(cmd.Context(), configPath, colorMode, includeIdle, noWatch)
		},
	}
	root.PersistentFlags().StringVar(&configPath, "config", "", "configuration file path")
	root.PersistentFlags().StringVar(&colorMode, "color", "auto", "color output: auto, always, or never")
	root.Flags().BoolVar(&includeIdle, "include-idle", false, "show projects with only idle work")
	root.Flags().BoolVar(&noWatch, "no-watch", false, "render cached agent state without requesting a refresh")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	root.AddCommand(
		a.initCommand(&configPath),
		a.scanCommand(&configPath),
		a.projectsCommand(&configPath),
		a.lanesCommand(&configPath),
		a.laneAttentionCommand(&configPath, "pin", agent.RequestSetLanePinned),
		a.laneAttentionCommand(&configPath, "park", agent.RequestSetLaneAttention),
		a.laneAttentionCommand(&configPath, "resume", agent.RequestSetLaneAttention),
		a.laneNoteCommand(&configPath),
		a.laneTagCommand(&configPath, "tag", agent.RequestAddLaneTag),
		a.laneTagCommand(&configPath, "untag", agent.RequestRemoveLaneTag),
		a.laneAddCommand(&configPath),
		a.laneSeenCommand(&configPath),
		a.selectCommand(&configPath),
		a.rootTrackingCommand(&configPath, true),
		a.rootTrackingCommand(&configPath, false),
		a.refreshCommand(&configPath),
		a.agentCommand(&configPath),
		a.doctorCommand(&configPath),
		a.openCommand(&configPath),
		a.openNextCommand(&configPath),
		a.configCommand(&configPath),
		versionCommand(a.Out),
	)
	return root
}

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

func (a App) tracker() projectTracker {
	if a.trackerSource != nil {
		return a.trackerSource
	}
	return tracking.Manager{Store: tracking.FileStore{}, Now: time.Now}
}

func (a App) scanSnapshot(ctx context.Context, cfg config.Config, repository string, refresh bool) (model.Snapshot, error) {
	snapshot, err := a.scanner().Scan(ctx, cfg, repository, refresh)
	if err != nil {
		return model.Snapshot{}, err
	}
	if snapshot.ConfigPath == "" {
		snapshot.ConfigPath = cfg.Path
	}
	return a.tracker().Reconcile(snapshot)
}

func (a App) scanCommand(configPath *string) *cobra.Command {
	var repository string
	var jsonOutput bool
	var noRefresh bool
	var includeIdle bool
	command := &cobra.Command{
		Use:   "scan",
		Short: "Scan configured repositories",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			colorMode, _ := cmd.Flags().GetString("color")
			if _, err := a.resolveColor(colorMode); err != nil {
				return err
			}
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
			return a.runHumanScan(cmd.Context(), *configPath, repository, !noRefresh, colorMode, includeIdle || repository != "", false, false)
		},
	}
	command.Flags().StringVar(&repository, "repo", "", "scan one configured repository")
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	command.Flags().BoolVar(&noRefresh, "no-refresh", false, "skip git fetch")
	command.Flags().BoolVar(&includeIdle, "include-idle", false, "show projects with only idle work")
	return command
}

func (a App) runHumanScan(ctx context.Context, path, repository string, refresh bool, colorMode string, includeIdle, offerInit, showLoader bool) error {
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
	return output.TerminalWithOptions(a.Out, snapshot, output.TerminalOptions{Color: color, Width: a.terminalWidth(), IncludeIdle: includeIdle})
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
