package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamesonstone/beacon/internal/notes"
	"github.com/jamesonstone/beacon/internal/tracking"
)

type Paths struct {
	Config            string
	State             string
	Notes             string
	CacheRoot         string
	Projects          string
	Activity          string
	ActivityLock      string
	IntegrationHealth string
	Socket            string
	PID               string
	LaunchAgent       string
	Logs              string
	StandardLog       string
	ErrorLog          string
}

func ResolvePaths(configPath string) (Paths, error) {
	statePath, err := tracking.ResolvePath(configPath)
	if err != nil {
		return Paths{}, err
	}
	notesPath, err := notes.ResolvePath()
	if err != nil {
		return Paths{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve home directory: %w", err)
	}
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		cacheHome = filepath.Join(home, ".cache")
	}
	cacheRoot := filepath.Join(cacheHome, "beacon")
	logs := filepath.Join(home, "Library", "Logs", "Beacon")
	return Paths{
		Config: configPath, State: statePath, Notes: notesPath, CacheRoot: cacheRoot,
		Projects:          filepath.Join(cacheRoot, "projects"),
		Activity:          filepath.Join(cacheRoot, "activity.json"),
		ActivityLock:      filepath.Join(cacheRoot, "activity.lock"),
		IntegrationHealth: filepath.Join(cacheRoot, "integration-health.json"),
		Socket:            filepath.Join(cacheRoot, "agent.sock"),
		PID:               filepath.Join(cacheRoot, "agent.pid"),
		LaunchAgent:       filepath.Join(home, "Library", "LaunchAgents", "com.jamesonstone.beacon.agent.plist"),
		Logs:              logs,
		StandardLog:       filepath.Join(logs, "agent.log"),
		ErrorLog:          filepath.Join(logs, "agent-error.log"),
	}, nil
}

func (p Paths) EnsureRuntime() error {
	for _, directory := range []string{filepath.Dir(p.State), filepath.Dir(p.Notes), p.CacheRoot, p.Projects, p.Logs} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return fmt.Errorf("create Beacon runtime directory %s: %w", directory, err)
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			return fmt.Errorf("secure Beacon runtime directory %s: %w", directory, err)
		}
	}
	return nil
}
