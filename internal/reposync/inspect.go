package reposync

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
)

const (
	localTimeout = 5 * time.Second
	fetchTimeout = 30 * time.Second
)

func (s Service) inspect(ctx context.Context, repository config.Repository, fetch bool) Repository {
	result := Repository{
		ProjectID: repository.GitHub, Name: repository.Name, Path: repository.Path,
		Base: repository.Base, Remote: repository.Remote, State: StateUnavailable,
		Action: ActionNone,
	}
	if fetch {
		refspec := fmt.Sprintf("refs/heads/%s:refs/remotes/%s/%s", repository.Base, repository.Remote, repository.Base)
		if _, err := s.run(ctx, fetchTimeout, repository.Path, "fetch", "--prune", "--no-tags", repository.Remote, refspec); err != nil {
			result.Error = err.Error()
			result.Reason = "Could not refresh the remote default branch; no update was attempted."
			return result
		}
		result.Fetched = true
	}

	branch, branchErr := s.run(ctx, localTimeout, repository.Path, "symbolic-ref", "--quiet", "--short", "HEAD")
	if branchErr != nil {
		result.Detached = true
	} else {
		result.CurrentBranch = strings.TrimSpace(string(branch))
	}
	status, err := s.run(ctx, localTimeout, repository.Path, "status", "--porcelain=v1", "-z", "--untracked-files=normal")
	if err != nil {
		return unavailable(result, "inspect worktree", err)
	}
	result.Dirty = len(status) > 0
	result.currentID, err = s.resolveRef(ctx, repository.Path, "HEAD")
	if err != nil {
		return unavailable(result, "resolve current branch", err)
	}

	result.localDefaultID, err = s.resolveRef(ctx, repository.Path, "refs/heads/"+repository.Base)
	if err != nil {
		result.State = StateBlocked
		result.Reason = fmt.Sprintf("Local default branch %s is missing; update it manually.", repository.Base)
		return result
	}
	remoteRef := fmt.Sprintf("refs/remotes/%s/%s", repository.Remote, repository.Base)
	result.remoteDefaultID, err = s.resolveRef(ctx, repository.Path, remoteRef)
	if err != nil {
		result.Reason = fmt.Sprintf("Remote default branch %s/%s is unavailable; check for updates first.", repository.Remote, repository.Base)
		return result
	}
	result.DefaultAhead, result.DefaultBehind, err = s.compare(ctx, repository.Path, "refs/heads/"+repository.Base, remoteRef)
	if err != nil {
		return unavailable(result, "compare local and remote default branches", err)
	}
	result.CurrentAhead, result.CurrentBehind, err = s.compare(ctx, repository.Path, "HEAD", remoteRef)
	if err != nil {
		return unavailable(result, "compare current and remote default branches", err)
	}
	worktrees, err := s.run(ctx, localTimeout, repository.Path, "worktree", "list", "--porcelain")
	if err != nil {
		return unavailable(result, "inspect worktrees", err)
	}
	result.BaseWorktree = branchWorktree(string(worktrees), repository.Base)
	result.NeedsUpdate = result.CurrentBehind > 0 || result.DefaultBehind > 0
	return classify(result)
}

func classify(result Repository) Repository {
	if result.Detached {
		result.State = StateBlocked
		result.Reason = "HEAD is detached; choose a branch manually before updating."
		return result
	}
	if result.DefaultAhead > 0 && result.DefaultBehind > 0 {
		result.State = StateDiverged
		result.Reason = fmt.Sprintf("Local %s has diverged from %s/%s; Beacon will not reset it.", result.Base, result.Remote, result.Base)
		return result
	}
	if !result.NeedsUpdate {
		if result.CurrentAhead > 0 || result.DefaultAhead > 0 {
			result.State = StateAhead
			result.Reason = fmt.Sprintf("%s contains local commits and is not behind %s/%s.", result.CurrentBranch, result.Remote, result.Base)
		} else {
			result.State = StateCurrent
			result.Reason = fmt.Sprintf("%s is up to date with %s/%s.", result.CurrentBranch, result.Remote, result.Base)
		}
		return result
	}
	if result.Dirty {
		result.State = StateBlocked
		result.Reason = "The checked-out worktree has local changes; commit, stash, or handle them manually."
		return result
	}
	if result.DefaultAhead > 0 {
		result.State = StateDiverged
		result.Reason = fmt.Sprintf("Local %s has commits not contained in %s/%s; Beacon will not rewrite it.", result.Base, result.Remote, result.Base)
		return result
	}
	if result.CurrentBranch == result.Base {
		result.State = StateBehind
		result.CanUpdate = true
		result.Action = ActionFastForward
		result.Reason = fmt.Sprintf("%s is %d commit(s) behind %s/%s and can fast-forward.", result.Base, result.CurrentBehind, result.Remote, result.Base)
		return result
	}
	if result.CurrentAhead == 0 && result.CurrentBehind == 0 && result.DefaultBehind > 0 {
		if result.BaseWorktree != "" && result.BaseWorktree != result.Path {
			result.State = StateBlocked
			result.Reason = fmt.Sprintf("%s is checked out in another worktree at %s; update it there.", result.Base, result.BaseWorktree)
			return result
		}
		result.State = StateBehind
		result.CanUpdate = true
		result.Action = ActionSwitchAndFastForward
		result.Reason = fmt.Sprintf("%s is fully merged into %s/%s; Beacon can return this clean worktree to %s.", result.CurrentBranch, result.Remote, result.Base, result.Base)
		return result
	}
	if result.CurrentBehind > 0 {
		if result.CurrentAhead > 0 {
			result.State = StateDiverged
			result.Reason = fmt.Sprintf("%s has unmerged commits and is %d commit(s) behind %s; merge or rebase manually.", result.CurrentBranch, result.CurrentBehind, result.Base)
			return result
		}
		if result.BaseWorktree != "" && result.BaseWorktree != result.Path {
			result.State = StateBlocked
			result.Reason = fmt.Sprintf("%s is checked out in another worktree at %s; update it there.", result.Base, result.BaseWorktree)
			return result
		}
		result.State = StateBehind
		result.CanUpdate = true
		result.Action = ActionSwitchAndFastForward
		result.Reason = fmt.Sprintf("%s is fully merged and %d commit(s) behind %s; Beacon can return this clean worktree to %s.", result.CurrentBranch, result.CurrentBehind, result.Base, result.Base)
		return result
	}
	if result.DefaultBehind > 0 {
		result.State = StateBlocked
		result.Reason = fmt.Sprintf("Local %s is %d commit(s) behind, but %s has commits not contained in %s/%s; update manually.", result.Base, result.DefaultBehind, result.CurrentBranch, result.Remote, result.Base)
	}
	return result
}

func (s Service) resolveRef(ctx context.Context, path, ref string) (string, error) {
	output, err := s.run(ctx, localTimeout, path, "rev-parse", "--verify", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (s Service) compare(ctx context.Context, path, left, right string) (int, int, error) {
	output, err := s.run(ctx, localTimeout, path, "rev-list", "--left-right", "--count", left+"..."+right)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(string(output))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected git rev-list output %q", strings.TrimSpace(string(output)))
	}
	ahead, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count: %w", err)
	}
	behind, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count: %w", err)
	}
	return ahead, behind, nil
}

func (s Service) run(ctx context.Context, timeout time.Duration, path string, args ...string) ([]byte, error) {
	commandContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return s.Runner.Run(commandContext, path, "git", args...)
}

func branchWorktree(output, branch string) string {
	var worktree string
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			worktree = strings.TrimPrefix(line, "worktree ")
		case line == "branch refs/heads/"+branch:
			return worktree
		}
	}
	return ""
}

func unavailable(result Repository, stage string, err error) Repository {
	result.State = StateUnavailable
	result.Error = err.Error()
	result.Reason = stage + " failed; no update was attempted."
	return result
}
