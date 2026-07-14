package agent

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLaunchAgentPlistEscapesPathsAndUsesSingleBinary(t *testing.T) {
	paths := testPaths(t.TempDir())
	plist := launchAgentPlist("/Applications/A&B/Beacon", paths)
	for _, expected := range []string{"com.jamesonstone.beacon.agent", "/Applications/A&amp;B/Beacon", "<string>agent</string><string>serve</string>", paths.StandardLog} {
		if !bytes.Contains([]byte(plist), []byte(expected)) {
			t.Fatalf("plist missing %q: %s", expected, plist)
		}
	}
}

func TestLifecycleInstallAndUninstallUseUserOnlyFiles(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("LaunchAgent lifecycle is supported on macOS only")
	}
	paths := testPaths(t.TempDir())
	runner := &lifecycleCommandRunner{}
	lifecycle := Lifecycle{Paths: paths, Runner: runner, Executable: "/Applications/Beacon & Co/Beacon"}
	if err := lifecycle.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(paths.LaunchAgent)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("LaunchAgent mode = %o", info.Mode().Perm())
	}
	contents, err := os.ReadFile(paths.LaunchAgent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "/Applications/Beacon &amp; Co/Beacon") || !strings.Contains(strings.Join(runner.commands, "\n"), "launchctl bootstrap") {
		t.Fatalf("plist=%s commands=%v", contents, runner.commands)
	}
	for _, path := range []string{paths.Socket, paths.PID} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := lifecycle.Uninstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.LaunchAgent, paths.Socket, paths.PID} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("lifecycle file still exists: %s", path)
		}
	}
}

func TestLifecycleStartInstallsOnceAndLeavesHealthyAgentRunning(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("LaunchAgent lifecycle is supported on macOS only")
	}
	paths := testPaths(t.TempDir())
	runner := &lifecycleCommandRunner{}
	running := false
	if err := os.MkdirAll(filepath.Dir(paths.LaunchAgent), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.LaunchAgent, []byte("stale executable"), 0o600); err != nil {
		t.Fatal(err)
	}
	lifecycle := Lifecycle{
		Paths: paths, Runner: runner, Executable: "/Applications/Beacon.app/Contents/MacOS/beacon-cli",
		StatusSource: func(context.Context) (Status, error) {
			if running {
				return Status{Running: true}, nil
			}
			return Status{}, ErrUnavailable
		},
		WaitForReady: func(context.Context, string, time.Duration) bool {
			running = true
			return true
		},
	}

	if err := lifecycle.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	commandsAfterStart := len(runner.commands)
	if commandsAfterStart != 2 || !strings.Contains(strings.Join(runner.commands, "\n"), "launchctl bootstrap") {
		t.Fatalf("start commands = %v", runner.commands)
	}
	plist, err := os.ReadFile(paths.LaunchAgent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plist), "/Applications/Beacon.app/Contents/MacOS/beacon-cli") {
		t.Fatalf("start did not refresh stale plist: %s", plist)
	}
	if err := lifecycle.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != commandsAfterStart {
		t.Fatalf("healthy start added commands: %v", runner.commands)
	}
}

func TestLifecycleStopIsIdempotent(t *testing.T) {
	paths := testPaths(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(paths.LaunchAgent), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.LaunchAgent, []byte("plist"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &lifecycleCommandRunner{}
	lifecycle := Lifecycle{Paths: paths, Runner: runner}

	if err := lifecycle.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	runner.err = errors.New("service is not loaded")
	if err := lifecycle.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	expectedCommands := 0
	if runtime.GOOS == "darwin" {
		expectedCommands = 2
	}
	if len(runner.commands) != expectedCommands {
		t.Fatalf("stop command count = %d, want %d: %v", len(runner.commands), expectedCommands, runner.commands)
	}
}

func TestPIDLockRejectsDuplicateAgentAndRecoversAfterRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.pid")
	release, err := acquirePIDLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquirePIDLock(path); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("duplicate lock error = %v", err)
	}
	release()
	releaseAgain, err := acquirePIDLock(path)
	if err != nil {
		t.Fatal(err)
	}
	releaseAgain()
}
