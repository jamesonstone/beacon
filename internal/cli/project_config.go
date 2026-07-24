package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
	"github.com/spf13/cobra"
)

const defaultProjectBrowserRoot = "~/go/src/github.com"

type projectBrowserActionKind string

const (
	projectBrowserEnter  projectBrowserActionKind = "enter"
	projectBrowserUp     projectBrowserActionKind = "up"
	projectBrowserToggle projectBrowserActionKind = "toggle"
	projectBrowserSave   projectBrowserActionKind = "save"
	projectBrowserCancel projectBrowserActionKind = "cancel"
)

type projectBrowserAction struct {
	Kind projectBrowserActionKind
	Path string
}

type projectBrowserEntry struct {
	Name       string
	Path       string
	Repository bool
	Selected   bool
}

type projectBrowserView struct {
	Root            string
	Current         string
	Entries         []projectBrowserEntry
	Selected        int
	SelectedOutside int
	FocusPath       string
}

type configuredProjectPrompter interface {
	ChooseConfiguredProject(context.Context, projectBrowserView) (projectBrowserAction, error)
}

func (a App) configuredProjectsCommand(configPath *string) *cobra.Command {
	var browserRoot string
	command := &cobra.Command{
		Use:   "projects",
		Short: "Select projects bctl scans",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			colorMode, _ := cmd.Flags().GetString("color")
			if _, err := a.resolveColor(colorMode); err != nil {
				return err
			}
			return a.runConfiguredProjectSelector(cmd.Context(), *configPath, browserRoot, cmd.CommandPath())
		},
	}
	command.Flags().StringVar(&browserRoot, "root", defaultProjectBrowserRoot, "project browser root")
	return command
}

func (a App) runConfiguredProjectSelector(ctx context.Context, configPath, root, commandPath string) error {
	if !a.inputIsTTY() {
		return usageError{fmt.Errorf("%s requires a TTY", commandPath)}
	}
	if root == "" {
		root = defaultProjectBrowserRoot
	}
	root, err := config.CanonicalizeSourcePath(root)
	if err != nil {
		return fmt.Errorf("open project browser root: %w", err)
	}
	resolvedConfig, err := config.ResolvePath(configPath)
	if err != nil {
		return err
	}
	cfg, err := config.Load(resolvedConfig)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		cfg = config.Config{Version: config.Version, Path: resolvedConfig}
	}
	selected := configuredProjectPaths(cfg)

	current := root
	focusPath := ""
	for {
		entries, err := projectBrowserEntries(current, selected)
		if err != nil {
			return err
		}
		view := projectBrowserView{
			Root: root, Current: current, Entries: entries, Selected: len(selected),
			SelectedOutside: selectedOutsideRoot(selected, root), FocusPath: focusPath,
		}
		action, err := a.configuredProjectPrompter().ChooseConfiguredProject(ctx, view)
		if err != nil {
			return fmt.Errorf("select configured projects: %w", err)
		}
		switch action.Kind {
		case projectBrowserEnter:
			if !browserEntryMatches(entries, action.Path, false) {
				return errors.New("project browser received an invalid directory selection")
			}
			current = action.Path
			focusPath = ""
		case projectBrowserUp:
			if current == root {
				return errors.New("project browser cannot move above its root")
			}
			focusPath = current
			current = filepath.Dir(current)
		case projectBrowserToggle:
			if !browserEntryMatches(entries, action.Path, true) {
				return errors.New("project browser received an invalid project selection")
			}
			focusPath = action.Path
			if _, exists := selected[action.Path]; exists {
				delete(selected, action.Path)
			} else {
				selected[action.Path] = struct{}{}
			}
		case projectBrowserSave:
			paths := sortedProjectPaths(selected)
			cfg, err = config.ReplaceProjectPaths(cfg, paths)
			if err != nil {
				return err
			}
			if err := (config.AtomicWriter{}).Write(resolvedConfig, cfg); err != nil {
				return err
			}
			_, err = fmt.Fprintf(a.Out, "updated project selection: %d project%s in %s\n", len(paths), pluralSuffix(len(paths)), resolvedConfig)
			return err
		case projectBrowserCancel:
			_, err := fmt.Fprintln(a.Out, "project selection unchanged")
			return err
		default:
			return errors.New("project browser received an unknown action")
		}
	}
}

func configuredProjectPaths(cfg config.Config) map[string]struct{} {
	selected := make(map[string]struct{}, len(cfg.Projects))
	for _, project := range cfg.Projects {
		selected[project.Path] = struct{}{}
	}
	return selected
}

func projectBrowserEntries(directory string, selected map[string]struct{}) ([]projectBrowserEntry, error) {
	if discovery.IsRepositoryRoot(directory) {
		_, isSelected := selected[directory]
		return []projectBrowserEntry{{
			Name: filepath.Base(directory), Path: directory, Repository: true, Selected: isSelected,
		}}, nil
	}
	directoryEntries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read project directory %s: %w", directory, err)
	}
	entries := make([]projectBrowserEntry, 0, len(directoryEntries))
	for _, entry := range directoryEntries {
		if strings.HasPrefix(entry.Name(), ".") || entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		repository := discovery.IsRepositoryRoot(path)
		_, isSelected := selected[path]
		entries = append(entries, projectBrowserEntry{
			Name: entry.Name(), Path: path, Repository: repository, Selected: isSelected,
		})
	}
	return entries, nil
}

func browserEntryMatches(entries []projectBrowserEntry, path string, repository bool) bool {
	for _, entry := range entries {
		if entry.Path == path && entry.Repository == repository {
			return true
		}
	}
	return false
}

func selectedOutsideRoot(selected map[string]struct{}, root string) int {
	count := 0
	for path := range selected {
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			count++
		}
	}
	return count
}

func sortedProjectPaths(selected map[string]struct{}) []string {
	paths := make([]string, 0, len(selected))
	for path := range selected {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func (a App) configuredProjectPrompter() configuredProjectPrompter {
	if a.configuredProjectPrompterSource != nil {
		return a.configuredProjectPrompterSource
	}
	return huhPrompter{input: a.input(), output: a.Out}
}
