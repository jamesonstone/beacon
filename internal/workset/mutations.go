package workset

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jamesonstone/beacon/internal/model"
)

func (m Manager) SetAttention(snapshot model.Snapshot, id string, state model.AttentionState) (model.Snapshot, error) {
	if _, err := m.Reconcile(snapshot); err != nil {
		return model.Snapshot{}, err
	}
	if err := m.ensure(snapshot, id); err != nil {
		return model.Snapshot{}, err
	}
	if err := m.update(id, func(entry *Entry) {
		entry.State = state
		entry.Explicit = true
		entry.ReactivationReason = ""
	}); err != nil {
		return model.Snapshot{}, err
	}
	return m.Reconcile(snapshot)
}

func (m Manager) SetPinned(snapshot model.Snapshot, id string, pinned bool) (model.Snapshot, error) {
	if _, err := m.Reconcile(snapshot); err != nil {
		return model.Snapshot{}, err
	}
	if err := m.ensure(snapshot, id); err != nil {
		return model.Snapshot{}, err
	}
	if err := m.update(id, func(entry *Entry) { entry.Pinned = pinned }); err != nil {
		return model.Snapshot{}, err
	}
	return m.Reconcile(snapshot)
}

func (m Manager) SetNote(snapshot model.Snapshot, id, note string) (model.Snapshot, error) {
	if _, err := m.Reconcile(snapshot); err != nil {
		return model.Snapshot{}, err
	}
	if err := m.ensure(snapshot, id); err != nil {
		return model.Snapshot{}, err
	}
	if len([]rune(note)) > 280 {
		return model.Snapshot{}, errors.New("lane note must be 280 characters or fewer")
	}
	if err := m.update(id, func(entry *Entry) {
		entry.Note = strings.TrimSpace(note)
		entry.NoteUpdatedAt = m.now()
		entry.Explicit = entry.Note != ""
	}); err != nil {
		return model.Snapshot{}, err
	}
	return m.Reconcile(snapshot)
}

func (m Manager) AddTag(snapshot model.Snapshot, id, tag string) (model.Snapshot, error) {
	if _, err := m.Reconcile(snapshot); err != nil {
		return model.Snapshot{}, err
	}
	if err := m.ensure(snapshot, id); err != nil {
		return model.Snapshot{}, err
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return model.Snapshot{}, errors.New("lane tag is required")
	}
	var mutationErr error
	if err := m.update(id, func(entry *Entry) {
		tags, err := normalizeTags(append(entry.Tags, tag))
		if err != nil {
			mutationErr = err
			return
		}
		entry.Tags = tags
	}); err != nil {
		return model.Snapshot{}, err
	}
	if mutationErr != nil {
		return model.Snapshot{}, mutationErr
	}
	return m.Reconcile(snapshot)
}

func (m Manager) RemoveTag(snapshot model.Snapshot, id, tag string) (model.Snapshot, error) {
	if _, err := m.Reconcile(snapshot); err != nil {
		return model.Snapshot{}, err
	}
	if err := m.ensure(snapshot, id); err != nil {
		return model.Snapshot{}, err
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return model.Snapshot{}, errors.New("lane tag is required")
	}
	if err := m.update(id, func(entry *Entry) {
		filtered := make([]string, 0, len(entry.Tags))
		for _, existing := range entry.Tags {
			if !strings.EqualFold(existing, tag) {
				filtered = append(filtered, existing)
			}
		}
		entry.Tags = filtered
	}); err != nil {
		return model.Snapshot{}, err
	}
	return m.Reconcile(snapshot)
}

func (m Manager) MarkSeen(snapshot model.Snapshot, id string) (model.Snapshot, error) {
	if _, err := m.Reconcile(snapshot); err != nil {
		return model.Snapshot{}, err
	}
	if err := m.ensure(snapshot, id); err != nil {
		return model.Snapshot{}, err
	}
	if err := m.update(id, func(entry *Entry) {
		entry.Previous = entry.Current
		entry.LastSeenAt = m.now()
	}); err != nil {
		return model.Snapshot{}, err
	}
	return m.Reconcile(snapshot)
}

func (m Manager) AddManual(snapshot model.Snapshot, title string) (model.Snapshot, string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return model.Snapshot{}, "", errors.New("manual lane title is required")
	}
	id, err := manualID()
	if err != nil {
		return model.Snapshot{}, "", err
	}
	now := m.now()
	path, err := ResolvePath()
	if err != nil {
		return model.Snapshot{}, "", err
	}
	stateMutex.Lock()
	state, err := m.store().Load(path)
	if err == nil {
		observation := model.LaneObservation{ObservedAt: now}
		state.Entries = append(state.Entries, Entry{ID: id, Title: title, State: model.AttentionActive, Manual: true, Previous: observation, Current: observation})
		state.UpdatedAt = now
		err = m.store().Write(path, state)
	}
	stateMutex.Unlock()
	if err != nil {
		return model.Snapshot{}, "", err
	}
	updated, err := m.Reconcile(snapshot)
	return updated, id, err
}

func (m Manager) update(id string, mutate func(*Entry)) error {
	path, err := ResolvePath()
	if err != nil {
		return err
	}
	stateMutex.Lock()
	defer stateMutex.Unlock()
	state, err := m.store().Load(path)
	if err != nil {
		return err
	}
	for index := range state.Entries {
		if state.Entries[index].ID == id {
			mutate(&state.Entries[index])
			state.UpdatedAt = m.now()
			return m.store().Write(path, state)
		}
	}
	return fmt.Errorf("lane not found: %s", id)
}

func (m Manager) ensure(snapshot model.Snapshot, id string) error {
	var lane *model.Lane
	for index := range snapshot.Lanes {
		if snapshot.Lanes[index].ID == id {
			lane = &snapshot.Lanes[index]
			break
		}
	}
	if lane == nil {
		return fmt.Errorf("lane not found: %s", id)
	}
	path, err := ResolvePath()
	if err != nil {
		return err
	}
	stateMutex.Lock()
	defer stateMutex.Unlock()
	state, err := m.store().Load(path)
	if err != nil {
		return err
	}
	for _, entry := range state.Entries {
		if entry.ID == id {
			return nil
		}
	}
	observation := observe(*lane, m.now())
	state.Entries = append(state.Entries, Entry{ID: id, Repository: lane.Repository, GitHub: lane.GitHub, Branch: lane.Branch, State: model.AttentionActive, Previous: observation, Current: observation})
	state.UpdatedAt = m.now()
	return m.store().Write(path, state)
}

func manualID() (string, error) {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("create manual lane id: %w", err)
	}
	return "manual:" + hex.EncodeToString(value[:]), nil
}
