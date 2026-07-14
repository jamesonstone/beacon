package reposync

import (
	"context"
	"fmt"
	"strings"

	"github.com/jamesonstone/beacon/internal/config"
)

func (s Service) apply(ctx context.Context, repository config.Repository) Repository {
	result := s.inspect(ctx, repository, true)
	if !result.CanUpdate {
		return result
	}
	if err := s.guardApply(ctx, result); err != nil {
		result.State = StateBlocked
		result.CanUpdate = false
		result.Error = err.Error()
		result.Reason = "The repository changed after inspection; no update was attempted."
		return result
	}
	remoteRef := fmt.Sprintf("refs/remotes/%s/%s", repository.Remote, repository.Base)
	var err error
	switch result.Action {
	case ActionFastForward:
		_, err = s.run(ctx, localTimeout, repository.Path, "merge", "--ff-only", remoteRef)
	case ActionSwitchAndFastForward:
		_, err = s.run(ctx, localTimeout, repository.Path, "update-ref", "refs/heads/"+repository.Base, result.remoteDefaultID, result.localDefaultID)
		if err == nil {
			_, err = s.run(ctx, localTimeout, repository.Path, "switch", repository.Base)
		}
	default:
		err = fmt.Errorf("repository has no safe update action")
	}
	if err != nil {
		result.State = StateBlocked
		result.CanUpdate = false
		result.Error = err.Error()
		result.Reason = "The fast-forward guard rejected the update; inspect the repository manually."
		return result
	}
	updated := s.inspect(ctx, repository, false)
	updated.Updated = true
	return updated
}

func (s Service) guardApply(ctx context.Context, result Repository) error {
	status, err := s.run(ctx, localTimeout, result.Path, "status", "--porcelain=v1", "-z", "--untracked-files=normal")
	if err != nil {
		return fmt.Errorf("recheck worktree: %w", err)
	}
	if len(status) > 0 {
		return fmt.Errorf("worktree now has local changes")
	}
	branch, err := s.run(ctx, localTimeout, result.Path, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || strings.TrimSpace(string(branch)) != result.CurrentBranch {
		return fmt.Errorf("checked-out branch changed")
	}
	checks := []struct {
		ref  string
		want string
	}{
		{ref: "HEAD", want: result.currentID},
		{ref: "refs/heads/" + result.Base, want: result.localDefaultID},
		{ref: fmt.Sprintf("refs/remotes/%s/%s", result.Remote, result.Base), want: result.remoteDefaultID},
	}
	for _, check := range checks {
		got, resolveErr := s.resolveRef(ctx, result.Path, check.ref)
		if resolveErr != nil || got != check.want {
			return fmt.Errorf("%s changed", check.ref)
		}
	}
	worktrees, err := s.run(ctx, localTimeout, result.Path, "worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("recheck worktrees: %w", err)
	}
	if got := branchWorktree(string(worktrees), result.Base); got != result.BaseWorktree {
		return fmt.Errorf("default-branch worktree changed")
	}
	return nil
}
