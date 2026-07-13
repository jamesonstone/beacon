package notes

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const MaxBytes = 256 * 1024

type Document struct {
	Content   string    `json:"content"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at,omitempty,omitzero"`
}

type Store interface {
	Load(string) (Document, error)
	Write(string, string) (Document, error)
	Append(string, string) (Document, error)
}

type FileStore struct{}

var fileMutex sync.Mutex

func ResolvePath() (string, error) {
	root := os.Getenv("XDG_DATA_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory for Beacon notes: %w", err)
		}
		root = filepath.Join(home, ".local", "share")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve Beacon notes directory: %w", err)
	}
	return filepath.Join(filepath.Clean(absolute), "beacon", "notes.md"), nil
}

func (FileStore) Load(path string) (Document, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	return load(path)
}

func load(path string) (Document, error) {
	entry, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Document{Content: "", Path: path}, nil
	}
	if err != nil {
		return Document{}, fmt.Errorf("inspect Beacon notes %s: %w", path, err)
	}
	if entry.Mode()&os.ModeSymlink != 0 || !entry.Mode().IsRegular() {
		return Document{}, fmt.Errorf("Beacon notes must be a regular file: %s", path)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return Document{}, fmt.Errorf("secure Beacon notes %s: %w", path, err)
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return Document{}, fmt.Errorf("secure Beacon notes directory: %w", err)
	}
	file, err := os.Open(path)
	if err != nil {
		return Document{}, fmt.Errorf("open Beacon notes %s: %w", path, err)
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, MaxBytes+1))
	if err != nil {
		return Document{}, fmt.Errorf("read Beacon notes %s: %w", path, err)
	}
	if len(contents) > MaxBytes {
		return Document{}, fmt.Errorf("Beacon notes exceed the %d-byte limit", MaxBytes)
	}
	info, err := file.Stat()
	if err != nil {
		return Document{}, fmt.Errorf("inspect Beacon notes %s: %w", path, err)
	}
	return Document{Content: string(contents), Path: path, UpdatedAt: info.ModTime().UTC()}, nil
}

func (FileStore) Write(path, content string) (Document, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	var document Document
	err := withWriteLock(path, func() error {
		var writeErr error
		document, writeErr = write(path, content)
		return writeErr
	})
	return document, err
}

func write(path, content string) (Document, error) {
	if err := validate(content); err != nil {
		return Document{}, err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return Document{}, fmt.Errorf("create Beacon notes directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return Document{}, fmt.Errorf("secure Beacon notes directory: %w", err)
	}
	file, err := os.CreateTemp(directory, ".beacon-notes-*.md")
	if err != nil {
		return Document{}, fmt.Errorf("create temporary Beacon notes: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return Document{}, fmt.Errorf("secure temporary Beacon notes: %w", err)
	}
	if _, err := io.WriteString(file, content); err != nil {
		file.Close()
		return Document{}, fmt.Errorf("write temporary Beacon notes: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return Document{}, fmt.Errorf("sync temporary Beacon notes: %w", err)
	}
	if err := file.Close(); err != nil {
		return Document{}, fmt.Errorf("close temporary Beacon notes: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return Document{}, fmt.Errorf("replace Beacon notes %s: %w", path, err)
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return Document{}, fmt.Errorf("open Beacon notes directory for sync: %w", err)
	}
	if err := directoryHandle.Sync(); err != nil {
		directoryHandle.Close()
		return Document{}, fmt.Errorf("sync Beacon notes directory: %w", err)
	}
	if err := directoryHandle.Close(); err != nil {
		return Document{}, fmt.Errorf("close Beacon notes directory: %w", err)
	}
	return load(path)
}

func (FileStore) Append(path, content string) (Document, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	var document Document
	err := withWriteLock(path, func() error {
		current, loadErr := load(path)
		if loadErr != nil {
			return loadErr
		}
		if content == "" {
			document = current
			return nil
		}
		combined := current.Content
		if combined != "" && !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += content
		if !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		var writeErr error
		document, writeErr = write(path, combined)
		return writeErr
	})
	return document, err
}

func withWriteLock(path string, operation func() error) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create Beacon notes directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("secure Beacon notes directory: %w", err)
	}
	handle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open Beacon notes directory for lock: %w", err)
	}
	defer handle.Close()
	if err := unix.Flock(int(handle.Fd()), unix.LOCK_EX); err != nil {
		return fmt.Errorf("lock Beacon notes directory: %w", err)
	}
	defer unix.Flock(int(handle.Fd()), unix.LOCK_UN) //nolint:errcheck -- unlock cannot repair a completed transaction
	return operation()
}

func validate(content string) error {
	if len(content) > MaxBytes {
		return fmt.Errorf("Beacon notes exceed the %d-byte limit", MaxBytes)
	}
	if strings.ContainsRune(content, '\x00') {
		return errors.New("Beacon notes cannot contain NUL bytes")
	}
	return nil
}
