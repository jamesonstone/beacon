package activity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

const (
	CacheVersion     = 1
	RefreshCoalesce  = 10 * time.Second
	defaultLockWait  = 75 * time.Millisecond
	maxCacheFileSize = 1 << 20
)

type Record struct {
	Provider   string    `json:"provider"`
	State      string    `json:"state"`
	SessionKey string    `json:"session_key"`
	ProjectID  string    `json:"project_id"`
	LaneID     string    `json:"lane_id,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type Cache struct {
	Version          int                  `json:"version"`
	Records          []Record             `json:"records"`
	ProjectRefreshes map[string]time.Time `json:"project_refreshes,omitempty"`
}

type Snapshot struct {
	Version    int       `json:"version"`
	Records    []Record  `json:"records"`
	NextExpiry time.Time `json:"next_expiry,omitzero"`
}

type Store struct {
	Path     string
	LockPath string
	LockWait time.Duration
}

func (s Store) List(ctx context.Context, now time.Time) (Snapshot, error) {
	cache, next, err := s.mutate(ctx, now, nil)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Version: cache.Version, Records: cache.Records, NextExpiry: next}, nil
}

func (s Store) Prune(ctx context.Context, now time.Time) (Snapshot, error) {
	return s.List(ctx, now)
}

func (s Store) Apply(ctx context.Context, event Event, target Target, now time.Time) (Snapshot, bool, error) {
	refresh := false
	cache, next, err := s.mutate(ctx, now, func(cache *Cache) bool {
		changed := false
		for _, record := range cache.Records {
			if record.Provider == event.Provider && record.SessionKey == event.SessionKey && record.ObservedAt.After(event.ObservedAt) {
				return false
			}
		}
		for index := len(cache.Records) - 1; index >= 0; index-- {
			record := cache.Records[index]
			if record.Provider == event.Provider && record.SessionKey == event.SessionKey {
				cache.Records = append(cache.Records[:index], cache.Records[index+1:]...)
				changed = true
			}
		}
		if event.Action == ActionUpsert {
			ttl := TTL(event.State)
			if ttl > 0 {
				cache.Records = append(cache.Records, Record{
					Provider: event.Provider, State: event.State, SessionKey: event.SessionKey,
					ProjectID: target.ProjectID, LaneID: target.LaneID,
					ObservedAt: event.ObservedAt, ExpiresAt: event.ObservedAt.Add(ttl),
				})
				changed = true
			}
		}
		if event.Action == ActionUpsert && event.State == StateTurnFinished {
			last := cache.ProjectRefreshes[target.ProjectID]
			if last.IsZero() || !now.Before(last.Add(RefreshCoalesce)) {
				cache.ProjectRefreshes[target.ProjectID] = now
				refresh = true
				changed = true
			}
		}
		return changed
	})
	if err != nil {
		return Snapshot{}, false, err
	}
	return Snapshot{Version: cache.Version, Records: cache.Records, NextExpiry: next}, refresh, nil
}

func (s Store) mutate(ctx context.Context, now time.Time, apply func(*Cache) bool) (Cache, time.Time, error) {
	release, err := s.lock(ctx)
	if err != nil {
		return Cache{}, time.Time{}, err
	}
	defer release()
	cache, err := s.load()
	if err != nil {
		return Cache{}, time.Time{}, err
	}
	changed, next := prune(&cache, now)
	if apply != nil && apply(&cache) {
		changed = true
	}
	prunedAfterApply, updatedNext := prune(&cache, now)
	changed = changed || prunedAfterApply
	next = updatedNext
	sort.Slice(cache.Records, func(i, j int) bool {
		if cache.Records[i].ProjectID != cache.Records[j].ProjectID {
			return cache.Records[i].ProjectID < cache.Records[j].ProjectID
		}
		if cache.Records[i].LaneID != cache.Records[j].LaneID {
			return cache.Records[i].LaneID < cache.Records[j].LaneID
		}
		if cache.Records[i].Provider != cache.Records[j].Provider {
			return cache.Records[i].Provider < cache.Records[j].Provider
		}
		return cache.Records[i].SessionKey < cache.Records[j].SessionKey
	})
	if changed {
		if err := s.write(cache); err != nil {
			return Cache{}, time.Time{}, err
		}
	}
	return cache, next, nil
}

func (s Store) load() (Cache, error) {
	cache := Cache{Version: CacheVersion, Records: []Record{}, ProjectRefreshes: map[string]time.Time{}}
	contents, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return cache, nil
	}
	if err != nil {
		return Cache{}, fmt.Errorf("read activity cache: %w", err)
	}
	if len(contents) > maxCacheFileSize {
		return Cache{}, errors.New("activity cache exceeds 1 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cache); err != nil {
		return Cache{}, fmt.Errorf("decode activity cache: %w", err)
	}
	if cache.Version != CacheVersion {
		return Cache{}, fmt.Errorf("unsupported activity cache version %d", cache.Version)
	}
	if cache.Records == nil {
		cache.Records = []Record{}
	}
	if cache.ProjectRefreshes == nil {
		cache.ProjectRefreshes = map[string]time.Time{}
	}
	return cache, nil
}

func (s Store) write(cache Cache) error {
	directory := filepath.Dir(s.Path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create activity cache directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("secure activity cache directory: %w", err)
	}
	contents, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("encode activity cache: %w", err)
	}
	contents = append(contents, '\n')
	file, err := os.CreateTemp(directory, ".activity-*.json")
	if err != nil {
		return fmt.Errorf("create activity cache temporary file: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return fmt.Errorf("secure activity cache temporary file: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return fmt.Errorf("write activity cache temporary file: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync activity cache temporary file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close activity cache temporary file: %w", err)
	}
	if err := os.Rename(temporary, s.Path); err != nil {
		return fmt.Errorf("replace activity cache: %w", err)
	}
	return os.Chmod(s.Path, 0o600)
}

func (s Store) lock(ctx context.Context) (func(), error) {
	path := s.LockPath
	if path == "" {
		path = s.Path + ".lock"
	}
	wait := s.LockWait
	if wait <= 0 {
		wait = defaultLockWait
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create activity lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open activity lock: %w", err)
	}
	_ = file.Chmod(0o600)
	deadline := time.Now().Add(wait)
	for {
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			return func() {
				_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
				_ = file.Close()
			}, nil
		}
		if time.Now().After(deadline) {
			file.Close()
			return nil, errors.New("activity cache lock is busy")
		}
		select {
		case <-ctx.Done():
			file.Close()
			return nil, ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func prune(cache *Cache, now time.Time) (bool, time.Time) {
	changed := false
	kept := cache.Records[:0]
	var next time.Time
	for _, record := range cache.Records {
		if !record.ExpiresAt.After(now) {
			changed = true
			continue
		}
		kept = append(kept, record)
		if next.IsZero() || record.ExpiresAt.Before(next) {
			next = record.ExpiresAt
		}
	}
	cache.Records = kept
	for projectID, refreshedAt := range cache.ProjectRefreshes {
		if !now.Before(refreshedAt.Add(RefreshCoalesce)) {
			delete(cache.ProjectRefreshes, projectID)
			changed = true
		}
	}
	return changed, next
}
