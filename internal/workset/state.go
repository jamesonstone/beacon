package workset

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

const Version = 1

type Entry struct {
	ID                 string                `json:"id"`
	Repository         string                `json:"repository,omitempty"`
	GitHub             string                `json:"github,omitempty"`
	Branch             string                `json:"branch,omitempty"`
	Title              string                `json:"title,omitempty"`
	State              model.AttentionState  `json:"state"`
	Pinned             bool                  `json:"pinned"`
	Manual             bool                  `json:"manual"`
	Explicit           bool                  `json:"explicit"`
	Tags               []string              `json:"tags,omitempty"`
	Note               string                `json:"note,omitempty"`
	NoteUpdatedAt      time.Time             `json:"note_updated_at,omitempty"`
	LastSeenAt         time.Time             `json:"last_seen_at,omitempty"`
	Previous           model.LaneObservation `json:"previous"`
	Current            model.LaneObservation `json:"current"`
	ReactivationReason string                `json:"reactivation_reason,omitempty"`
}

type State struct {
	Version   int       `json:"version"`
	Migrated  bool      `json:"project_tracking_migrated"`
	Order     []string  `json:"order,omitempty"`
	Entries   []Entry   `json:"lanes"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store interface {
	Load(string) (State, error)
	Write(string, State) error
}

type FileStore struct{}

func (m Manager) FrequentRepositories() (map[string]struct{}, error) {
	path, err := ResolvePath()
	if err != nil {
		return nil, err
	}
	stateMutex.Lock()
	defer stateMutex.Unlock()
	state, err := m.store().Load(path)
	if err != nil {
		return nil, err
	}
	repositories := make(map[string]struct{})
	for _, entry := range state.Entries {
		if entry.GitHub == "" || (entry.State == model.AttentionParked && !entry.Pinned) {
			continue
		}
		repositories[entry.GitHub] = struct{}{}
	}
	return repositories, nil
}

func ResolvePath() (string, error) {
	root := os.Getenv("XDG_STATE_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory for lane state: %w", err)
		}
		root = filepath.Join(home, ".local", "state")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve lane state directory: %w", err)
	}
	return filepath.Join(filepath.Clean(absolute), "beacon", "lanes.json"), nil
}

func (FileStore) Load(path string) (State, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{Version: Version, Entries: []Entry{}}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("open lane state %s: %w", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var state State
	if err := decoder.Decode(&state); err != nil {
		return State{}, fmt.Errorf("decode lane state %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return State{}, fmt.Errorf("decode lane state %s: trailing JSON is not supported", path)
		}
		return State{}, fmt.Errorf("decode lane state %s: %w", path, err)
	}
	if err := validate(&state); err != nil {
		return State{}, fmt.Errorf("validate lane state %s: %w", path, err)
	}
	return state, nil
}

func (FileStore) Write(path string, state State) error {
	if err := validate(&state); err != nil {
		return fmt.Errorf("validate lane state: %w", err)
	}
	sort.Slice(state.Entries, func(i, j int) bool { return state.Entries[i].ID < state.Entries[j].ID })
	contents, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lane state: %w", err)
	}
	contents = append(contents, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create lane state directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".beacon-lanes-*.json")
	if err != nil {
		return fmt.Errorf("create temporary lane state: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("replace lane state %s: %w", path, err)
	}
	return nil
}

func validate(state *State) error {
	if state.Version != Version {
		return fmt.Errorf("version must equal %d", Version)
	}
	if state.Entries == nil {
		state.Entries = []Entry{}
	}
	if state.Order == nil {
		state.Order = []string{}
	}
	seen := make(map[string]struct{}, len(state.Entries))
	for index := range state.Entries {
		entry := &state.Entries[index]
		if strings.TrimSpace(entry.ID) == "" {
			return errors.New("lane id is required")
		}
		if _, found := seen[entry.ID]; found {
			return fmt.Errorf("duplicate lane id %q", entry.ID)
		}
		seen[entry.ID] = struct{}{}
		switch entry.State {
		case model.AttentionActive, model.AttentionWaiting, model.AttentionRecent, model.AttentionParked:
		default:
			return fmt.Errorf("invalid attention state %q", entry.State)
		}
		if entry.Manual && strings.TrimSpace(entry.Title) == "" {
			return fmt.Errorf("manual lane %q requires a title", entry.ID)
		}
		tags, err := normalizeTags(entry.Tags)
		if err != nil {
			return fmt.Errorf("lane %q: %w", entry.ID, err)
		}
		entry.Tags = tags
	}
	orderSeen := make(map[string]struct{}, len(state.Order))
	normalizedOrder := make([]string, 0, len(state.Order))
	for _, id := range state.Order {
		if strings.TrimSpace(id) == "" {
			return errors.New("lane order id is required")
		}
		if _, found := orderSeen[id]; found {
			return fmt.Errorf("duplicate lane order id %q", id)
		}
		orderSeen[id] = struct{}{}
		if _, found := seen[id]; found {
			normalizedOrder = append(normalizedOrder, id)
		}
	}
	state.Order = normalizedOrder
	return nil
}

func normalizeTags(tags []string) ([]string, error) {
	if len(tags) > 12 {
		return nil, errors.New("lane may have at most 12 tags")
	}
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return nil, errors.New("lane tag cannot be empty")
		}
		if len([]rune(tag)) > 48 {
			return nil, errors.New("lane tag must be 48 characters or fewer")
		}
		key := strings.ToLower(tag)
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized, nil
}
