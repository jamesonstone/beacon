package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/spf13/cobra"
)

type projectPrompter interface {
	SelectTrackedProjects(context.Context, []model.Project) ([]string, error)
	Confirm(context.Context, string) (bool, error)
}

func (a App) projectsCommand(configPath *string) *cobra.Command {
	var showUntracked bool
	var showTracked bool
	command := &cobra.Command{
		Use:   "projects",
		Short: "Manage tracked and untracked projects",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if showTracked && showUntracked {
				return usageError{errors.New("--tracked and --untracked cannot be used together")}
			}
			colorMode, _ := cmd.Flags().GetString("color")
			color, err := a.resolveColor(colorMode)
			if err != nil {
				return err
			}
			if !showUntracked && !showTracked {
				return a.runProjectSelector(cmd.Context(), *configPath, cmd.CommandPath())
			}
			snapshot, err := a.projectSnapshot(cmd.Context(), *configPath)
			if err != nil {
				return err
			}
			if showUntracked {
				return output.Projects(a.Out, snapshot, model.TrackingUntracked, output.TerminalOptions{Color: color, Width: a.terminalWidth()})
			}
			if showTracked {
				return output.Projects(a.Out, snapshot, model.TrackingTracked, output.TerminalOptions{Color: color, Width: a.terminalWidth()})
			}
			return nil
		},
	}
	command.Flags().BoolVar(&showUntracked, "untracked", false, "list untracked projects")
	command.Flags().BoolVar(&showTracked, "tracked", false, "list tracked projects")
	command.AddCommand(
		a.setProjectTrackingCommand(configPath, true),
		a.setProjectTrackingCommand(configPath, false),
	)
	return command
}

func (a App) selectCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "select", Short: "Interactively select projects Beacon should track", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			colorMode, _ := cmd.Flags().GetString("color")
			if _, err := a.resolveColor(colorMode); err != nil {
				return err
			}
			return a.runProjectSelector(cmd.Context(), *configPath, cmd.CommandPath())
		},
	}
}

func (a App) runProjectSelector(ctx context.Context, configPath, commandPath string) error {
	if !a.inputIsTTY() {
		return usageError{fmt.Errorf("%s requires a TTY; use beacon projects --tracked, beacon projects --untracked, track OWNER/REPO, or untrack OWNER/REPO", commandPath)}
	}
	snapshot, err := a.projectSnapshot(ctx, configPath)
	if err != nil {
		return err
	}
	selected, err := a.projectPrompter().SelectTrackedProjects(ctx, snapshot.Projects)
	if err != nil {
		return fmt.Errorf("select tracked projects: %w", err)
	}
	trackedChanges, untrackedChanges := selectionChanges(snapshot.Projects, selected)
	if trackedChanges == 0 && untrackedChanges == 0 {
		_, err := fmt.Fprintln(a.Out, "project tracking unchanged")
		return err
	}
	confirmed, err := a.projectPrompter().Confirm(ctx, fmt.Sprintf("Track %d and untrack %d project(s)?", trackedChanges, untrackedChanges))
	if err != nil {
		return fmt.Errorf("confirm project tracking: %w", err)
	}
	if !confirmed {
		return errors.New("project tracking update cancelled")
	}
	if err := a.setProjectSelection(ctx, configPath, snapshot, selected); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.Out, "updated project tracking: %d tracked, %d untracked\n", len(selected), len(snapshot.Projects)-len(selected))
	return err
}

func (a App) setProjectTrackingCommand(configPath *string, tracked bool) *cobra.Command {
	verb := "track"
	if !tracked {
		verb = "untrack"
	}
	return &cobra.Command{
		Use:   verb + " <project>...",
		Short: verb + " one or more projects",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageError{fmt.Errorf("%s requires at least one project name or owner/repo", cmd.CommandPath())}
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

func (a App) setProjects(ctx context.Context, configPath string, targets []string, tracked bool) error {
	if a.scannerSource == nil {
		if _, paths, err := a.agentConfig(configPath); err == nil {
			state := "muted"
			if tracked {
				state = "tracked"
			}
			event, requestErr := (agent.Client{Socket: paths.Socket}).Request(ctx, agent.Request{
				Type: agent.RequestSetTrackingBatch, ProjectIDs: targets, TrackingState: state,
			})
			if requestErr == nil {
				if event.Type == agent.EventProjectFailed {
					return errors.New(event.Message)
				}
				return nil
			}
		}
	}
	return a.setProjectsDirect(ctx, configPath, targets, tracked)
}

func (a App) setProjectsDirect(ctx context.Context, configPath string, targets []string, tracked bool) error {
	snapshot, err := a.projectSnapshot(ctx, configPath)
	if err != nil {
		return err
	}
	_, err = a.tracker().SetTracked(snapshot, targets, tracked)
	return err
}

func (a App) setProjectSelection(ctx context.Context, configPath string, snapshot model.Snapshot, selected []string) error {
	if a.scannerSource == nil {
		if _, paths, err := a.agentConfig(configPath); err == nil {
			event, requestErr := (agent.Client{Socket: paths.Socket}).Request(ctx, agent.Request{
				Type: agent.RequestSetSelection, ProjectIDs: selected,
			})
			if requestErr == nil {
				if event.Type == agent.EventProjectFailed {
					return errors.New(event.Message)
				}
				return nil
			}
		}
	}
	_, err := a.tracker().SetSelection(snapshot, selected)
	return err
}

func (a App) projectSnapshot(ctx context.Context, configPath string) (model.Snapshot, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return model.Snapshot{}, err
	}
	if a.scannerSource == nil {
		paths, pathErr := agent.ResolvePaths(cfg.Path)
		if pathErr == nil {
			event, requestErr := (agent.Client{Socket: paths.Socket}).Request(ctx, agent.Request{Type: agent.RequestGetSnapshot})
			if requestErr == nil && event.Snapshot != nil && len(event.Snapshot.Projects) > 0 {
				return *event.Snapshot, nil
			}
			records, _ := (agent.Cache{Directory: paths.Projects, Now: time.Now}).LoadAll()
			if len(records) > 0 {
				snapshot := agent.Assemble(records, cfg.Path, paths.State, time.Now().UTC())
				return a.tracker().Reconcile(snapshot)
			}
		}
	}
	return a.scanSnapshot(ctx, cfg, "", false)
}

func (a App) projectPrompter() projectPrompter {
	if a.projectPrompterSource != nil {
		return a.projectPrompterSource
	}
	return huhPrompter{input: a.input(), output: a.Out}
}

func selectionChanges(projects []model.Project, selected []string) (int, int) {
	selectedSet := make(map[string]struct{}, len(selected))
	for _, github := range selected {
		selectedSet[github] = struct{}{}
	}
	var track, untrack int
	for _, project := range projects {
		_, selected := selectedSet[project.GitHub]
		if selected && project.TrackingState == model.TrackingUntracked {
			track++
		}
		if !selected && project.TrackingState != model.TrackingUntracked {
			untrack++
		}
	}
	return track, untrack
}

func trackedProjectIDs(projects []model.Project) []string {
	tracked := make([]string, 0, len(projects))
	for _, project := range projects {
		if project.TrackingState != model.TrackingUntracked {
			tracked = append(tracked, project.GitHub)
		}
	}
	sort.Strings(tracked)
	return tracked
}

func verbPastTense(tracked bool) string {
	if tracked {
		return "tracked"
	}
	return "untracked"
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
