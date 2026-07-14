package notes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func readManifest(generalPath string) (workspaceManifest, error) {
	path := manifestPath(generalPath)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultManifest(), nil
	}
	if err != nil {
		return workspaceManifest{}, fmt.Errorf("inspect Beacon notes workspace: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return workspaceManifest{}, fmt.Errorf("Beacon notes workspace must be a regular file: %s", path)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return workspaceManifest{}, fmt.Errorf("secure Beacon notes workspace: %w", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return workspaceManifest{}, fmt.Errorf("read Beacon notes workspace: %w", err)
	}
	var manifest workspaceManifest
	if json.Unmarshal(contents, &manifest) != nil || manifest.Version != WorkspaceVersion {
		return defaultManifest(), nil
	}
	return manifest, nil
}

func writeManifest(generalPath string, manifest workspaceManifest) error {
	manifest.Version = WorkspaceVersion
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Beacon notes workspace: %w", err)
	}
	contents = append(contents, '\n')
	path := manifestPath(generalPath)
	if err := secureDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".beacon-workspace-*.json")
	if err != nil {
		return fmt.Errorf("create temporary Beacon notes workspace: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return fmt.Errorf("secure temporary Beacon notes workspace: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return fmt.Errorf("write temporary Beacon notes workspace: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync temporary Beacon notes workspace: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary Beacon notes workspace: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("replace Beacon notes workspace: %w", err)
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("open Beacon notes workspace directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync Beacon notes workspace directory: %w", err)
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open Beacon notes directory for sync: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync Beacon notes directory: %w", err)
	}
	return nil
}

func defaultManifest() workspaceManifest {
	return workspaceManifest{Version: WorkspaceVersion, ActiveID: GeneralID, OpenIDs: []string{GeneralID}, Entries: []manifestEntry{}}
}

func detailDirectory(generalPath string) string {
	return filepath.Join(filepath.Dir(generalPath), "notes")
}

func manifestPath(generalPath string) string {
	return filepath.Join(detailDirectory(generalPath), "workspace.json")
}

func secureDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create Beacon notes directory: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect Beacon notes directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("Beacon notes directory must be a regular directory: %s", path)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("secure Beacon notes directory: %w", err)
	}
	return nil
}

func newNoteID(now time.Time) (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate Beacon note ID: %w", err)
	}
	return now.Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(bytes), nil
}

func titleFor(content string) string {
	line := content
	if index := strings.IndexByte(line, '\n'); index >= 0 {
		line = line[:index]
	}
	line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
	if line == "" {
		return "Untitled"
	}
	return line
}

func validID(id string) bool {
	if id == "" || id == GeneralID || id == NewTabID || id == "." || id == ".." {
		return false
	}
	for _, character := range id {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune("._-", character) {
			continue
		}
		return false
	}
	return true
}

func appendOpenID(ids []string, id string) []string {
	if indexOf(ids, id) >= 0 {
		return append([]string{}, ids...)
	}
	return append(append([]string{}, ids...), id)
}

func removeID(ids []string, id string) []string {
	result := make([]string, 0, len(ids))
	for _, candidate := range ids {
		if candidate != id {
			result = append(result, candidate)
		}
	}
	return result
}

func indexOf(ids []string, id string) int {
	for index, candidate := range ids {
		if candidate == id {
			return index
		}
	}
	return -1
}

func setOpenedAt(manifest *workspaceManifest, id string, openedAt time.Time) {
	for index := range manifest.Entries {
		if manifest.Entries[index].ID == id {
			manifest.Entries[index].OpenedAt = openedAt
			return
		}
	}
}
