package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CanonicalizeSourcePath rejects a source that is itself a symlink and
// resolves any symlinked ancestors before discovery starts. Discovery can then
// walk the canonical tree without crossing symlink directory entries.
func CanonicalizeSourcePath(path string) (string, error) {
	canonical, err := CanonicalizePath(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(canonical)
	if err != nil {
		return "", fmt.Errorf("inspect path %s: %w", canonical, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("source path is a symbolic link: %s", canonical)
	}
	resolved, err := filepath.EvalSymlinks(canonical)
	if err != nil {
		return "", fmt.Errorf("resolve source path %s: %w", canonical, err)
	}
	return filepath.Clean(resolved), nil
}

func CanonicalizePath(path string) (string, error) {
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
