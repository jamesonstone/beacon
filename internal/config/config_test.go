package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolvePathPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("BEACON_CONFIG", filepath.Join(home, "environment.yaml"))

	path, err := ResolvePath("explicit.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "explicit.yaml") {
		t.Fatalf("explicit path = %q", path)
	}

	path, err = ResolvePath("")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, "environment.yaml") {
		t.Fatalf("environment path = %q", path)
	}

	t.Setenv("BEACON_CONFIG", "")
	path, err = ResolvePath("")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, ".config", "beacon", "config.yaml") {
		t.Fatalf("default path = %q", path)
	}
}

func TestLoadDefaultsAndExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repositoryPath := filepath.Join(home, "repo")
	if err := os.Mkdir(repositoryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "config.yaml")
	writeConfig(t, path, `version: 1
repositories:
  - name: example
    path: ~/repo
    github: owner/example
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Repositories[0].Path != repositoryPath {
		t.Fatalf("expanded path = %q", cfg.Repositories[0].Path)
	}
	if cfg.Repositories[0].Base != "main" || cfg.Repositories[0].Remote != "origin" {
		t.Fatalf("repository defaults = %#v", cfg.Repositories[0])
	}
	if cfg.Settings.ScanInterval != time.Minute || cfg.Settings.RemoteRefreshInterval != 5*time.Minute || cfg.Settings.StaleAfter != 24*time.Hour {
		t.Fatalf("duration defaults = %#v", cfg.Settings)
	}
	if cfg.Settings.MaxParallel != 4 || cfg.Settings.GitHubAuthor != "@me" {
		t.Fatalf("setting defaults = %#v", cfg.Settings)
	}
}

func TestLoadRejectsUnknownAndDuplicateFields(t *testing.T) {
	repositoryPath := t.TempDir()
	tests := map[string]string{
		"unknown": `version: 1
unexpected: true
repositories:
  - name: example
    path: ` + repositoryPath + `
    github: owner/example
`,
		"duplicate": `version: 1
repositories:
  - name: example
    path: ` + repositoryPath + `
    github: owner/example
  - name: example
    path: ` + repositoryPath + `
    github: owner/other
`,
	}
	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			writeConfig(t, path, contents)
			if _, err := Load(path); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, `version: 1
settings:
  scan_interval: soon
repositories:
  - name: example
    path: `+t.TempDir()+`
    github: owner/example
`)
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "scan_interval") {
		t.Fatalf("error = %v", err)
	}
}

func writeConfig(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
