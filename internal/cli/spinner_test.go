package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestScanLoaderUsesLighthouseSweepFrames(t *testing.T) {
	want := []string{"◜", "◠", "◝", "◞", "◡", "◟"}
	if !reflect.DeepEqual(scanLoaderFrames, want) {
		t.Fatalf("frames = %v, want %v", scanLoaderFrames, want)
	}
}

func TestScanLoaderSuccessRestoresCursorAndPrintsReady(t *testing.T) {
	var output bytes.Buffer
	loader := startScanLoaderWithInterval(&output, true, true, time.Hour)
	loader.Stop(true)

	text := output.String()
	for _, want := range []string{hideCursor, "◜", "beacon scanning the horizon…", clearLine, "✓", "beacon ready", showCursor} {
		if !strings.Contains(text, want) {
			t.Fatalf("loader output %q does not contain %q", text, want)
		}
	}
	if !strings.Contains(text, scanLoaderColors[0]) {
		t.Fatalf("colored loader output = %q", text)
	}
}

func TestScanLoaderFailureClearsLineAndRestoresCursor(t *testing.T) {
	var output bytes.Buffer
	loader := startScanLoaderWithInterval(&output, true, false, time.Hour)
	loader.Stop(false)

	text := output.String()
	if strings.Contains(text, "beacon ready") {
		t.Fatalf("failed loader reported ready: %q", text)
	}
	if !strings.HasSuffix(text, clearLine+showCursor) {
		t.Fatalf("failed loader did not clear and restore: %q", text)
	}
}

func TestScanLoaderDisabledDoesNotWrite(t *testing.T) {
	var output bytes.Buffer
	loader := startScanLoaderWithInterval(&output, false, true, time.Millisecond)
	loader.Stop(true)
	if output.Len() != 0 {
		t.Fatalf("disabled loader output = %q", output.String())
	}
}

func TestBareDashboardShowsLoaderOnlyForTTY(t *testing.T) {
	for _, test := range []struct {
		name        string
		outputIsTTY bool
		wantLoader  bool
	}{
		{name: "interactive", outputIsTTY: true, wantLoader: true},
		{name: "redirected", outputIsTTY: false, wantLoader: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := executeScanCommand(t, test.outputIsTTY)
			hasLoader := strings.Contains(output, "beacon scanning the horizon…")
			if hasLoader != test.wantLoader {
				t.Fatalf("loader present = %t, want %t; output = %q", hasLoader, test.wantLoader, output)
			}
		})
	}
}

func TestExplicitScanDoesNotShowLoader(t *testing.T) {
	output := executeScanCommand(t, true, "scan", "--no-refresh")
	if strings.Contains(output, "beacon scanning the horizon…") || strings.Contains(output, hideCursor) {
		t.Fatalf("explicit scan included loader output: %q", output)
	}
}

func TestBareDashboardScanFailureRestoresCursor(t *testing.T) {
	output, err := executeScanCommandResult(t, true, errors.New("scan failed"))
	if err == nil || err.Error() != "scan failed" {
		t.Fatalf("error = %v", err)
	}
	if !strings.HasSuffix(output, clearLine+showCursor) {
		t.Fatalf("failed bare scan did not clear and restore: %q", output)
	}
	if strings.Contains(output, "beacon ready") {
		t.Fatalf("failed bare scan reported ready: %q", output)
	}
}

func executeScanCommand(t *testing.T, outputIsTTY bool, args ...string) string {
	t.Helper()
	output, err := executeScanCommandResult(t, outputIsTTY, nil, args...)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func executeScanCommandResult(t *testing.T, outputIsTTY bool, scanErr error, args ...string) (string, error) {
	t.Helper()
	repository := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath, `version: 2
repositories:
  - name: example
    path: `+repository+`
    github: owner/example
`)

	var output bytes.Buffer
	app := App{
		Out: &output, Err: &bytes.Buffer{}, Runner: &recordingRunner{},
		InputIsTTY: func() bool { return false }, OutputIsTTY: func() bool { return outputIsTTY },
		TerminalWidth: func() int { return 120 },
		scannerSource: fixedSnapshotScanner{snapshot: model.Snapshot{
			SchemaVersion: model.SchemaVersion,
			GeneratedAt:   time.Date(2026, 7, 10, 14, 0, 0, 0, time.UTC),
			Groups:        model.Groups{}, Projects: []model.Project{}, Lanes: []model.Lane{}, Errors: []model.ScanError{},
		}, err: scanErr},
	}
	command := app.Root()
	command.SetArgs(append([]string{"--config", configPath, "--color", "never"}, args...))
	err := command.ExecuteContext(context.Background())
	return output.String(), err
}

type fixedSnapshotScanner struct {
	snapshot model.Snapshot
	err      error
}

func (s fixedSnapshotScanner) Scan(context.Context, config.Config, string, bool) (model.Snapshot, error) {
	return s.snapshot, s.err
}
