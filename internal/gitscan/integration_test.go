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
	detachedOnePath := filepath.Join(root, "detached-one")
	detachedTwoPath := filepath.Join(root, "detached-two")
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
	runGit(t, repositoryPath, "worktree", "add", "--detach", detachedOnePath, "main")
	runGit(t, repositoryPath, "worktree", "add", "--detach", detachedTwoPath, "main")
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
	if len(result.Lanes) != 4 {
		t.Fatalf("lanes = %#v", result.Lanes)
	}
	var feature *LocalLane
	detached := make([]LocalLane, 0, 2)
	lanePathsByID := make(map[string]string, len(result.Lanes))
	for index := range result.Lanes {
		if previousPath, exists := lanePathsByID[result.Lanes[index].ID]; exists {
			t.Fatalf("duplicate lane ID %q for %q and %q", result.Lanes[index].ID, previousPath, result.Lanes[index].Worktree.Path)
		}
		lanePathsByID[result.Lanes[index].ID] = result.Lanes[index].Worktree.Path
		if result.Lanes[index].Branch == "feature" {
			feature = &result.Lanes[index]
		}
		if result.Lanes[index].Worktree.Detached {
			detached = append(detached, result.Lanes[index])
		}
	}
	if feature == nil {
		t.Fatal("feature worktree was not discovered")
	}
	if feature.Publication != model.PublicationNoUpstream || feature.Worktree.Untracked != 1 || feature.Worktree.AheadBase != 1 {
		t.Fatalf("feature lane = %#v", feature)
	}
	if len(detached) != 2 || detached[0].Branch != detached[1].Branch || detached[0].ID == detached[1].ID {
		t.Fatalf("detached lanes = %#v", detached)
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
