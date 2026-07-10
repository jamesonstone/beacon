package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
	"github.com/spf13/cobra"
)

type initOptions struct {
	sources     []string
	githubScope string
	yes         bool
}

type initPrompter interface {
	ChooseMode(context.Context) (initMode, error)
	Directory(context.Context) (string, error)
	SelectRepositories(context.Context, []config.Repository) ([]config.Repository, error)
	Confirm(context.Context, string) (bool, error)
}

type inheritedRunner interface {
	Run(context.Context, string, ...string) error
}

type repositoryDiscoverer interface {
	Discover(context.Context, []config.Source) discovery.Result
}

type initService struct {
	runner     command.Runner
	discoverer repositoryDiscoverer
	writer     config.Writer
	prompter   initPrompter
	authRunner inheritedRunner
	lookup     func(string) (string, error)
	isTTY      func() bool
	out        io.Writer
	errOut     io.Writer
	configPath string
	options    initOptions
}

func (a App) initCommand(configPath *string) *cobra.Command {
	return a.initCommandWithUse(configPath, "init")
}

func (a App) initCommandWithUse(configPath *string, use string) *cobra.Command {
	var options initOptions
	command := &cobra.Command{
		Use: use, Short: "Discover repositories and initialize Beacon", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.newInitService(*configPath, options).run(cmd.Context())
		},
	}
	command.Flags().StringArrayVar(&options.sources, "source", nil, "repository or parent directory to discover (repeatable)")
	command.Flags().StringVar(&options.githubScope, "github-scope", "", "GitHub work scope: mine or all")
	command.Flags().BoolVar(&options.yes, "yes", false, "write configuration without confirmation")
	return command
}

func (a App) newInitService(path string, options initOptions) initService {
	return initService{
		runner: a.Runner, discoverer: discovery.Discoverer{Runner: a.Runner},
		writer: config.AtomicWriter{}, lookup: exec.LookPath,
		prompter:   a.initPrompter(),
		authRunner: inheritedCommand{input: a.input(), output: a.Out, errOut: a.Err},
		isTTY:      a.inputIsTTY,
		out:        a.Out, errOut: a.Err, configPath: path, options: options,
	}
}

func (s initService) run(ctx context.Context) error {
	path, err := config.ResolvePath(s.configPath)
	if err != nil {
		return err
	}
	s.configPath = path
	interactive := s.isTTY()
	if !interactive && len(s.options.sources) == 0 {
		return usageError{errors.New("beacon init requires --source in non-interactive mode; use --source PATH --yes")}
	}
	if !interactive && !s.options.yes {
		return usageError{errors.New("non-interactive initialization requires --yes")}
	}
	if s.options.githubScope != "" && s.options.githubScope != string(config.GitHubScopeMine) && s.options.githubScope != string(config.GitHubScopeAll) {
		return usageError{errors.New("--github-scope must be mine or all")}
	}
	if err := s.checkPrerequisites(ctx); err != nil {
		return err
	}

	current, err := s.loadCurrent()
	if err != nil {
		return err
	}
	additions, found, warnings, err := s.collect(ctx)
	if err != nil {
		return err
	}
	if s.options.githubScope != "" {
		additions.Settings.GitHubScope = config.GitHubScope(s.options.githubScope)
	}
	merged := config.Merge(current, additions)
	config.Sort(&merged)
	if len(merged.Sources) == 0 && len(merged.Repositories) == 0 {
		return errors.New("no accessible GitHub repositories were discovered; configuration was not changed")
	}
	currentDiscovery := s.discoverer.Discover(ctx, merged.Sources)
	warnings = deduplicateWarnings(append(warnings, currentDiscovery.Warnings...))
	found = effectiveRepositoryCount(merged.Repositories, currentDiscovery.Repositories)
	s.preview(merged, found, warnings)
	if !s.options.yes {
		confirmed, err := s.prompter.Confirm(ctx, "Write this Beacon configuration?")
		if err != nil {
			return fmt.Errorf("confirm configuration: %w", err)
		}
		if !confirmed {
			return errors.New("initialization cancelled; configuration was not changed")
		}
	}
	if err := s.writer.Write(s.configPath, merged); err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.out, "configured %s\n", s.configPath)
	return err
}

func effectiveRepositoryCount(explicit, discovered []config.Repository) int {
	seen := make(map[string]struct{}, len(explicit)+len(discovered))
	for _, repository := range append(append([]config.Repository{}, explicit...), discovered...) {
		seen[repository.GitHub] = struct{}{}
	}
	return len(seen)
}

func (s initService) collect(ctx context.Context) (config.Config, int, []discovery.Warning, error) {
	if len(s.options.sources) > 0 {
		return s.collectSources(ctx, s.options.sources)
	}
	mode, err := s.prompter.ChooseMode(ctx)
	if err != nil {
		return config.Config{}, 0, nil, fmt.Errorf("choose initialization mode: %w", err)
	}
	directory, err := s.prompter.Directory(ctx)
	if err != nil {
		return config.Config{}, 0, nil, fmt.Errorf("choose source directory: %w", err)
	}
	if mode == initModeWatch {
		return s.collectSources(ctx, []string{directory})
	}
	path, err := validateSourcePath(directory)
	if err != nil {
		return config.Config{}, 0, nil, err
	}
	result := s.discoverer.Discover(ctx, []config.Source{{Path: path}})
	selected, err := s.prompter.SelectRepositories(ctx, result.Repositories)
	if err != nil {
		return config.Config{}, 0, nil, fmt.Errorf("select repositories: %w", err)
	}
	return config.Config{Repositories: selected}, len(result.Repositories), result.Warnings, nil
}

func (s initService) collectSources(ctx context.Context, values []string) (config.Config, int, []discovery.Warning, error) {
	var additions config.Config
	var allWarnings []discovery.Warning
	for _, value := range values {
		path, err := validateSourcePath(value)
		if err != nil {
			return config.Config{}, 0, nil, err
		}
		if discovery.IsRepositoryRoot(path) {
			result := s.discoverer.Discover(ctx, []config.Source{{Path: path}})
			allWarnings = append(allWarnings, result.Warnings...)
			additions.Repositories = append(additions.Repositories, result.Repositories...)
		} else {
			additions.Sources = append(additions.Sources, config.Source{Path: path})
		}
	}
	discovered := s.discoverer.Discover(ctx, additions.Sources)
	allWarnings = append(allWarnings, discovered.Warnings...)
	return additions, len(additions.Repositories) + len(discovered.Repositories), deduplicateWarnings(allWarnings), nil
}

func validateSourcePath(value string) (string, error) {
	path, err := config.CanonicalizeSourcePath(value)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("inspect source %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("source must not be a symbolic link: %s", path)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("source is not a directory: %s", path)
	}
	return path, nil
}

func (s initService) loadCurrent() (config.Config, error) {
	cfg, err := config.Load(s.configPath)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return config.Config{}, nil
	}
	return config.Config{}, err
}

func (s initService) preview(cfg config.Config, found int, warnings []discovery.Warning) {
	fmt.Fprintf(s.out, "\nBeacon configuration preview\n  sources: %d\n  explicit repositories: %d\n  repositories currently discovered: %d\n  GitHub scope: %s\n",
		len(cfg.Sources), len(cfg.Repositories), found, cfg.Settings.GitHubScope)
	for _, warning := range warnings {
		fmt.Fprintf(s.errOut, "warning: %s (%s): %s\n", warning.Path, warning.Stage, warning.Message)
	}
}

func deduplicateWarnings(warnings []discovery.Warning) []discovery.Warning {
	sort.Slice(warnings, func(i, j int) bool {
		left := warnings[i].Path + "\x00" + warnings[i].Stage + "\x00" + warnings[i].Message
		right := warnings[j].Path + "\x00" + warnings[j].Stage + "\x00" + warnings[j].Message
		return left < right
	})
	result := warnings[:0]
	for _, warning := range warnings {
		if len(result) > 0 && result[len(result)-1] == warning {
			continue
		}
		result = append(result, warning)
	}
	return result
}

func installGuidance(name string) string {
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("%s is required; install it with: brew install %s", name, name)
	}
	if name == "gh" {
		return "gh is required; see https://cli.github.com/manual/installation"
	}
	return "git is required; install Git with your system package manager"
}
