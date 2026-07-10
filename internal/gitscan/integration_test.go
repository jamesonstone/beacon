package gitscan

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestScannerDiscoversMultipleWorktrees(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	repositoryPath := filepath.Join(root, "repository")
	featurePath := filepath.Join(root, "feature")
	runGit(t, root, "init", "--bare", remote)
	runGit(t, root, "init", "-b", "main", repositoryPath)
	runGit(t, repositoryPath, "config", "user.name", "Beacon Test")
	runGit(t, repositoryPath, "config", "user.email", "beacon@example.test")
	runGit(t, repositoryPath, "config", "commit.gpgsign", "false")
	writeTestFile(t, filepath.Join(repositoryPath, "README.md"), "main\n")
	runGit(t, repositoryPath, "add", "README.md")
	runGit(t, repositoryPath, "commit", "-m", "initial")
	runGit(t, repositoryPath, "remote", "add", "origin", remote)
	runGit(t, repositoryPath, "push", "-u", "origin", "main")
	runGit(t, repositoryPath, "worktree", "add", "-b", "feature", featurePath, "main")
	writeTestFile(t, filepath.Join(featurePath, "feature.txt"), "feature\n")
	runGit(t, featurePath, "add", "feature.txt")
	runGit(t, featurePath, "commit", "-m", "feature")
	writeTestFile(t, filepath.Join(featurePath, "untracked.txt"), "work in progress\n")

	scanner := Scanner{Runner: command.ExecRunner{}, Now: time.Now}
	result := scanner.Scan(context.Background(), config.Repository{
		Name: "example", Path: repositoryPath, GitHub: "owner/example", Base: "main", Remote: "origin",
	}, false, time.Hour)
	if len(result.Errors) != 0 {
		t.Fatalf("scan errors = %#v", result.Errors)
	}
	if len(result.Lanes) != 2 {
		t.Fatalf("lanes = %#v", result.Lanes)
	}
	var feature *LocalLane
	for index := range result.Lanes {
		if result.Lanes[index].Branch == "feature" {
			feature = &result.Lanes[index]
		}
	}
	if feature == nil {
		t.Fatal("feature worktree was not discovered")
	}
	if feature.Publication != model.PublicationNoUpstream || feature.Worktree.Untracked != 1 || feature.Worktree.AheadBase != 1 {
		t.Fatalf("feature lane = %#v", feature)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
