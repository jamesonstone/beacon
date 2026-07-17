package notes

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func loadWorkspace(generalPath string) (Workspace, error) {
	workspace, _, err := loadWorkspaceState(generalPath)
	return workspace, err
}

func loadWorkspaceState(generalPath string) (Workspace, workspaceManifest, error) {
	general, err := load(generalPath)
	if err != nil {
		return Workspace{}, workspaceManifest{}, err
	}
	general.ID = GeneralID
	general.Title = "General"
	manifest, err := readManifest(generalPath)
	if err != nil {
		return Workspace{}, workspaceManifest{}, err
	}
	entries := make(map[string]manifestEntry, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if validID(entry.ID) {
			entries[entry.ID] = entry
		}
	}

	detailTabs, err := discoverDetails(generalPath, entries)
	if err != nil {
		return Workspace{}, workspaceManifest{}, err
	}
	valid := map[string]bool{GeneralID: true, NewTabID: true}
	for _, tab := range detailTabs {
		valid[tab.ID] = true
		entry := entries[tab.ID]
		entry.ID = tab.ID
		entry.CreatedAt = tab.CreatedAt
		entry.OpenedAt = tab.OpenedAt
		entries[tab.ID] = entry
	}

	pinnedIDs := []string{GeneralID}
	manifestPinnedIDs := make([]string, 0, len(manifest.PinnedIDs))
	for _, id := range manifest.PinnedIDs {
		if id != GeneralID && id != NewTabID && valid[id] && indexOf(pinnedIDs, id) < 0 {
			pinnedIDs = append(pinnedIDs, id)
			manifestPinnedIDs = append(manifestPinnedIDs, id)
		}
	}
	openIDs := append([]string{}, pinnedIDs...)
	for _, id := range manifest.OpenIDs {
		if id != GeneralID && valid[id] && indexOf(openIDs, id) < 0 {
			openIDs = append(openIDs, id)
		}
	}
	activeID := manifest.ActiveID
	if indexOf(openIDs, activeID) < 0 {
		activeID = GeneralID
	}
	openSet := make(map[string]bool, len(openIDs))
	for _, id := range openIDs {
		openSet[id] = true
	}
	for index := range detailTabs {
		detailTabs[index].Open = openSet[detailTabs[index].ID]
		detailTabs[index].Pinned = indexOf(pinnedIDs, detailTabs[index].ID) >= 0
	}
	sort.Slice(detailTabs, func(i, j int) bool {
		if !detailTabs[i].OpenedAt.Equal(detailTabs[j].OpenedAt) {
			return detailTabs[i].OpenedAt.After(detailTabs[j].OpenedAt)
		}
		if !detailTabs[i].CreatedAt.Equal(detailTabs[j].CreatedAt) {
			return detailTabs[i].CreatedAt.After(detailTabs[j].CreatedAt)
		}
		return detailTabs[i].ID < detailTabs[j].ID
	})
	tabs := []Tab{{ID: GeneralID, Title: "General", Path: generalPath, UpdatedAt: general.UpdatedAt, Open: true, Pinned: true}}
	if openSet[NewTabID] {
		tabs = append(tabs, Tab{ID: NewTabID, Title: "New Tab", Open: true})
	}
	tabs = append(tabs, detailTabs...)
	workspace := Workspace{
		Version: WorkspaceVersion, ActiveID: activeID, OpenIDs: openIDs,
		PinnedIDs: pinnedIDs, Tabs: tabs,
	}
	if activeID != NewTabID {
		active, loadErr := loadDocument(generalPath, workspace, activeID)
		if loadErr != nil {
			return Workspace{}, workspaceManifest{}, loadErr
		}
		workspace.Active = &active
	}

	entryList := make([]manifestEntry, 0, len(entries))
	for _, entry := range entries {
		entryList = append(entryList, entry)
	}
	sort.Slice(entryList, func(i, j int) bool { return entryList[i].ID < entryList[j].ID })
	normalized := workspaceManifest{
		Version: WorkspaceVersion, ActiveID: activeID, OpenIDs: openIDs,
		PinnedIDs: manifestPinnedIDs, Entries: entryList,
	}
	return workspace, normalized, nil
}

func discoverDetails(generalPath string, entries map[string]manifestEntry) ([]Tab, error) {
	directory := detailDirectory(generalPath)
	info, statErr := os.Lstat(directory)
	if errors.Is(statErr, os.ErrNotExist) {
		return []Tab{}, nil
	}
	if statErr != nil {
		return nil, fmt.Errorf("inspect Beacon detail notes directory: %w", statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("Beacon detail notes directory must be a regular directory: %s", directory)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, fmt.Errorf("secure Beacon detail notes directory: %w", err)
	}
	items, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read Beacon detail notes: %w", err)
	}
	tabs := make([]Tab, 0, len(items))
	for _, item := range items {
		if filepath.Ext(item.Name()) != ".md" {
			continue
		}
		id := strings.TrimSuffix(item.Name(), ".md")
		if !validID(id) {
			continue
		}
		if item.Type()&os.ModeSymlink != 0 || !item.Type().IsRegular() {
			return nil, fmt.Errorf("Beacon detail notes must be regular files: %s", filepath.Join(directory, item.Name()))
		}
		tab, err := loadTabMetadata(filepath.Join(directory, item.Name()), id, entries[id])
		if err != nil {
			return nil, err
		}
		tabs = append(tabs, tab)
	}
	return tabs, nil
}

func loadTabMetadata(path, id string, entry manifestEntry) (Tab, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Tab{}, fmt.Errorf("inspect Beacon detail note %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Tab{}, fmt.Errorf("Beacon detail notes must be regular files: %s", path)
	}
	if info.Size() > MaxBytes {
		return Tab{}, fmt.Errorf("Beacon notes exceed the %d-byte limit", MaxBytes)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return Tab{}, fmt.Errorf("secure Beacon detail note %s: %w", path, err)
	}
	file, err := os.Open(path)
	if err != nil {
		return Tab{}, fmt.Errorf("open Beacon detail note %s: %w", path, err)
	}
	defer file.Close()
	line, readErr := bufio.NewReader(io.LimitReader(file, MaxBytes+1)).ReadString('\n')
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return Tab{}, fmt.Errorf("read Beacon detail note title %s: %w", path, readErr)
	}
	if strings.ContainsRune(line, '\x00') {
		return Tab{}, errors.New("Beacon notes cannot contain NUL bytes")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = info.ModTime().UTC()
	}
	if entry.OpenedAt.IsZero() {
		entry.OpenedAt = entry.CreatedAt
	}
	return Tab{
		ID: id, Title: titleFor(line), Path: path,
		CreatedAt: entry.CreatedAt, UpdatedAt: info.ModTime().UTC(), OpenedAt: entry.OpenedAt,
	}, nil
}

func loadDocument(generalPath string, workspace Workspace, id string) (Document, error) {
	path, err := notePath(generalPath, workspace, id)
	if err != nil {
		return Document{}, err
	}
	document, err := load(path)
	if err != nil {
		return Document{}, err
	}
	document.ID = id
	if id == GeneralID {
		document.Title = "General"
		return document, nil
	}
	document.Title = titleFor(document.Content)
	for _, tab := range workspace.Tabs {
		if tab.ID == id {
			document.CreatedAt = tab.CreatedAt
			document.OpenedAt = tab.OpenedAt
			break
		}
	}
	return document, nil
}

func notePath(generalPath string, workspace Workspace, id string) (string, error) {
	if id == GeneralID {
		return generalPath, nil
	}
	if id == NewTabID {
		return "", errors.New("New Tab is not a Markdown document")
	}
	for _, tab := range workspace.Tabs {
		if tab.ID == id && tab.Path != "" {
			return tab.Path, nil
		}
	}
	return "", fmt.Errorf("Beacon note not found: %s", id)
}

func resolveSelector(workspace Workspace, selector string, allowNew bool) (string, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" || selector == GeneralID {
		return GeneralID, nil
	}
	if selector == NewTabID {
		if allowNew {
			return NewTabID, nil
		}
		return "", errors.New("New Tab is not a Markdown document")
	}
	for _, tab := range workspace.Tabs {
		if tab.ID == selector {
			return tab.ID, nil
		}
	}
	matches := []string{}
	for _, tab := range workspace.Tabs {
		if tab.ID != GeneralID && tab.ID != NewTabID && tab.Title == selector {
			matches = append(matches, tab.ID)
		}
	}
	sort.Strings(matches)
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("Beacon note title %q is ambiguous; use one of: %s", selector, strings.Join(matches, ", "))
	}
	return "", fmt.Errorf("Beacon note not found: %s", selector)
}
