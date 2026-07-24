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
	SelectFollowedProjects(context.Context, []model.Project) ([]string, error)
	Confirm(context.Context, string) (bool, error)
}

func (a App) projectsCommand(configPath *string) *cobra.Command {
	var showUntracked bool
	var showTracked bool
	var showFollowed bool
	var showRecent bool
	var showQuiet bool
	var browserRoot string
	command := &cobra.Command{
		Use:   "projects",
		Short: "Select projects Beacon scans",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			views := 0
			for _, selected := range []bool{showTracked || showFollowed, showUntracked, showRecent, showQuiet} {
				if selected {
					views++
				}
			}
			if views > 1 {
				return usageError{errors.New("select only one project inventory view")}
			}
			colorMode, _ := cmd.Flags().GetString("color")
			color, err := a.resolveColor(colorMode)
			if err != nil {
				return err
			}
			if views == 0 {
				return a.runConfiguredProjectSelector(cmd.Context(), *configPath, browserRoot, cmd.CommandPath())
			}
			snapshot, err := a.projectSnapshot(cmd.Context(), *configPath)
			if err != nil {
				return err
			}
			if showUntracked {
				return output.Projects(a.Out, snapshot, model.TrackingUntracked, output.TerminalOptions{Color: color, Width: a.terminalWidth()})
			}
			followState := model.FollowFollowing
			if showRecent {
				followState = model.FollowRecent
			} else if showQuiet {
				followState = model.FollowQuiet
			}
			return output.FollowingProjects(a.Out, snapshot, followState, output.TerminalOptions{Color: color, Width: a.terminalWidth()})
		},
	}
	command.Flags().BoolVar(&showFollowed, "followed", false, "list followed projects")
	command.Flags().BoolVar(&showRecent, "recent", false, "list recently updated projects outside Following")
	command.Flags().BoolVar(&showQuiet, "quiet", false, "list quiet projects outside Following")
	command.Flags().BoolVar(&showUntracked, "untracked", false, "compatibility alias: list all projects outside Following")
	command.Flags().BoolVar(&showTracked, "tracked", false, "compatibility alias: list followed projects")
	command.Flags().StringVar(&browserRoot, "root", defaultProjectBrowserRoot, "project browser root")
	command.AddCommand(
		a.setProjectFollowingCommand(configPath, true),
		a.setProjectFollowingCommand(configPath, false),
		a.setProjectTrackingCommand(configPath, true),
		a.setProjectTrackingCommand(configPath, false),
	)
	return command
}

func (a App) selectCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "select", Short: "Interactively select projects Beacon should follow", Args: noArgs,
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
		return usageError{fmt.Errorf("%s requires a TTY; use beacon projects --followed, --recent, --quiet, follow OWNER/REPO, or unfollow OWNER/REPO", commandPath)}
	}
	snapshot, err := a.projectSnapshot(ctx, configPath)
	if err != nil {
		return err
	}
	selected, err := a.projectPrompter().SelectFollowedProjects(ctx, snapshot.Projects)
	if err != nil {
		return fmt.Errorf("select followed projects: %w", err)
	}
	followedChanges, outsideChanges := selectionChanges(snapshot.Projects, selected)
	if followedChanges == 0 && outsideChanges == 0 {
		_, err := fmt.Fprintln(a.Out, "project following unchanged")
		return err
	}
	confirmed, err := a.projectPrompter().Confirm(ctx, fmt.Sprintf("Follow %d and stop following %d project(s)?", followedChanges, outsideChanges))
	if err != nil {
		return fmt.Errorf("confirm project following: %w", err)
	}
	if !confirmed {
		return errors.New("project following update cancelled")
	}
	if err := a.setProjectSelection(ctx, configPath, snapshot, selected); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.Out, "updated project following: %d followed, %d outside Following\n", len(selected), len(snapshot.Projects)-len(selected))
	return err
}

func (a App) setProjectFollowingCommand(configPath *string, followed bool) *cobra.Command {
	verb := "unfollow"
	if followed {
		verb = "follow"
	}
	return a.projectMutationCommand(configPath, verb, followed, false)
}

func (a App) setProjectTrackingCommand(configPath *string, tracked bool) *cobra.Command {
	verb := "track"
	if !tracked {
		verb = "untrack"
	}
	return a.projectMutationCommand(configPath, verb, tracked, true)
}

func (a App) projectMutationCommand(configPath *string, verb string, followed, compatibility bool) *cobra.Command {
	short := verb + " one or more projects"
	if compatibility {
		short += " (compatibility alias)"
	}
	return &cobra.Command{
		Use:   verb + " <project>...",
		Short: short,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageError{fmt.Errorf("%s requires at least one project name or owner/repo", cmd.CommandPath())}
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
				snapshot := agent.AssembleWithRecentWindow(records, cfg.Path, paths.State, time.Now().UTC(), cfg.Settings.StaleAfter)
				return a.trackerFor(cfg.Settings.StaleAfter).ReconcilePartial(snapshot)
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
	var follow, unfollow int
	for _, project := range projects {
		_, selected := selectedSet[project.GitHub]
		if selected && project.TrackingState == model.TrackingUntracked {
			follow++
		}
		if !selected && project.TrackingState != model.TrackingUntracked {
			unfollow++
		}
	}
	return follow, unfollow
}

func followedProjectIDs(projects []model.Project) []string {
	followed := make([]string, 0, len(projects))
	for _, project := range projects {
		if project.TrackingState != model.TrackingUntracked {
			followed = append(followed, project.GitHub)
		}
	}
	sort.Strings(followed)
	return followed
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
