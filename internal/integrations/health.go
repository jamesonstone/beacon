package integrations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const healthVersion = 1

type HealthEntry struct {
	Fingerprint string `json:"fingerprint"`
	Observed    bool   `json:"observed"`
}

type healthFile struct {
	Version      int                    `json:"version"`
	Integrations map[string]HealthEntry `json:"integrations"`
}

type HealthStore struct {
	Path string
}

func (s HealthStore) Entry(provider string) (HealthEntry, error) {
	var entry HealthEntry
	err := s.withLock(func() error {
		value, loadErr := s.load()
		if loadErr != nil {
			return loadErr
		}
		entry = value.Integrations[provider]
		return nil
	})
	if err != nil {
		return HealthEntry{}, err
	}
	return entry, nil
}

func (s HealthStore) MarkObserved(provider, fingerprint string) error {
	return s.withLock(func() error {
		value, err := s.load()
		if err != nil {
			return err
		}
		value.Integrations[provider] = HealthEntry{Fingerprint: fingerprint, Observed: true}
		return s.write(value)
	})
}

func (s HealthStore) Reset(provider, fingerprint string) error {
	return s.withLock(func() error {
		value, err := s.load()
		if err != nil {
			return err
		}
		value.Integrations[provider] = HealthEntry{Fingerprint: fingerprint}
		return s.write(value)
	})
}

func (s HealthStore) Remove(provider string) error {
	return s.withLock(func() error {
		value, err := s.load()
		if err != nil {
			return err
		}
		if _, ok := value.Integrations[provider]; !ok {
			return nil
		}
		delete(value.Integrations, provider)
		return s.write(value)
	})
}

func (s HealthStore) load() (healthFile, error) {
	value := healthFile{Version: healthVersion, Integrations: map[string]HealthEntry{}}
	contents, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return value, nil
	}
	if err != nil {
		return healthFile{}, fmt.Errorf("read integration health: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return healthFile{}, fmt.Errorf("decode integration health: %w", err)
	}
	if value.Version != healthVersion {
		return healthFile{}, fmt.Errorf("unsupported integration health version %d", value.Version)
	}
	if value.Integrations == nil {
		value.Integrations = map[string]HealthEntry{}
	}
	return value, nil
}

func (s HealthStore) write(value healthFile) error {
	directory := filepath.Dir(s.Path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create integration health directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("secure integration health directory: %w", err)
	}
	contents, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode integration health: %w", err)
	}
	contents = append(contents, '\n')
	file, err := os.CreateTemp(directory, ".integration-health-*.json")
	if err != nil {
		return fmt.Errorf("create integration health temporary file: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return fmt.Errorf("secure integration health temporary file: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return fmt.Errorf("write integration health temporary file: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync integration health temporary file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close integration health temporary file: %w", err)
	}
	if err := os.Rename(temporary, s.Path); err != nil {
		return fmt.Errorf("replace integration health: %w", err)
	}
	return os.Chmod(s.Path, 0o600)
}

func (s HealthStore) withLock(action func() error) error {
	path := s.Path + ".lock"
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create integration health lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open integration health lock: %w", err)
	}
	defer file.Close()
	_ = file.Chmod(0o600)
	deadline := time.Now().Add(75 * time.Millisecond)
	for {
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
			return action()
		}
		if time.Now().After(deadline) {
			return errors.New("integration health lock is busy")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
