package gitscan

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

const (
	localTimeout = 5 * time.Second
	fetchTimeout = 30 * time.Second
)

type Scanner struct {
	Runner command.Runner
	Now    func() time.Time
}

type LocalLane struct {
	ID          string
	Branch      string
	Publication model.PublicationState
	Worktree    model.Worktree
}

type Result struct {
	Lanes    []LocalLane
	Refresh  model.Refresh
	Errors   []model.ScanError
	Warnings []model.ScanError
}

type worktreeRecord struct {
	Path     string
	HeadOID  string
	Branch   string
	Detached bool
	Locked   bool
	Prunable bool
}

type statusRecord struct {
	Head       string
	HeadOID    string
	Upstream   string
	Ahead      int
	Behind     int
	Staged     int
	Unstaged   int
	Untracked  int
	Conflicted int
}

func (s Scanner) Scan(ctx context.Context, repo config.Repository, refresh bool, interval time.Duration) Result {
	if s.Now == nil {
		s.Now = time.Now
	}
	result := Result{Refresh: model.Refresh{Repository: repo.Name}}
	result.Refresh = s.refresh(ctx, repo, refresh, interval)
	if result.Refresh.Error != "" {
		result.Errors = append(result.Errors, model.ScanError{Repository: repo.Name, Stage: "fetch", Message: result.Refresh.Error})
	}

	output, err := s.run(ctx, localTimeout, repo.Path, "git", "worktree", "list", "--porcelain", "-z")
	if err != nil {
		result.Errors = append(result.Errors, model.ScanError{Repository: repo.Name, Stage: "worktrees", Message: err.Error()})
		return result
	}
	for _, record := range parseWorktrees(output) {
		lane, scanErrors, scanWarnings := s.scanWorktree(ctx, repo, record)
		result.Lanes = append(result.Lanes, lane)
		for _, scanErr := range scanErrors {
			scanErr.Repository = repo.Name
			result.Errors = append(result.Errors, scanErr)
		}
		for _, scanWarning := range scanWarnings {
			scanWarning.Repository = repo.Name
			result.Warnings = append(result.Warnings, scanWarning)
		}
	}
	return result
}

func (s Scanner) refresh(ctx context.Context, repo config.Repository, enabled bool, interval time.Duration) model.Refresh {
	refresh := model.Refresh{Repository: repo.Name}
	if !enabled {
		return refresh
	}
	commonDirOutput, err := s.run(ctx, localTimeout, repo.Path, "git", "rev-parse", "--git-common-dir")
	if err != nil {
		refresh.Error = err.Error()
		return refresh
	}
	commonDir := strings.TrimSpace(string(commonDirOutput))
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(repo.Path, commonDir)
	}
	if info, statErr := os.Stat(filepath.Join(commonDir, "FETCH_HEAD")); statErr == nil && s.Now().Sub(info.ModTime()) < interval {
		refresh.At = info.ModTime()
		return refresh
	}

	refresh.Attempted = true
	if _, err := s.run(ctx, fetchTimeout, repo.Path, "git", "fetch", "--prune", "--no-tags", repo.Remote); err != nil {
		refresh.Error = err.Error()
		return refresh
	}
	refresh.Refreshed = true
	refresh.At = s.Now()
	return refresh
}

func (s Scanner) scanWorktree(ctx context.Context, repo config.Repository, record worktreeRecord) (LocalLane, []model.ScanError, []model.ScanError) {
	branch := record.Branch
	if branch == "" {
		branch = "detached-" + shortOID(record.HeadOID)
	}
	identity := branch
	if record.Detached {
		pathHash := sha256.Sum256([]byte(filepath.Clean(record.Path)))
		identity += "-" + fmt.Sprintf("%x", pathHash[:8])
	}
	lane := LocalLane{
		ID:     "git:" + repo.GitHub + "@" + url.PathEscape(identity),
		Branch: branch,
		Worktree: model.Worktree{
			Path: record.Path, HeadOID: record.HeadOID, Detached: record.Detached,
			Locked: record.Locked, Prunable: record.Prunable,
		},
		Publication: model.PublicationUnknown,
	}
	if record.Prunable {
		return lane, nil, []model.ScanError{{Stage: "worktree", Message: fmt.Sprintf("worktree is prunable: %s", record.Path)}}
	}

	output, err := s.run(ctx, localTimeout, record.Path, "git", "status", "--porcelain=v2", "--branch", "--untracked-files=all", "-z")
	if err != nil {
		return lane, []model.ScanError{{Stage: "status", Message: err.Error()}}, nil
	}
	status, err := parseStatus(output)
	if err != nil {
		return lane, []model.ScanError{{Stage: "status", Message: err.Error()}}, nil
	}
	if status.Head != "" && status.Head != "(detached)" {
		lane.Branch = status.Head
	}
	lane.Worktree.HeadOID = status.HeadOID
	lane.Worktree.Upstream = status.Upstream
	lane.Worktree.Staged = status.Staged
	lane.Worktree.Unstaged = status.Unstaged
	lane.Worktree.Untracked = status.Untracked
	lane.Worktree.Conflicted = status.Conflicted
	lane.Worktree.Ahead = status.Ahead
	lane.Worktree.Behind = status.Behind
	lane.Worktree.StatusHash = fmt.Sprintf("%x", sha256.Sum256(output))

	var scanErrors []model.ScanError
	baseOutput, err := s.run(ctx, localTimeout, record.Path, "git", "rev-list", "--left-right", "--count", repo.Remote+"/"+repo.Base+"...HEAD")
	if err != nil {
		scanErrors = append(scanErrors, model.ScanError{Stage: "base", Message: err.Error()})
	} else if left, right, parseErr := parseCounts(baseOutput); parseErr != nil {
		scanErrors = append(scanErrors, model.ScanError{Stage: "base", Message: parseErr.Error()})
	} else {
		lane.Worktree.BehindBase = left
		lane.Worktree.AheadBase = right
	}

	dateOutput, err := s.run(ctx, localTimeout, record.Path, "git", "log", "-1", "--format=%cI")
	if err != nil {
		scanErrors = append(scanErrors, model.ScanError{Stage: "commit", Message: err.Error()})
	} else if updatedAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(string(dateOutput))); parseErr == nil {
		lane.Worktree.UpdatedAt = updatedAt
	}
	lane.Publication = publication(repo.Base, lane.Branch, record.Detached, lane.Worktree)
	return lane, scanErrors, nil
}

func (s Scanner) run(ctx context.Context, timeout time.Duration, dir, name string, args ...string) ([]byte, error) {
	commandContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return s.Runner.Run(commandContext, dir, name, args...)
}

func publication(base, branch string, detached bool, worktree model.Worktree) model.PublicationState {
	if detached {
		return model.PublicationUnknown
	}
	if branch == base {
		return model.PublicationBase
	}
	if worktree.Upstream == "" {
		if worktree.AheadBase > 0 {
			return model.PublicationNoUpstream
		}
		return model.PublicationUnknown
	}
	if worktree.Ahead > 0 && worktree.Behind > 0 {
		return model.PublicationDiverged
	}
	if worktree.Ahead > 0 {
		return model.PublicationUnpushed
	}
	if worktree.Behind > 0 {
		return model.PublicationBehind
	}
	return model.PublicationPublished
}
