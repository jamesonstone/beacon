package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

type Writer interface {
	Write(path string, cfg Config) error
}

type AtomicWriter struct{}

func (AtomicWriter) Write(path string, cfg Config) error {
	contents, err := Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".beacon-config-*.yaml")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o644); err != nil {
		file.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	return nil
}

func Marshal(cfg Config) ([]byte, error) {
	ollamaModel, err := NormalizeOllamaModel(cfg.Settings.OllamaModel)
	if err != nil {
		return nil, err
	}
	cfg.Settings.OllamaModel = ollamaModel
	defaults := defaultSettings()
	if cfg.Settings.ScanInterval <= 0 {
		cfg.Settings.ScanInterval = defaults.ScanInterval
	}
	if cfg.Settings.TrackedRefreshInterval <= 0 {
		cfg.Settings.TrackedRefreshInterval = cfg.Settings.ScanInterval
	}
	if cfg.Settings.UntrackedProbeInterval <= 0 {
		cfg.Settings.UntrackedProbeInterval = defaults.UntrackedProbeInterval
	}
	if cfg.Settings.RemoteRefreshInterval <= 0 {
		cfg.Settings.RemoteRefreshInterval = defaults.RemoteRefreshInterval
	}
	if cfg.Settings.StaleAfter <= 0 {
		cfg.Settings.StaleAfter = defaults.StaleAfter
	}
	if cfg.Settings.MaxParallel == 0 {
		cfg.Settings.MaxParallel = defaults.MaxParallel
	}
	if cfg.Settings.GitHubAuthor == "" {
		cfg.Settings.GitHubAuthor = defaults.GitHubAuthor
	}
	if cfg.Settings.GitHubScope == "" {
		cfg.Settings.GitHubScope = defaults.GitHubScope
	}
	raw := rawConfig{
		Version: Version,
		Settings: rawSettings{
			ScanInterval:           cfg.Settings.ScanInterval.String(),
			TrackedRefreshInterval: cfg.Settings.TrackedRefreshInterval.String(),
			UntrackedProbeInterval: cfg.Settings.UntrackedProbeInterval.String(),
			RemoteRefreshInterval:  cfg.Settings.RemoteRefreshInterval.String(),
			StaleAfter:             cfg.Settings.StaleAfter.String(),
			MaxParallel:            cfg.Settings.MaxParallel,
			GitHubAuthor:           cfg.Settings.GitHubAuthor,
			GitHubScope:            string(cfg.Settings.GitHubScope),
			OllamaModel:            cfg.Settings.OllamaModel,
		},
	}
	for _, project := range cfg.Projects {
		raw.Projects = append(raw.Projects, rawSource{Path: project.Path})
	}
	for _, source := range cfg.Sources {
		raw.Sources = append(raw.Sources, rawSource{Path: source.Path})
	}
	for _, repository := range cfg.Repositories {
		raw.Repositories = append(raw.Repositories, rawRepository{
			Name: repository.Name, Path: repository.Path, GitHub: repository.GitHub,
			Base: repository.Base, Remote: repository.Remote,
		})
	}
	contents, err := yaml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	return contents, nil
}

func Merge(current Config, additions Config) Config {
	merged := current
	merged.Version = Version
	if merged.Settings.MaxParallel == 0 {
		merged.Settings = defaultSettings()
	}
	if additions.Settings.GitHubScope != "" {
		merged.Settings.GitHubScope = additions.Settings.GitHubScope
	}
	if additions.Settings.OllamaModel != "" {
		merged.Settings.OllamaModel = additions.Settings.OllamaModel
	}
	if merged.Settings.GitHubScope == "" {
		merged.Settings.GitHubScope = GitHubScopeMine
	}

	seenSources := make(map[string]struct{}, len(merged.Sources))
	for _, source := range merged.Sources {
		seenSources[source.Path] = struct{}{}
	}
	for _, source := range additions.Sources {
		if _, exists := seenSources[source.Path]; exists {
			continue
		}
		seenSources[source.Path] = struct{}{}
		merged.Sources = append(merged.Sources, source)
	}

	seenCommon := make(map[string]struct{}, len(merged.Repositories))
	seenGitHub := make(map[string]struct{}, len(merged.Repositories))
	seenNames := make(map[string]struct{}, len(merged.Repositories))
	for _, repository := range merged.Repositories {
		seenCommon[repository.Path] = struct{}{}
		seenGitHub[repository.GitHub] = struct{}{}
		seenNames[repository.Name] = struct{}{}
	}
	for _, repository := range additions.Repositories {
		if _, exists := seenCommon[repository.Path]; exists {
			continue
		}
		if _, exists := seenGitHub[repository.GitHub]; exists {
			continue
		}
		name := uniqueName(repository.Name, seenNames)
		repository.Name = name
		seenCommon[repository.Path] = struct{}{}
		seenGitHub[repository.GitHub] = struct{}{}
		seenNames[name] = struct{}{}
		merged.Repositories = append(merged.Repositories, repository)
	}
	return merged
}

// ReplaceProjectPaths replaces bctl's project selection with exact repository
// roots without changing Beacon's legacy discovery inventory.
func ReplaceProjectPaths(current Config, paths []string) (Config, error) {
	selected := make(map[string]struct{}, len(paths))
	replacement := current
	replacement.Version = Version
	replacement.Projects = nil
	for _, path := range paths {
		project, err := normalizeSource(rawSource{Path: path})
		if err != nil {
			return Config{}, fmt.Errorf("project path %q: %w", path, err)
		}
		if _, exists := selected[project.Path]; exists {
			continue
		}
		selected[project.Path] = struct{}{}
		replacement.Projects = append(replacement.Projects, project)
	}
	Sort(&replacement)
	return replacement, nil
}

func defaultSettings() Settings {
	settings, _ := normalizeSettings(rawSettings{})
	return settings
}

func uniqueName(name string, seen map[string]struct{}) string {
	if _, exists := seen[name]; !exists {
		return name
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", name, suffix)
		if _, exists := seen[candidate]; !exists {
			return candidate
		}
	}
}

func Sort(cfg *Config) {
	sort.SliceStable(cfg.Projects, func(i, j int) bool { return cfg.Projects[i].Path < cfg.Projects[j].Path })
	sort.SliceStable(cfg.Sources, func(i, j int) bool { return cfg.Sources[i].Path < cfg.Sources[j].Path })
	sort.SliceStable(cfg.Repositories, func(i, j int) bool {
		if cfg.Repositories[i].Name != cfg.Repositories[j].Name {
			return cfg.Repositories[i].Name < cfg.Repositories[j].Name
		}
		return cfg.Repositories[i].Path < cfg.Repositories[j].Path
	})
}
