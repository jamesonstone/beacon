package activity

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jamesonstone/beacon/internal/model"
)

var ErrUnmatched = errors.New("activity cwd does not map to one followed project")

type Target struct {
	ProjectID string
	LaneID    string
}

type pathTarget struct {
	path   string
	target Target
}

func Match(snapshot model.Snapshot, cwd string) (Target, error) {
	canonicalCWD, err := canonicalExistingDirectory(cwd)
	if err != nil {
		return Target{}, fmt.Errorf("canonicalize activity cwd: %w", err)
	}
	followed := make(map[string]model.Project)
	for _, project := range snapshot.Projects {
		if project.FollowState == model.FollowFollowing {
			followed[project.GitHub] = project
		}
	}
	worktrees := make([]pathTarget, 0)
	for _, lane := range snapshot.Lanes {
		if lane.Worktree == nil || lane.Worktree.Path == "" {
			continue
		}
		if _, ok := followed[lane.GitHub]; !ok {
			continue
		}
		path, pathErr := canonicalExistingDirectory(lane.Worktree.Path)
		if pathErr != nil || !containsPath(path, canonicalCWD) {
			continue
		}
		worktrees = append(worktrees, pathTarget{path: path, target: Target{ProjectID: lane.GitHub, LaneID: lane.ID}})
	}
	if len(worktrees) > 0 {
		longest := 0
		for _, candidate := range worktrees {
			if len(candidate.path) > longest {
				longest = len(candidate.path)
			}
		}
		var selected []pathTarget
		for _, candidate := range worktrees {
			if len(candidate.path) == longest {
				selected = append(selected, candidate)
			}
		}
		if len(selected) != 1 {
			return Target{}, ErrUnmatched
		}
		return selected[0].target, nil
	}
	repositories := make([]Target, 0)
	for _, project := range followed {
		path, pathErr := canonicalExistingDirectory(project.Path)
		if pathErr != nil || !containsPath(path, canonicalCWD) {
			continue
		}
		repositories = append(repositories, Target{ProjectID: project.GitHub})
	}
	if len(repositories) != 1 {
		return Target{}, ErrUnmatched
	}
	return repositories[0], nil
}

func canonicalExistingDirectory(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path is empty")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return filepath.Clean(canonical), nil
}

func containsPath(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil || filepath.IsAbs(relative) {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
