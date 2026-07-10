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

func TestScanFactsContainsExactly150UniqueFacts(t *testing.T) {
	if len(scanFacts) != 150 {
		t.Fatalf("fact count = %d, want 150", len(scanFacts))
	}
	seen := make(map[string]struct{}, len(scanFacts))
	for _, fact := range scanFacts {
		if strings.TrimSpace(fact) == "" {
			t.Fatal("fact deck contains an empty fact")
		}
		if _, exists := seen[fact]; exists {
			t.Fatalf("duplicate fact = %q", fact)
		}
		seen[fact] = struct{}{}
	}
}

func TestFactDeckNeverRepeatsDuringOneRun(t *testing.T) {
	order := make([]int, len(scanFacts))
	for index := range order {
		order[index] = len(order) - index - 1
	}
	deck := newFactDeck(scanFacts, order)
	seen := make(map[string]struct{}, len(scanFacts))
	for {
		fact := deck.current()
		if _, exists := seen[fact]; exists {
			t.Fatalf("fact repeated before deck exhaustion: %q", fact)
		}
		seen[fact] = struct{}{}
		if !deck.advance() {
			break
		}
	}
	if len(seen) != 150 || deck.advance() {
		t.Fatalf("visited %d facts; exhausted deck advanced again", len(seen))
	}
}

func TestRandomFactDelayIsBetweenOneAndFiveSeconds(t *testing.T) {
	for range 1000 {
		delay := randomFactDelay()
		if delay < time.Second || delay > 5*time.Second {
			t.Fatalf("delay = %s", delay)
		}
	}
}

func TestScanLoaderChangesFactsWithoutRepeatingSelections(t *testing.T) {
	var output bytes.Buffer
	loader := startScanLoaderWithOptions(&output, true, false, scanLoaderOptions{
		frameInterval: time.Hour,
		minFactDelay:  time.Millisecond,
		width:         120,
		facts:         []string{"first odd fact", "second odd fact", "third odd fact"},
		factOrder:     []int{0, 1, 2},
		nextFactDelay: func() time.Duration { return time.Millisecond },
	})
	time.Sleep(20 * time.Millisecond)
	loader.Stop(false)

	text := output.String()
	for _, fact := range []string{"first odd fact", "second odd fact", "third odd fact"} {
		if strings.Count(text, fact) != 1 {
			t.Fatalf("loader output selected %q %d times: %q", fact, strings.Count(text, fact), text)
		}
	}
}

func TestFitLoaderFactHonorsTerminalWidth(t *testing.T) {
	if got := fitLoaderFact("a fact that is too long", 12); got != "a fact th…" {
		t.Fatalf("fitted fact = %q", got)
	}
	if got := fitLoaderFact("short", 12); got != "short" {
		t.Fatalf("short fact = %q", got)
	}
}

func TestScanLoaderSuccessRestoresCursorAndPrintsReady(t *testing.T) {
	var output bytes.Buffer
	loader := testScanLoader(&output, true, true)
	loader.Stop(true)

	text := output.String()
	for _, want := range []string{hideCursor, "◜", "Octopuses have three hearts.", clearLine, "✓", "beacon ready", showCursor} {
		if !strings.Contains(text, want) {
			t.Fatalf("loader output %q does not contain %q", text, want)
		}
	}
	if !strings.Contains(text, scanLoaderColors[0]) {
		t.Fatalf("colored loader output = %q", text)
	}
	if !strings.Contains(text, scanFactColors[0]) {
		t.Fatalf("colored fact output = %q", text)
	}
}

func TestScanLoaderFailureClearsLineAndRestoresCursor(t *testing.T) {
	var output bytes.Buffer
	loader := testScanLoader(&output, true, false)
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
	loader := testScanLoader(&output, false, true)
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
			hasLoader := strings.Contains(output, hideCursor) && containsScanFact(output)
			if hasLoader != test.wantLoader {
				t.Fatalf("loader present = %t, want %t; output = %q", hasLoader, test.wantLoader, output)
			}
		})
	}
}

func TestExplicitScanDoesNotShowLoader(t *testing.T) {
	output := executeScanCommand(t, true, "scan", "--no-refresh")
	if strings.Contains(output, hideCursor) || containsScanFact(output) {
		t.Fatalf("explicit scan included loader output: %q", output)
	}
}

func testScanLoader(writer *bytes.Buffer, enabled, color bool) *scanLoader {
	return startScanLoaderWithOptions(writer, enabled, color, scanLoaderOptions{
		frameInterval: time.Hour,
		minFactDelay:  time.Second,
		width:         120,
		facts:         []string{"Octopuses have three hearts."},
		factOrder:     []int{0},
		nextFactDelay: func() time.Duration { return time.Second },
	})
}

func containsScanFact(output string) bool {
	for _, fact := range scanFacts {
		if strings.Contains(output, fact) {
			return true
		}
	}
	return false
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
