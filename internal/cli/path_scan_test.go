package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestPathScanUsesEphemeralSourcesAndDoesNotStartAgent(t *testing.T) {
	source := t.TempDir()
	scanner := &recordingWorkScanner{result: model.WorkScan{
		SchemaVersion: model.WorkScanSchemaVersion,
		Items:         []model.WorkItem{},
		Errors:        []model.ScanError{},
		Warnings:      []model.ScanError{},
	}}
	starter := &recordingAgentStarter{}
	var output bytes.Buffer
	app := App{
		Out: &output, Err: &bytes.Buffer{}, Runner: &recordingRunner{},
		OutputIsTTY: func() bool { return false }, TerminalWidth: func() int { return 120 },
		workScannerSource: scanner, autoStartAgent: true,
		agentStarterSource: func(agent.Paths) agentStarter { return starter },
	}
	command := app.Root()
	command.SetArgs([]string{"--color", "never", "scan", "--no-refresh", "--json", source})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if scanner.calls != 1 || scanner.refresh || scanner.includeIdle {
		t.Fatalf("scanner calls=%d refresh=%t includeIdle=%t", scanner.calls, scanner.refresh, scanner.includeIdle)
	}
	canonical, err := filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(scanner.cfg.Sources) != 1 || scanner.cfg.Sources[0].Path != canonical || scanner.cfg.Path != "" {
		t.Fatalf("config = %#v", scanner.cfg)
	}
	if starter.calls != 0 {
		t.Fatalf("agent starts = %d", starter.calls)
	}
	if !strings.Contains(output.String(), `"schema_version": 1`) {
		t.Fatalf("output = %q", output.String())
	}
}

func TestPathScanRejectsPersistentConfigAndRepositoryFilter(t *testing.T) {
	source := t.TempDir()
	for _, args := range [][]string{
		{"--config", "/tmp/config.yaml", "scan", source},
		{"scan", "--repo", "example", source},
	} {
		app := App{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, workScannerSource: &recordingWorkScanner{}}
		command := app.Root()
		command.SetArgs(args)
		err := command.ExecuteContext(context.Background())
		if err == nil || ExitCode(err) != 2 {
			t.Fatalf("args=%v error=%v", args, err)
		}
	}
}

type recordingWorkScanner struct {
	result      model.WorkScan
	err         error
	cfg         config.Config
	calls       int
	refresh     bool
	includeIdle bool
}

func (s *recordingWorkScanner) Scan(
	_ context.Context,
	cfg config.Config,
	refresh bool,
	includeIdle bool,
) (model.WorkScan, error) {
	s.calls++
	s.cfg = cfg
	s.refresh = refresh
	s.includeIdle = includeIdle
	return s.result, s.err
}
