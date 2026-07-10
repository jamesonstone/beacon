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
	if cfg.Settings.GitHubScope != GitHubScopeMine {
		t.Fatalf("github scope = %q", cfg.Settings.GitHubScope)
	}
}

func TestLoadVersionTwoSources(t *testing.T) {
	sourcePath := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, `version: 2
settings:
  github_scope: all
sources:
  - path: `+sourcePath+`
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != Version || cfg.Settings.GitHubScope != GitHubScopeAll {
		t.Fatalf("config = %#v", cfg)
	}
	canonicalSource, err := filepath.EvalSymlinks(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Sources) != 1 || cfg.Sources[0].Path != canonicalSource {
		t.Fatalf("sources = %#v", cfg.Sources)
	}
}

func TestCanonicalizeSourcePathResolvesAncestorsButRejectsFinalSymlink(t *testing.T) {
	root := t.TempDir()
	realParent := filepath.Join(root, "real")
	realSource := filepath.Join(realParent, "source")
	if err := os.MkdirAll(realSource, 0o755); err != nil {
		t.Fatal(err)
	}
	linkedParent := filepath.Join(root, "linked-parent")
	if err := os.Symlink(realParent, linkedParent); err != nil {
		t.Fatal(err)
	}
	resolved, err := CanonicalizeSourcePath(filepath.Join(linkedParent, "source"))
	canonicalRealSource, evalErr := filepath.EvalSymlinks(realSource)
	if evalErr != nil {
		t.Fatal(evalErr)
	}
	if err != nil || resolved != canonicalRealSource {
		t.Fatalf("resolved = %q, %v", resolved, err)
	}
	finalLink := filepath.Join(root, "source-link")
	if err := os.Symlink(realSource, finalLink); err != nil {
		t.Fatal(err)
	}
	if _, err := CanonicalizeSourcePath(finalLink); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("final symlink error = %v", err)
	}
}

func TestLoadRejectsVersionSpecificAndSourceErrors(t *testing.T) {
	sourcePath := t.TempDir()
	symlinkPath := filepath.Join(t.TempDir(), "source-link")
	if err := os.Symlink(sourcePath, symlinkPath); err != nil {
		t.Fatal(err)
	}
	tests := map[string]string{
		"unsupported version": `version: 3
repositories:
  - name: example
    path: ` + sourcePath + `
    github: owner/example
`,
		"v1 source": `version: 1
sources:
  - path: ` + sourcePath + `
`,
		"v1 scope": `version: 1
settings:
  github_scope: all
repositories:
  - name: example
    path: ` + sourcePath + `
    github: owner/example
`,
		"scope": `version: 2
settings:
  github_scope: friends
sources:
  - path: ` + sourcePath + `
`,
		"duplicate source": `version: 2
sources:
  - path: ` + sourcePath + `
  - path: ` + sourcePath + `
`,
		"missing source": `version: 2
sources:
  - path: ` + filepath.Join(sourcePath, "missing") + `
`,
		"symlink source": `version: 2
sources:
  - path: ` + symlinkPath + `
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
