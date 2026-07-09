package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/githubscan"
	"github.com/jamesonstone/beacon/internal/gitscan"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/jamesonstone/beacon/internal/scan"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type App struct {
	Out    io.Writer
	Err    io.Writer
	Runner command.Runner
}

func New() *cobra.Command {
	app := App{Out: os.Stdout, Err: os.Stderr, Runner: command.ExecRunner{}}
	return app.Root()
}

func (a App) Root() *cobra.Command {
	var configPath string
	root := &cobra.Command{
		Use:           "beacon",
		Short:         "Review readiness for agent-driven Git work",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().StringVar(&configPath, "config", "", "configuration file path")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	root.AddCommand(
		a.scanCommand(&configPath),
		a.doctorCommand(&configPath),
		a.openCommand(&configPath),
		a.openNextCommand(&configPath),
		a.configCommand(&configPath),
		versionCommand(a.Out),
	)
	return root
}

func (a App) scanner() scan.Scanner {
	git := gitscan.Scanner{Runner: a.Runner, Now: time.Now}
	github := githubscan.Client{Runner: a.Runner}
	return scan.Scanner{Git: git, GitHub: github, Now: time.Now}
}

func (a App) scanCommand(configPath *string) *cobra.Command {
	var repository string
	var jsonOutput bool
	var noRefresh bool
	command := &cobra.Command{
		Use:   "scan",
		Short: "Scan configured repositories",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(*configPath)
			if err != nil {
				return err
			}
			snapshot, err := a.scanner().Scan(cmd.Context(), cfg, repository, !noRefresh)
			if err != nil {
				return err
			}
			if jsonOutput {
				return output.JSON(a.Out, snapshot)
			}
			return output.Terminal(a.Out, snapshot)
		},
	}
	command.Flags().StringVar(&repository, "repo", "", "scan one configured repository")
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	command.Flags().BoolVar(&noRefresh, "no-refresh", false, "skip git fetch")
	return command
}

func (a App) openCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "open <lane-id>",
		Short: "Open a lane's pull request or worktree",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snapshot, err := a.loadSnapshot(cmd.Context(), *configPath)
			if err != nil {
				return err
			}
			for _, lane := range snapshot.Lanes {
				if lane.ID == args[0] {
					return a.openLane(cmd.Context(), lane)
				}
			}
			return fmt.Errorf("lane not found: %s", args[0])
		},
	}
}

func (a App) openNextCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "open-next",
		Short: "Open the highest-priority lane",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot, err := a.loadSnapshot(cmd.Context(), *configPath)
			if err != nil {
				return err
			}
			if len(snapshot.Lanes) == 0 {
				return errors.New("no work lanes found")
			}
			return a.openLane(cmd.Context(), snapshot.Lanes[0])
		},
	}
}

func (a App) loadSnapshot(ctx context.Context, path string) (model.Snapshot, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return model.Snapshot{}, err
	}
	return a.scanner().Scan(ctx, cfg, "", true)
}

func (a App) openLane(ctx context.Context, lane model.Lane) error {
	target := ""
	if lane.PullRequest != nil {
		target = lane.PullRequest.URL
	} else if lane.Worktree != nil {
		target = lane.Worktree.Path
	}
	if target == "" {
		return fmt.Errorf("lane has no openable target: %s", lane.ID)
	}
	commandContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := a.Runner.Run(commandContext, "", "open", target)
	return err
}

func versionCommand(writer io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(writer, "beacon %s (%s, %s)\n", Version, Commit, Date)
			return err
		},
	}
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
