package agent

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
