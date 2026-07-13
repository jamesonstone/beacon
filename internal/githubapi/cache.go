package githubapi

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	cacheVersion          = 1
	activityStaleLifetime = 24 * time.Hour
	repositoryStaleLife   = 30 * 24 * time.Hour
)

type diskEntry struct {
	Version   int       `json:"version"`
	Key       string    `json:"key"`
	UpdatedAt time.Time `json:"updated_at"`
	Output    []byte    `json:"output"`
}

func DefaultCacheDirectory() string {
	root := os.Getenv("XDG_CACHE_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		root = filepath.Join(home, ".cache")
	}
	return filepath.Join(root, "beacon", "github")
}

func (r *Runner) cached(key string, args []string) (cacheEntry, bool) {
	r.cacheMutex.Lock()
	entry, found := r.cache[key]
	r.cacheMutex.Unlock()
	if found {
		if r.now().Sub(entry.updatedAt) <= r.staleLifetime(args) {
			return entry, true
		}
		return cacheEntry{}, false
	}
	entry, found = r.loadDisk(key, args)
	if !found {
		return cacheEntry{}, false
	}
	r.cacheMutex.Lock()
	r.cache[key] = entry
	r.cacheMutex.Unlock()
	return entry, true
}

func (r *Runner) store(key string, output []byte) {
	entry := cacheEntry{output: clone(output), updatedAt: r.now()}
	r.cacheMutex.Lock()
	r.cache[key] = entry
	r.cacheMutex.Unlock()
	_ = r.storeDisk(key, entry)
}

func (r *Runner) loadDisk(key string, args []string) (cacheEntry, bool) {
	if r.cacheDirectory == "" {
		return cacheEntry{}, false
	}
	file, err := os.Open(r.cachePath(key))
	if err != nil {
		return cacheEntry{}, false
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var value diskEntry
	if err := decoder.Decode(&value); err != nil {
		return cacheEntry{}, false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return cacheEntry{}, false
	}
	if value.Version != cacheVersion || value.Key != key || value.UpdatedAt.IsZero() {
		return cacheEntry{}, false
	}
	if r.now().Sub(value.UpdatedAt) > r.staleLifetime(args) {
		return cacheEntry{}, false
	}
	return cacheEntry{output: clone(value.Output), updatedAt: value.UpdatedAt}, true
}

func (r *Runner) storeDisk(key string, entry cacheEntry) error {
	if r.cacheDirectory == "" {
		return nil
	}
	if err := os.MkdirAll(r.cacheDirectory, 0o700); err != nil {
		return fmt.Errorf("create GitHub response cache: %w", err)
	}
	if err := os.Chmod(r.cacheDirectory, 0o700); err != nil {
		return fmt.Errorf("secure GitHub response cache: %w", err)
	}
	contents, err := json.Marshal(diskEntry{
		Version: cacheVersion, Key: key, UpdatedAt: entry.updatedAt, Output: clone(entry.output),
	})
	if err != nil {
		return fmt.Errorf("encode GitHub response cache: %w", err)
	}
	contents = append(contents, '\n')
	file, err := os.CreateTemp(r.cacheDirectory, ".beacon-github-*.json")
	if err != nil {
		return fmt.Errorf("create GitHub response cache temporary file: %w", err)
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
	if err := os.Rename(temporary, r.cachePath(key)); err != nil {
		return fmt.Errorf("replace GitHub response cache: %w", err)
	}
	return nil
}

func (r *Runner) cachePath(key string) string {
	return filepath.Join(r.cacheDirectory, fmt.Sprintf("%x.json", sha256.Sum256([]byte(key))))
}

func (r *Runner) staleLifetime(args []string) time.Duration {
	if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
		return repositoryStaleLife
	}
	return activityStaleLifetime
}
