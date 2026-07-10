package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAtomicWriterWritesLoadableVersionTwoConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "config.yaml")
	cfg := Config{
		Version: Version1,
		Settings: Settings{
			ScanInterval: time.Minute, RemoteRefreshInterval: 5 * time.Minute,
			StaleAfter: 24 * time.Hour, MaxParallel: 4, GitHubAuthor: "@me", GitHubScope: GitHubScopeMine,
		},
		Sources: []Source{{Path: root}},
	}
	if err := (AtomicWriter{}).Write(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != Version || len(loaded.Sources) != 1 {
		t.Fatalf("loaded = %#v", loaded)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(contents), "version: 2\n") {
		t.Fatalf("contents = %s", contents)
	}
}

func TestMergePreservesExistingAndDeduplicatesAdditions(t *testing.T) {
	current := Config{
		Version:      Version1,
		Settings:     Settings{MaxParallel: 4, GitHubAuthor: "@me", GitHubScope: GitHubScopeMine},
		Sources:      []Source{{Path: "/source/a"}},
		Repositories: []Repository{{Name: "repo", Path: "/repo/a", GitHub: "owner/repo", Base: "main", Remote: "origin"}},
	}
	additions := Config{
		Settings: Settings{GitHubScope: GitHubScopeAll},
		Sources:  []Source{{Path: "/source/a"}, {Path: "/source/b"}},
		Repositories: []Repository{
			{Name: "repo", Path: "/repo/a", GitHub: "owner/repo"},
			{Name: "repo", Path: "/repo/b", GitHub: "other/repo", Base: "trunk", Remote: "upstream"},
		},
	}
	merged := Merge(current, additions)
	if merged.Version != Version || merged.Settings.GitHubScope != GitHubScopeAll {
		t.Fatalf("merged = %#v", merged)
	}
	if len(merged.Sources) != 2 || len(merged.Repositories) != 2 {
		t.Fatalf("merged collections = %#v %#v", merged.Sources, merged.Repositories)
	}
	if merged.Repositories[1].Name != "repo-2" {
		t.Fatalf("deduplicated name = %q", merged.Repositories[1].Name)
	}
}
