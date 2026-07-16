package agent

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jamesonstone/beacon/internal/checkoutwarn"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

const (
	maxCheckoutConfirmationsPerRefresh = 3
	checkoutConfirmationRetry          = 6 * time.Hour
	checkoutConfirmationErrorRetry     = time.Hour
)

func (e *Engine) enrichCheckoutWarnings(
	ctx context.Context,
	scanID string,
	repository config.Repository,
	record ProjectRecord,
	cached, followed bool,
	snapshot *model.Snapshot,
) []checkoutwarn.Confirmation {
	if !cached || !followed {
		return nil
	}
	confirmations := checkoutConfirmationCandidates(record, *snapshot)
	now := e.now()
	for index := range confirmations {
		confirmation := confirmations[index]
		if confirmation.Status == checkoutwarn.StatusConfirmed || now.Before(confirmation.RetryAfter) {
			continue
		}
		if e.CheckoutConfirmer == nil || !e.takeCheckoutConfirmation(scanID) {
			continue
		}
		confirmed, err := e.CheckoutConfirmer.Confirm(ctx, repository, confirmation)
		if err != nil {
			confirmation.Status = checkoutwarn.StatusPending
			confirmation.CheckedAt = now
			confirmation.RetryAfter = now.Add(checkoutConfirmationErrorRetry)
			confirmations[index] = confirmation
			continue
		}
		if confirmed.Status == checkoutwarn.StatusConfirmed {
			confirmed.RetryAfter = time.Time{}
		} else {
			confirmed.RetryAfter = now.Add(checkoutConfirmationRetry)
		}
		confirmations[index] = confirmed
	}
	attachCheckoutWarnings(snapshot, confirmations)
	return confirmations
}

func checkoutConfirmationCandidates(record ProjectRecord, snapshot model.Snapshot) []checkoutwarn.Confirmation {
	localByBranch := make(map[string]model.Lane)
	openPullRequests := make(map[int]struct{})
	openBranches := make(map[string]struct{})
	for _, lane := range snapshot.Lanes {
		if lane.PullRequest != nil {
			openPullRequests[lane.PullRequest.Number] = struct{}{}
			openBranches[lane.PullRequest.HeadRefName] = struct{}{}
		}
		if lane.Worktree != nil && lane.PullRequest == nil && lane.Branch != "" && lane.Branch != lane.Base {
			localByBranch[lane.Branch] = lane
		}
	}

	confirmations := make([]checkoutwarn.Confirmation, 0, len(record.CheckoutConfirmations)+1)
	seen := make(map[string]struct{})
	appendIfCurrent := func(confirmation checkoutwarn.Confirmation) {
		if _, found := localByBranch[confirmation.Branch]; !found {
			return
		}
		if _, found := openPullRequests[confirmation.PullRequestNumber]; found {
			return
		}
		if _, found := openBranches[confirmation.Branch]; found {
			return
		}
		key := confirmationKey(confirmation)
		if _, found := seen[key]; found {
			return
		}
		seen[key] = struct{}{}
		confirmations = append(confirmations, confirmation)
	}
	for _, confirmation := range record.CheckoutConfirmations {
		appendIfCurrent(confirmation)
	}
	for _, lane := range record.Snapshot.Lanes {
		if lane.PullRequest == nil {
			continue
		}
		pullRequest := lane.PullRequest
		if _, stillOpen := openPullRequests[pullRequest.Number]; stillOpen {
			continue
		}
		local, found := localByBranch[pullRequest.HeadRefName]
		if !found || pullRequest.BaseRefName != local.Base {
			continue
		}
		appendIfCurrent(checkoutwarn.Confirmation{
			PullRequestNumber: pullRequest.Number,
			PullRequestURL:    pullRequest.URL,
			Branch:            pullRequest.HeadRefName,
			Base:              pullRequest.BaseRefName,
			HeadOID:           pullRequest.HeadRefOID,
			Status:            checkoutwarn.StatusPending,
		})
	}
	sort.Slice(confirmations, func(i, j int) bool {
		if confirmations[i].Branch != confirmations[j].Branch {
			return confirmations[i].Branch < confirmations[j].Branch
		}
		return confirmations[i].PullRequestNumber < confirmations[j].PullRequestNumber
	})
	return confirmations
}

func attachCheckoutWarnings(snapshot *model.Snapshot, confirmations []checkoutwarn.Confirmation) {
	byBranch := make(map[string]checkoutwarn.Confirmation, len(confirmations))
	for _, confirmation := range confirmations {
		if confirmation.Status == checkoutwarn.StatusConfirmed {
			byBranch[confirmation.Branch] = confirmation
		}
	}
	for index := range snapshot.Lanes {
		lane := &snapshot.Lanes[index]
		confirmation, found := byBranch[lane.Branch]
		if !found || lane.Worktree == nil || lane.PullRequest != nil {
			continue
		}
		severity := "warning"
		message := fmt.Sprintf(
			"PR #%d merged and the remote %s branch was deleted; this worktree is still checked out on %s.",
			confirmation.PullRequestNumber, confirmation.Branch, confirmation.Branch,
		)
		if lane.Signals.Worktree != model.WorktreeClean {
			severity = "critical"
			message = fmt.Sprintf(
				"PR #%d merged and the remote %s branch was deleted, but this local worktree has changes; review it before switching to %s.",
				confirmation.PullRequestNumber, confirmation.Branch, confirmation.Base,
			)
		} else if confirmation.HeadOID == "" || lane.Worktree.HeadOID != confirmation.HeadOID {
			severity = "critical"
			message = fmt.Sprintf(
				"PR #%d merged and the remote %s branch was deleted, but the local branch has commits beyond the recorded PR head; review it before switching to %s.",
				confirmation.PullRequestNumber, confirmation.Branch, confirmation.Base,
			)
		}
		lane.CheckoutWarning = &model.CheckoutWarning{
			Kind:              "merged_remote_branch_deleted",
			Severity:          severity,
			PullRequestNumber: confirmation.PullRequestNumber,
			PullRequestURL:    confirmation.PullRequestURL,
			Branch:            confirmation.Branch,
			Base:              confirmation.Base,
			MergedAt:          confirmation.MergedAt,
			ConfirmedAt:       confirmation.CheckedAt,
			Message:           message,
		}
	}
}

func confirmationKey(confirmation checkoutwarn.Confirmation) string {
	return fmt.Sprintf("%d:%s:%s", confirmation.PullRequestNumber, confirmation.Branch, confirmation.HeadOID)
}

func (e *Engine) takeCheckoutConfirmation(scanID string) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.checkoutConfirmations == nil {
		e.checkoutConfirmations = make(map[string]int)
	}
	if e.checkoutConfirmations[scanID] >= maxCheckoutConfirmationsPerRefresh {
		return false
	}
	e.checkoutConfirmations[scanID]++
	return true
}

func (e *Engine) clearCheckoutConfirmationBudget(scanID string) {
	e.mutex.Lock()
	delete(e.checkoutConfirmations, scanID)
	e.mutex.Unlock()
}
