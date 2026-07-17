package notes

import (
	"errors"
	"fmt"
)

func (FileStore) SetNotePinned(generalPath, selector string, pinned bool) (Workspace, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	return mutatePinnedWorkspace(generalPath, func(current Workspace, manifest *workspaceManifest) error {
		id, err := resolveSelector(current, selector, true)
		if err != nil {
			return err
		}
		switch id {
		case GeneralID:
			if !pinned {
				return errors.New("General Notes is always pinned")
			}
			return nil
		case NewTabID:
			return errors.New("New Tab cannot be pinned")
		}

		manifest.PinnedIDs = detailPinnedIDs(current.PinnedIDs)
		manifest.OpenIDs = append([]string{}, current.OpenIDs...)
		if pinned {
			manifest.PinnedIDs = appendOpenID(manifest.PinnedIDs, id)
			manifest.OpenIDs = appendOpenID(manifest.OpenIDs, id)
		} else {
			manifest.PinnedIDs = removeID(manifest.PinnedIDs, id)
		}
		return nil
	})
}

func (FileStore) ReorderPinnedNotes(generalPath string, selectors []string) (Workspace, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	return mutatePinnedWorkspace(generalPath, func(current Workspace, manifest *workspaceManifest) error {
		ordered := make([]string, 0, len(selectors))
		for _, selector := range selectors {
			id, err := resolveSelector(current, selector, true)
			if err != nil {
				return err
			}
			if id == GeneralID || id == NewTabID {
				return fmt.Errorf("reserved Notes tab cannot be reordered: %s", id)
			}
			if indexOf(ordered, id) >= 0 {
				return fmt.Errorf("duplicate pinned Notes tab: %s", id)
			}
			ordered = append(ordered, id)
		}

		currentPinned := detailPinnedIDs(current.PinnedIDs)
		if !sameIDs(ordered, currentPinned) {
			return errors.New("pinned Notes reorder must include every pinned detail exactly once")
		}
		manifest.PinnedIDs = ordered
		manifest.OpenIDs = append([]string{}, current.OpenIDs...)
		return nil
	})
}

func mutatePinnedWorkspace(
	generalPath string,
	mutation func(Workspace, *workspaceManifest) error,
) (Workspace, error) {
	var workspace Workspace
	err := withWriteLock(generalPath, func() error {
		current, manifest, err := loadWorkspaceState(generalPath)
		if err != nil {
			return err
		}
		if err := mutation(current, &manifest); err != nil {
			return err
		}
		if err := writeManifest(generalPath, manifest); err != nil {
			return err
		}
		workspace, err = loadWorkspace(generalPath)
		return err
	})
	return workspace, err
}

func detailPinnedIDs(ids []string) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != GeneralID && id != NewTabID && indexOf(result, id) < 0 {
			result = append(result, id)
		}
	}
	return result
}

func sameIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for _, id := range left {
		if indexOf(right, id) < 0 {
			return false
		}
	}
	return true
}
