package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenTargetCommandUsesPlatformOpeners(t *testing.T) {
	for _, test := range []struct {
		goos     string
		target   string
		wantName string
		wantArgs []string
		wantErr  string
	}{
		{goos: "darwin", target: "https://github.com/owner/repo/pull/2", wantName: "open", wantArgs: []string{"https://github.com/owner/repo/pull/2"}},
		{goos: "linux", target: "/tmp/repo", wantName: "xdg-open", wantArgs: []string{"/tmp/repo"}},
		{goos: "windows", target: "https://github.com/owner/repo/issues/1", wantName: "rundll32", wantArgs: []string{"url.dll,FileProtocolHandler", "https://github.com/owner/repo/issues/1"}},
		{goos: "plan9", target: "/tmp/repo", wantErr: "unsupported"},
		{goos: "linux", target: "", wantErr: "required"},
	} {
		t.Run(test.goos+"/"+test.target, func(t *testing.T) {
			name, args, err := openTargetCommand(test.goos, test.target)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if name != test.wantName || !sameStrings(args, test.wantArgs) {
				t.Fatalf("command = %s %v, want %s %v", name, args, test.wantName, test.wantArgs)
			}
		})
	}
}

func TestConfigOpenUsesPlatformOpener(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("version: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	app := App{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: runner}
	command := app.Root()
	command.SetArgs([]string{"--config", path, "config", "open"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantName, wantArgs, err := openTargetCommand(runtime.GOOS, path)
	if err != nil {
		t.Fatal(err)
	}
	if runner.name != wantName || !sameStrings(runner.args, wantArgs) {
		t.Fatalf("opener = %s %v, want %s %v", runner.name, runner.args, wantName, wantArgs)
	}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
