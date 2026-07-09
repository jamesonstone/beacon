package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

const Version = 1

var githubName = regexp.MustCompile(`^[^/\s]+/[^/\s]+$`)

type Settings struct {
	ScanInterval          time.Duration
	RemoteRefreshInterval time.Duration
	StaleAfter            time.Duration
	MaxParallel           int
	GitHubAuthor          string
}

type Repository struct {
	Name   string `yaml:"name" json:"name"`
	Path   string `yaml:"path" json:"path"`
	GitHub string `yaml:"github" json:"github"`
	Base   string `yaml:"base" json:"base"`
	Remote string `yaml:"remote" json:"remote"`
}

type Config struct {
	Version      int
	Settings     Settings
	Repositories []Repository
	Path         string
}

type rawConfig struct {
	Version      int             `yaml:"version"`
	Settings     rawSettings     `yaml:"settings"`
	Repositories []rawRepository `yaml:"repositories"`
}

type rawSettings struct {
	ScanInterval          string `yaml:"scan_interval"`
	RemoteRefreshInterval string `yaml:"remote_refresh_interval"`
	StaleAfter            string `yaml:"stale_after"`
	MaxParallel           int    `yaml:"max_parallel"`
	GitHubAuthor          string `yaml:"github_author"`
}

type rawRepository struct {
	Name   string `yaml:"name"`
	Path   string `yaml:"path"`
	GitHub string `yaml:"github"`
	Base   string `yaml:"base"`
	Remote string `yaml:"remote"`
}

func ResolvePath(explicit string) (string, error) {
	if explicit != "" {
		return expandPath(explicit)
	}
	if value := os.Getenv("BEACON_CONFIG"); value != "" {
		return expandPath(value)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "beacon", "config.yaml"), nil
}

func Load(path string) (Config, error) {
	resolved, err := ResolvePath(path)
	if err != nil {
		return Config{}, err
	}
	file, err := os.Open(resolved)
	if err != nil {
		return Config{}, fmt.Errorf("open config %s: %w", resolved, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	var raw rawConfig
	if err := decoder.Decode(&raw); err != nil {
		return Config{}, fmt.Errorf("decode config %s: %w", resolved, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Config{}, fmt.Errorf("decode config %s: multiple YAML documents are not supported", resolved)
		}
		return Config{}, fmt.Errorf("decode config %s: %w", resolved, err)
	}
	return normalize(raw, resolved)
}

func normalize(raw rawConfig, path string) (Config, error) {
	if raw.Version != Version {
		return Config{}, fmt.Errorf("config version must be %d", Version)
	}
	if len(raw.Repositories) == 0 {
		return Config{}, errors.New("config must contain at least one repository")
	}
	settings, err := normalizeSettings(raw.Settings)
	if err != nil {
		return Config{}, err
	}
	config := Config{Version: raw.Version, Settings: settings, Path: path}
	seen := make(map[string]struct{}, len(raw.Repositories))
	for index, rawRepo := range raw.Repositories {
		repo, err := normalizeRepository(rawRepo)
		if err != nil {
			return Config{}, fmt.Errorf("repository %d: %w", index+1, err)
		}
		if _, exists := seen[repo.Name]; exists {
			return Config{}, fmt.Errorf("repository name %q is duplicated", repo.Name)
		}
		seen[repo.Name] = struct{}{}
		config.Repositories = append(config.Repositories, repo)
	}
	return config, nil
}

func normalizeSettings(raw rawSettings) (Settings, error) {
	settings := Settings{MaxParallel: raw.MaxParallel, GitHubAuthor: raw.GitHubAuthor}
	if settings.MaxParallel == 0 {
		settings.MaxParallel = 4
	}
	if settings.MaxParallel < 1 || settings.MaxParallel > 32 {
		return Settings{}, errors.New("settings.max_parallel must be between 1 and 32")
	}
	if settings.GitHubAuthor == "" {
		settings.GitHubAuthor = "@me"
	}
	var err error
	if settings.ScanInterval, err = durationOrDefault(raw.ScanInterval, time.Minute); err != nil {
		return Settings{}, fmt.Errorf("settings.scan_interval: %w", err)
	}
	if settings.RemoteRefreshInterval, err = durationOrDefault(raw.RemoteRefreshInterval, 5*time.Minute); err != nil {
		return Settings{}, fmt.Errorf("settings.remote_refresh_interval: %w", err)
	}
	if settings.StaleAfter, err = durationOrDefault(raw.StaleAfter, 24*time.Hour); err != nil {
		return Settings{}, fmt.Errorf("settings.stale_after: %w", err)
	}
	return settings, nil
}

func normalizeRepository(raw rawRepository) (Repository, error) {
	repo := Repository{
		Name: strings.TrimSpace(raw.Name), GitHub: strings.TrimSpace(raw.GitHub),
		Base: strings.TrimSpace(raw.Base), Remote: strings.TrimSpace(raw.Remote),
	}
	if repo.Name == "" || raw.Path == "" || repo.GitHub == "" {
		return Repository{}, errors.New("name, path, and github are required")
	}
	if !githubName.MatchString(repo.GitHub) {
		return Repository{}, fmt.Errorf("github must use owner/repository form: %q", repo.GitHub)
	}
	if repo.Base == "" {
		repo.Base = "main"
	}
	if repo.Remote == "" {
		repo.Remote = "origin"
	}
	path, err := expandPath(raw.Path)
	if err != nil {
		return Repository{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Repository{}, fmt.Errorf("inspect path %s: %w", path, err)
	}
	if !info.IsDir() {
		return Repository{}, fmt.Errorf("path is not a directory: %s", path)
	}
	repo.Path = path
	return repo, nil
}

func durationOrDefault(value string, fallback time.Duration) (time.Duration, error) {
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("must be a positive Go duration: %q", value)
	}
	return duration, nil
}

func expandPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %s: %w", path, err)
	}
	return filepath.Clean(absolute), nil
}

func Example() string {
	return `version: 1

settings:
  scan_interval: 1m
  remote_refresh_interval: 5m
  stale_after: 24h
  max_parallel: 4
  github_author: "@me"

repositories:
  - name: beacon
    path: ~/go/src/github.com/jamesonstone/beacon
    github: jamesonstone/beacon
    base: main
    remote: origin
`
}
