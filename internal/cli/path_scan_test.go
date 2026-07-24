package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestBareBctlUsesConfiguredHyperLightScannerAndDoesNotStartAgent(t *testing.T) {
	project := t.TempDir()
	legacySource := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(
		"version: 2\nprojects:\n  - path: "+project+"\nsources:\n  - path: "+legacySource+"\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
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
	command := app.BctlRoot()
	command.SetArgs([]string{"--config", configPath, "--color", "never", "--no-refresh", "--include-idle", "--json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	canonicalProject, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	if scanner.calls != 1 || scanner.cfg.Path != configPath || scanner.refresh || !scanner.includeIdle ||
		len(scanner.cfg.Sources) != 1 || scanner.cfg.Sources[0].Path != canonicalProject ||
		len(scanner.cfg.Repositories) != 0 {
		t.Fatalf("scanner = %#v", scanner)
	}
	if starter.calls != 0 {
		t.Fatalf("agent starts = %d", starter.calls)
	}
	if !strings.Contains(output.String(), `"schema_version": 1`) {
		t.Fatalf("output = %q", output.String())
	}
}

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
	command := app.BctlRoot()
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

func TestExplicitBctlScanUsesConfiguredSelection(t *testing.T) {
	project := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath, "version: 2\nprojects:\n  - path: "+project+"\n")
	scanner := &recordingWorkScanner{result: model.WorkScan{
		SchemaVersion: model.WorkScanSchemaVersion,
		Items:         []model.WorkItem{},
		Errors:        []model.ScanError{},
		Warnings:      []model.ScanError{},
	}}
	app := App{
		Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &recordingRunner{},
		OutputIsTTY: func() bool { return false }, TerminalWidth: func() int { return 120 },
		workScannerSource: scanner,
	}
	command := app.BctlRoot()
	command.SetArgs([]string{"--config", configPath, "--color", "never", "scan", "--no-refresh", "--json"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	canonicalProject, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	if scanner.calls != 1 || scanner.refresh || len(scanner.cfg.Sources) != 1 ||
		scanner.cfg.Sources[0].Path != canonicalProject {
		t.Fatalf("scanner = %#v", scanner)
	}
}

func TestPathScanRejectsPersistentConfigAndRepositoryFilter(t *testing.T) {
	source := t.TempDir()
	for _, args := range [][]string{
		{"--config", "/tmp/config.yaml", "scan", source},
	} {
		app := App{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, workScannerSource: &recordingWorkScanner{}}
		command := app.BctlRoot()
		command.SetArgs(args)
		err := command.ExecuteContext(context.Background())
		if err == nil || ExitCode(err) != 2 {
			t.Fatalf("args=%v error=%v", args, err)
		}
	}
}

func TestBeaconRejectsMovedHyperLightCommandForms(t *testing.T) {
	source := t.TempDir()
	for _, args := range [][]string{
		{"scan", source},
		{"projects", "--root", source},
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
