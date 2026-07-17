package notes

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	GeneralID        = "general"
	NewTabID         = "new"
	WorkspaceVersion = 1
)

type Tab struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Path      string    `json:"path,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty,omitzero"`
	UpdatedAt time.Time `json:"updated_at,omitempty,omitzero"`
	OpenedAt  time.Time `json:"opened_at,omitempty,omitzero"`
	Open      bool      `json:"open"`
	Pinned    bool      `json:"pinned,omitempty"`
}

type Workspace struct {
	Version   int       `json:"version"`
	ActiveID  string    `json:"active_id"`
	OpenIDs   []string  `json:"open_ids"`
	PinnedIDs []string  `json:"pinned_ids"`
	Tabs      []Tab     `json:"tabs"`
	Active    *Document `json:"active,omitempty"`
}

type workspaceManifest struct {
	Version   int             `json:"version"`
	ActiveID  string          `json:"active_id"`
	OpenIDs   []string        `json:"open_ids"`
	PinnedIDs []string        `json:"pinned_ids,omitempty"`
	Entries   []manifestEntry `json:"entries"`
}

type manifestEntry struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	OpenedAt  time.Time `json:"opened_at"`
}

func (FileStore) LoadWorkspace(generalPath string) (Workspace, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	return loadWorkspace(generalPath)
}

func (FileStore) LoadNote(generalPath, selector string) (Document, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	workspace, err := loadWorkspace(generalPath)
	if err != nil {
		return Document{}, err
	}
	id, err := resolveSelector(workspace, selector, false)
	if err != nil {
		return Document{}, err
	}
	return loadDocument(generalPath, workspace, id)
}

func (FileStore) WriteNote(generalPath, selector, content string) (Workspace, error) {
	return mutateNote(generalPath, selector, func(path string) (Document, error) {
		return write(path, content)
	})
}

func (FileStore) AppendNote(generalPath, selector, content string) (Workspace, error) {
	return mutateNote(generalPath, selector, func(path string) (Document, error) {
		current, err := load(path)
		if err != nil {
			return Document{}, err
		}
		if content == "" {
			return current, nil
		}
		combined := current.Content
		if combined != "" && !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += content
		if !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		return write(path, combined)
	})
}

func mutateNote(generalPath, selector string, operation func(string) (Document, error)) (Workspace, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	var workspace Workspace
	err := withWriteLock(generalPath, func() error {
		current, err := loadWorkspace(generalPath)
		if err != nil {
			return err
		}
		id, err := resolveSelector(current, selector, false)
		if err != nil {
			return err
		}
		path, err := notePath(generalPath, current, id)
		if err != nil {
			return err
		}
		if _, err := operation(path); err != nil {
			return err
		}
		workspace, err = loadWorkspace(generalPath)
		return err
	})
	return workspace, err
}

func (FileStore) CreateNote(generalPath, content string) (Workspace, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	var workspace Workspace
	err := withWriteLock(generalPath, func() error {
		if err := validate(content); err != nil {
			return err
		}
		current, manifest, err := loadWorkspaceState(generalPath)
		if err != nil {
			return err
		}
		directory := detailDirectory(generalPath)
		if err := secureDirectory(directory); err != nil {
			return err
		}
		now := time.Now().UTC()
		id, err := newNoteID(now)
		if err != nil {
			return err
		}
		path := filepath.Join(directory, id+".md")
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			if err == nil {
				return fmt.Errorf("Beacon note ID collision: %s", id)
			}
			return fmt.Errorf("inspect Beacon detail note %s: %w", path, err)
		}
		if _, err := write(path, content); err != nil {
			return err
		}
		manifest.Entries = append(manifest.Entries, manifestEntry{ID: id, CreatedAt: now, OpenedAt: now})
		manifest.OpenIDs = appendOpenID(current.OpenIDs, id)
		manifest.ActiveID = id
		if err := writeManifest(generalPath, manifest); err != nil {
			return err
		}
		workspace, err = loadWorkspace(generalPath)
		return err
	})
	return workspace, err
}

func (FileStore) OpenNote(generalPath, selector string) (Workspace, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	var workspace Workspace
	err := withWriteLock(generalPath, func() error {
		current, manifest, err := loadWorkspaceState(generalPath)
		if err != nil {
			return err
		}
		id, err := resolveSelector(current, selector, true)
		if err != nil {
			return err
		}
		manifest.OpenIDs = appendOpenID(current.OpenIDs, id)
		manifest.ActiveID = id
		if id != GeneralID && id != NewTabID {
			setOpenedAt(&manifest, id, time.Now().UTC())
		}
		if err := writeManifest(generalPath, manifest); err != nil {
			return err
		}
		workspace, err = loadWorkspace(generalPath)
		return err
	})
	return workspace, err
}

func (FileStore) CloseNote(generalPath, selector string) (Workspace, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	var workspace Workspace
	err := withWriteLock(generalPath, func() error {
		current, manifest, err := loadWorkspaceState(generalPath)
		if err != nil {
			return err
		}
		id, err := resolveSelector(current, selector, true)
		if err != nil {
			return err
		}
		if id == GeneralID {
			return errors.New("General Notes cannot be closed")
		}
		if indexOf(current.PinnedIDs, id) >= 0 {
			return fmt.Errorf("Beacon note is pinned; unpin it before closing: %s", id)
		}
		index := indexOf(current.OpenIDs, id)
		if index < 0 {
			return fmt.Errorf("Beacon note is not open: %s", id)
		}
		manifest.OpenIDs = removeID(current.OpenIDs, id)
		if current.ActiveID == id {
			manifest.ActiveID = GeneralID
			if index > 0 && index-1 < len(manifest.OpenIDs) {
				manifest.ActiveID = manifest.OpenIDs[index-1]
			}
		}
		if err := writeManifest(generalPath, manifest); err != nil {
			return err
		}
		workspace, err = loadWorkspace(generalPath)
		return err
	})
	return workspace, err
}

func (FileStore) DeleteNote(generalPath, selector string) (Workspace, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	var workspace Workspace
	err := withWriteLock(generalPath, func() error {
		current, manifest, err := loadWorkspaceState(generalPath)
		if err != nil {
			return err
		}
		id, err := resolveSelector(current, selector, true)
		if err != nil {
			return err
		}
		switch id {
		case GeneralID:
			return errors.New("General Notes cannot be deleted")
		case NewTabID:
			return errors.New("New Tab cannot be deleted")
		}
		path, err := notePath(generalPath, current, id)
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect Beacon detail note %s: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("Beacon detail notes must be regular files: %s", path)
		}

		index := indexOf(current.OpenIDs, id)
		manifest.OpenIDs = removeID(current.OpenIDs, id)
		manifest.PinnedIDs = removeID(manifest.PinnedIDs, id)
		manifest.Entries = removeManifestEntry(manifest.Entries, id)
		if current.ActiveID == id {
			manifest.ActiveID = GeneralID
			if index > 0 && index-1 < len(manifest.OpenIDs) {
				manifest.ActiveID = manifest.OpenIDs[index-1]
			}
		}
		if err := writeManifest(generalPath, manifest); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("delete Beacon detail note %s: %w", path, err)
		}
		if err := syncDirectory(filepath.Dir(path)); err != nil {
			return err
		}
		workspace, err = loadWorkspace(generalPath)
		return err
	})
	return workspace, err
}

func removeManifestEntry(entries []manifestEntry, id string) []manifestEntry {
	result := make([]manifestEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.ID != id {
			result = append(result, entry)
		}
	}
	return result
}
