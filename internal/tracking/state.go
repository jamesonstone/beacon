package tracking

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

const Version = 1

var (
	githubPattern      = regexp.MustCompile(`^[^/\s]+/[^/\s]+$`)
	fingerprintPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type Entry struct {
	GitHub      string    `yaml:"github"`
	Name        string    `yaml:"name"`
	Path        string    `yaml:"path"`
	UntrackedAt time.Time `yaml:"untracked_at"`
	Baseline    string    `yaml:"baseline"`
}

type State struct {
	Version   int     `yaml:"version"`
	Untracked []Entry `yaml:"untracked"`
}

type Store interface {
	Load(string) (State, error)
	Write(string, State) error
}

type FileStore struct{}

func ResolvePath(configPath string) (string, error) {
	if strings.TrimSpace(configPath) == "" {
		return "", errors.New("resolved configuration path is required for tracking state")
	}
	absolute, err := filepath.Abs(configPath)
	if err != nil {
		return "", fmt.Errorf("resolve tracking state path: %w", err)
	}
	return filepath.Join(filepath.Dir(filepath.Clean(absolute)), "tracking.yaml"), nil
}

func (FileStore) Load(path string) (State, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyState(), nil
	}
	if err != nil {
		return State{}, fmt.Errorf("open tracking state %s: %w", path, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	var state State
	if err := decoder.Decode(&state); err != nil {
		return State{}, fmt.Errorf("decode tracking state %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return State{}, fmt.Errorf("decode tracking state %s: multiple YAML documents are not supported", path)
		}
		return State{}, fmt.Errorf("decode tracking state %s: %w", path, err)
	}
	if err := validate(&state); err != nil {
		return State{}, fmt.Errorf("validate tracking state %s: %w", path, err)
	}
	return state, nil
}

func (FileStore) Write(path string, state State) error {
	if err := validate(&state); err != nil {
		return fmt.Errorf("validate tracking state: %w", err)
	}
	sortEntries(state.Untracked)
	contents, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode tracking state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create tracking state directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".beacon-tracking-*.yaml")
	if err != nil {
		return fmt.Errorf("create temporary tracking state: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o644); err != nil {
		file.Close()
		return fmt.Errorf("set tracking state permissions: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return fmt.Errorf("write temporary tracking state: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync temporary tracking state: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary tracking state: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("replace tracking state %s: %w", path, err)
	}
	return nil
}

func emptyState() State {
	return State{Version: Version, Untracked: []Entry{}}
}

func validate(state *State) error {
	if state.Version != Version {
		return fmt.Errorf("version must equal %d", Version)
	}
	if state.Untracked == nil {
		state.Untracked = []Entry{}
	}
	seen := make(map[string]struct{}, len(state.Untracked))
	for index, entry := range state.Untracked {
		if !githubPattern.MatchString(entry.GitHub) {
			return fmt.Errorf("untracked entry %d has invalid github identity %q", index+1, entry.GitHub)
		}
		if _, exists := seen[entry.GitHub]; exists {
			return fmt.Errorf("github identity %q is duplicated", entry.GitHub)
		}
		seen[entry.GitHub] = struct{}{}
		if entry.UntrackedAt.IsZero() {
			return fmt.Errorf("untracked entry %d is missing untracked_at", index+1)
		}
		if !fingerprintPattern.MatchString(entry.Baseline) {
			return fmt.Errorf("untracked entry %d has invalid baseline", index+1)
		}
	}
	sortEntries(state.Untracked)
	return nil
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].GitHub < entries[j].GitHub })
}
