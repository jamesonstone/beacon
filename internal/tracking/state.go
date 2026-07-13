package tracking

import (
	"encoding/json"
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

type StateKind string

const (
	StateMuted   StateKind = "muted"
	StateIgnored StateKind = "ignored"
)

var (
	githubPattern      = regexp.MustCompile(`^[^/\s]+/[^/\s]+$`)
	fingerprintPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type Entry struct {
	GitHub             string    `json:"github" yaml:"github"`
	Name               string    `json:"name" yaml:"name"`
	Path               string    `json:"path" yaml:"path"`
	State              StateKind `json:"state" yaml:"-"`
	UntrackedAt        time.Time `json:"muted_at" yaml:"untracked_at"`
	Baseline           string    `json:"baseline" yaml:"baseline"`
	ProbeBaseline      string    `json:"probe_baseline,omitempty" yaml:"-"`
	ProbeFormat        string    `json:"probe_format,omitempty" yaml:"-"`
	ProbeLocal         string    `json:"probe_local,omitempty" yaml:"-"`
	ProbeRemote        string    `json:"probe_remote,omitempty" yaml:"-"`
	LastProbeAt        time.Time `json:"last_probe_at,omitempty" yaml:"-"`
	ReactivationReason string    `json:"reactivation_reason,omitempty" yaml:"-"`
}

type State struct {
	Version       int            `json:"version" yaml:"version"`
	Untracked     []Entry        `json:"projects" yaml:"untracked"`
	Reactivations []Reactivation `json:"reactivations" yaml:"-"`
}

type Reactivation struct {
	GitHub string    `json:"github"`
	At     time.Time `json:"at"`
	Reason string    `json:"reason"`
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
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory for tracking state: %w", err)
		}
		stateHome = filepath.Join(home, ".local", "state")
	}
	absolute, err := filepath.Abs(stateHome)
	if err != nil {
		return "", fmt.Errorf("resolve tracking state path: %w", err)
	}
	return filepath.Join(filepath.Clean(absolute), "beacon", "tracking.json"), nil
}

func LegacyPath(configPath string) (string, error) {
	if strings.TrimSpace(configPath) == "" {
		return "", errors.New("resolved configuration path is required for legacy tracking state")
	}
	absolute, err := filepath.Abs(configPath)
	if err != nil {
		return "", fmt.Errorf("resolve legacy tracking state path: %w", err)
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

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var state State
	if err := decoder.Decode(&state); err != nil {
		return State{}, fmt.Errorf("decode tracking state %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return State{}, fmt.Errorf("decode tracking state %s: trailing JSON is not supported", path)
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
	contents, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tracking state: %w", err)
	}
	contents = append(contents, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create tracking state directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".beacon-tracking-*.json")
	if err != nil {
		return fmt.Errorf("create temporary tracking state: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
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

func MigrateLegacy(configPath, statePath string) (bool, error) {
	if _, err := os.Stat(statePath); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect tracking state %s: %w", statePath, err)
	}
	legacyPath, err := LegacyPath(configPath)
	if err != nil {
		return false, err
	}
	file, err := os.Open(legacyPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open legacy tracking state %s: %w", legacyPath, err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	var state State
	if err := decoder.Decode(&state); err != nil {
		return false, fmt.Errorf("decode legacy tracking state %s: %w", legacyPath, err)
	}
	for index := range state.Untracked {
		state.Untracked[index].State = StateMuted
	}
	if err := (FileStore{}).Write(statePath, state); err != nil {
		return false, fmt.Errorf("migrate legacy tracking state: %w", err)
	}
	migrated := legacyPath + ".migrated"
	if err := os.Rename(legacyPath, migrated); err != nil {
		return false, fmt.Errorf("archive migrated tracking state: %w", err)
	}
	return true, nil
}

func emptyState() State {
	return State{Version: Version, Untracked: []Entry{}, Reactivations: []Reactivation{}}
}

func validate(state *State) error {
	if state.Version != Version {
		return fmt.Errorf("version must equal %d", Version)
	}
	if state.Untracked == nil {
		state.Untracked = []Entry{}
	}
	if state.Reactivations == nil {
		state.Reactivations = []Reactivation{}
	}
	seen := make(map[string]struct{}, len(state.Untracked))
	for index, entry := range state.Untracked {
		if entry.State == "" {
			state.Untracked[index].State = StateMuted
			entry.State = StateMuted
		}
		if entry.State != StateMuted && entry.State != StateIgnored {
			return fmt.Errorf("untracked entry %d has invalid state %q", index+1, entry.State)
		}
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
		if entry.Baseline != "" && !fingerprintPattern.MatchString(entry.Baseline) {
			return fmt.Errorf("untracked entry %d has invalid baseline", index+1)
		}
		if entry.ProbeBaseline != "" && !fingerprintPattern.MatchString(entry.ProbeBaseline) {
			return fmt.Errorf("untracked entry %d has invalid probe baseline", index+1)
		}
		if entry.ProbeLocal != "" && !fingerprintPattern.MatchString(entry.ProbeLocal) {
			return fmt.Errorf("untracked entry %d has invalid local probe", index+1)
		}
		if entry.ProbeRemote != "" && !fingerprintPattern.MatchString(entry.ProbeRemote) {
			return fmt.Errorf("untracked entry %d has invalid remote probe", index+1)
		}
	}
	for index, reactivation := range state.Reactivations {
		if !githubPattern.MatchString(reactivation.GitHub) || reactivation.At.IsZero() || strings.TrimSpace(reactivation.Reason) == "" {
			return fmt.Errorf("reactivation %d is invalid", index+1)
		}
	}
	if len(state.Reactivations) > 50 {
		state.Reactivations = append([]Reactivation{}, state.Reactivations[len(state.Reactivations)-50:]...)
	}
	sortEntries(state.Untracked)
	return nil
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].GitHub < entries[j].GitHub })
}
