package agent

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
)

const launchAgentLabel = "com.jamesonstone.beacon.agent"

type Lifecycle struct {
	Paths        Paths
	Runner       command.Runner
	Executable   string
	StatusSource func(context.Context) (Status, error)
	WaitForReady func(context.Context, string, time.Duration) bool
}

// Start makes the installed user agent available without restarting a healthy
// instance. When no LaunchAgent exists yet, it installs one using the current
// executable before waiting for the socket authority to become ready.
func (l Lifecycle) Start(ctx context.Context) error {
	if runtime.GOOS != "darwin" {
		return errors.New("Beacon background agent startup is supported on macOS only")
	}
	if status, err := l.Status(ctx); err == nil && status.Running {
		return nil
	}
	// Refresh the plist whenever a stopped agent starts so an app update or a
	// switch between the standalone and bundled CLI cannot retain a stale
	// executable path. Install already performs bootout before bootstrap.
	if err := l.Install(ctx); err != nil {
		return err
	}
	wait := l.WaitForReady
	if wait == nil {
		wait = waitForStart
	}
	if wait(ctx, l.Paths.Socket, 2*time.Second) {
		return nil
	}
	return errors.New("Beacon background agent did not become ready")
}

func (l Lifecycle) Install(ctx context.Context) error {
	if runtime.GOOS != "darwin" {
		return errors.New("Beacon background agent installation is supported on macOS only")
	}
	if err := l.Paths.EnsureRuntime(); err != nil {
		return err
	}
	executable := l.Executable
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return fmt.Errorf("resolve Beacon executable: %w", err)
		}
	}
	contents := launchAgentPlist(executable, l.Paths)
	if err := os.MkdirAll(filepath.Dir(l.Paths.LaunchAgent), 0o700); err != nil {
		return fmt.Errorf("create LaunchAgents directory: %w", err)
	}
	if err := atomicWrite(l.Paths.LaunchAgent, []byte(contents), 0o600); err != nil {
		return fmt.Errorf("write LaunchAgent: %w", err)
	}
	domain := "gui/" + strconv.Itoa(os.Getuid())
	_, _ = l.Runner.Run(ctx, "", "launchctl", "bootout", domain, l.Paths.LaunchAgent)
	if _, err := l.Runner.Run(ctx, "", "launchctl", "bootstrap", domain, l.Paths.LaunchAgent); err != nil {
		return fmt.Errorf("load Beacon LaunchAgent: %w", err)
	}
	return nil
}

func (l Lifecycle) Status(ctx context.Context) (Status, error) {
	if l.StatusSource != nil {
		return l.StatusSource(ctx)
	}
	event, err := (Client{Socket: l.Paths.Socket, Timeout: 500 * time.Millisecond}).Request(ctx, Request{Type: RequestGetAgentStatus})
	if err != nil {
		return Status{Running: false, Socket: l.Paths.Socket}, err
	}
	if event.Status == nil {
		return Status{}, errors.New("agent status response is missing status")
	}
	return *event.Status, nil
}

func (l Lifecycle) Stop(ctx context.Context) error {
	if runtime.GOOS == "darwin" {
		if _, statErr := os.Stat(l.Paths.LaunchAgent); statErr == nil {
			domain := "gui/" + strconv.Itoa(os.Getuid())
			if _, err := l.Runner.Run(ctx, "", "launchctl", "bootout", domain, l.Paths.LaunchAgent); err != nil {
				if agentProcessAlive(l.Paths) {
					return fmt.Errorf("stop Beacon LaunchAgent: %w", err)
				}
				return nil
			}
			if waitForStop(l.Paths.Socket, 2*time.Second) {
				return nil
			}
		}
	}
	_, err := (Client{Socket: l.Paths.Socket, Timeout: 500 * time.Millisecond}).Request(ctx, Request{Type: RequestShutdown})
	if err == nil {
		if waitForStop(l.Paths.Socket, 2*time.Second) {
			return nil
		}
		return errors.New("Beacon agent did not stop after shutdown request")
	}
	contents, readErr := os.ReadFile(l.Paths.PID)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return nil
		}
		return readErr
	}
	pid, parseErr := strconv.Atoi(strings.TrimSpace(string(contents)))
	if parseErr != nil || !processAlive(pid) {
		_ = os.Remove(l.Paths.PID)
		return nil
	}
	process, findErr := os.FindProcess(pid)
	if findErr != nil {
		return findErr
	}
	if err := process.Signal(os.Interrupt); err != nil {
		return err
	}
	if waitForStop(l.Paths.Socket, 2*time.Second) {
		return nil
	}
	return errors.New("Beacon agent did not stop after interrupt")
}

func (l Lifecycle) Uninstall(ctx context.Context) error {
	if runtime.GOOS != "darwin" {
		return errors.New("Beacon background agent installation is supported on macOS only")
	}
	domain := "gui/" + strconv.Itoa(os.Getuid())
	_, _ = l.Runner.Run(ctx, "", "launchctl", "bootout", domain, l.Paths.LaunchAgent)
	for _, path := range []string{l.Paths.LaunchAgent, l.Paths.Socket, l.Paths.PID} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	return nil
}

func launchAgentPlist(executable string, paths Paths) string {
	escape := func(value string) string {
		var builder strings.Builder
		_ = xml.EscapeText(&builder, []byte(value))
		return builder.String()
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>` + launchAgentLabel + `</string>
  <key>ProgramArguments</key>
  <array>
    <string>` + escape(executable) + `</string>
    <string>--config</string><string>` + escape(paths.Config) + `</string>
    <string>agent</string><string>serve</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>ProcessType</key><string>Background</string>
  <key>EnvironmentVariables</key>
  <dict><key>PATH</key><string>/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string></dict>
  <key>StandardOutPath</key><string>` + escape(paths.StandardLog) + `</string>
  <key>StandardErrorPath</key><string>` + escape(paths.ErrorLog) + `</string>
</dict>
</plist>
`
}

func RotateLogs(paths Paths, maximum int64) error {
	for _, path := range []string{paths.StandardLog, paths.ErrorLog} {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Size() <= maximum {
			continue
		}
		_ = os.Remove(path + ".1")
		if err := os.Rename(path, path+".1"); err != nil {
			return fmt.Errorf("rotate log %s: %w", path, err)
		}
	}
	return nil
}

func atomicWrite(path string, contents []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".beacon-agent-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(mode); err != nil {
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
	return os.Rename(temporary, path)
}

func waitForStop(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

func waitForStart(ctx context.Context, socket string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := (Lifecycle{Paths: Paths{Socket: socket}}).Status(ctx)
		if err == nil && status.Running {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(25 * time.Millisecond):
		}
	}
	return false
}

func agentProcessAlive(paths Paths) bool {
	contents, err := os.ReadFile(paths.PID)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(contents)))
	return err == nil && processAlive(pid)
}
