package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/checkoutwarn"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
)

func TestCheckoutWarningConfirmsObservedTransitionAndPersists(t *testing.T) {
	now := time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC)
	confirmer := &recordingCheckoutConfirmer{now: now, status: checkoutwarn.StatusConfirmed}
	engine := NewEngine(config.Config{}, Paths{}, Cache{}, nil, nil, nil, tracking.Manager{})
	engine.Now = func() time.Time { return now }
	engine.CheckoutConfirmer = confirmer
	record, snapshot := checkoutTransition("owner/repo", 32)

	confirmations := engine.enrichCheckoutWarnings(
		context.Background(), "scan-1", checkoutRepository("owner/repo"), record, true, true, &snapshot,
	)
	if len(confirmations) != 1 || confirmer.calls != 1 {
		t.Fatalf("confirmations=%#v calls=%d", confirmations, confirmer.calls)
	}
	warning := snapshot.Lanes[0].CheckoutWarning
	if warning == nil || warning.Severity != "warning" || warning.PullRequestNumber != 32 {
		t.Fatalf("warning = %#v", warning)
	}

	cached := record
	cached.Snapshot = snapshot
	cached.CheckoutConfirmations = confirmations
	next := snapshot
	next.Lanes[0].CheckoutWarning = nil
	confirmations = engine.enrichCheckoutWarnings(
		context.Background(), "scan-2", checkoutRepository("owner/repo"), cached, true, true, &next,
	)
	if len(confirmations) != 1 || confirmer.calls != 1 || next.Lanes[0].CheckoutWarning == nil {
		t.Fatalf("cached confirmations=%#v calls=%d warning=%#v", confirmations, confirmer.calls, next.Lanes[0].CheckoutWarning)
	}
}

func TestCheckoutWarningLimitsConfirmationAndSkipsNonFollowedProjects(t *testing.T) {
	now := time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC)
	confirmer := &recordingCheckoutConfirmer{now: now, status: checkoutwarn.StatusConfirmed}
	engine := NewEngine(config.Config{}, Paths{}, Cache{}, nil, nil, nil, tracking.Manager{})
	engine.Now = func() time.Time { return now }
	engine.CheckoutConfirmer = confirmer

	for index := range 4 {
		projectID := fmt.Sprintf("owner/repo-%d", index)
		record, snapshot := checkoutTransition(projectID, 40+index)
		confirmations := engine.enrichCheckoutWarnings(
			context.Background(), "scan-budget", checkoutRepository(projectID), record, true, true, &snapshot,
		)
		if index < maxCheckoutConfirmationsPerRefresh && snapshot.Lanes[0].CheckoutWarning == nil {
			t.Fatalf("candidate %d was not confirmed: %#v", index, confirmations)
		}
		if index == maxCheckoutConfirmationsPerRefresh && snapshot.Lanes[0].CheckoutWarning != nil {
			t.Fatalf("over-budget candidate warned: %#v", snapshot.Lanes[0].CheckoutWarning)
		}
	}
	if confirmer.calls != maxCheckoutConfirmationsPerRefresh {
		t.Fatalf("confirmation calls = %d", confirmer.calls)
	}

	record, snapshot := checkoutTransition("owner/quiet", 99)
	if confirmations := engine.enrichCheckoutWarnings(
		context.Background(), "scan-quiet", checkoutRepository("owner/quiet"), record, true, false, &snapshot,
	); len(confirmations) != 0 || snapshot.Lanes[0].CheckoutWarning != nil {
		t.Fatalf("non-followed confirmations=%#v warning=%#v", confirmations, snapshot.Lanes[0].CheckoutWarning)
	}

	record, snapshot = checkoutTransition("owner/cold", 100)
	if confirmations := engine.enrichCheckoutWarnings(
		context.Background(), "scan-cold", checkoutRepository("owner/cold"), record, false, true, &snapshot,
	); len(confirmations) != 0 || snapshot.Lanes[0].CheckoutWarning != nil {
		t.Fatalf("cold-cache confirmations=%#v warning=%#v", confirmations, snapshot.Lanes[0].CheckoutWarning)
	}
}

func TestCheckoutWarningMarksDirtyOrAdvancedCheckoutCritical(t *testing.T) {
	now := time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC)
	for name, mutate := range map[string]func(*model.Lane){
		"dirty":    func(lane *model.Lane) { lane.Signals.Worktree = model.WorktreeDirty },
		"advanced": func(lane *model.Lane) { lane.Worktree.HeadOID = "later-local-head" },
	} {
		t.Run(name, func(t *testing.T) {
			_, snapshot := checkoutTransition("owner/repo", 32)
			mutate(&snapshot.Lanes[0])
			attachCheckoutWarnings(&snapshot, []checkoutwarn.Confirmation{{
				PullRequestNumber: 32, Branch: "GH-31", Base: "main", HeadOID: "pr-head",
				Status: checkoutwarn.StatusConfirmed, MergedAt: now, CheckedAt: now,
			}})
			if warning := snapshot.Lanes[0].CheckoutWarning; warning == nil || warning.Severity != "critical" {
				t.Fatalf("warning = %#v", warning)
			}
		})
	}
}

type recordingCheckoutConfirmer struct {
	calls  int
	now    time.Time
	status checkoutwarn.Status
}

func (c *recordingCheckoutConfirmer) Confirm(_ context.Context, _ config.Repository, confirmation checkoutwarn.Confirmation) (checkoutwarn.Confirmation, error) {
	c.calls++
	confirmation.Status = c.status
	confirmation.CheckedAt = c.now
	confirmation.MergedAt = c.now.Add(-time.Hour)
	confirmation.HeadOID = "pr-head"
	return confirmation, nil
}

func checkoutTransition(projectID string, pullRequestNumber int) (ProjectRecord, model.Snapshot) {
	now := time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC)
	repository := checkoutRepository(projectID)
	oldLane := model.Lane{
		ID: fmt.Sprintf("gh:%s#%d", projectID, pullRequestNumber), Repository: repository.Name,
		GitHub: projectID, Base: "main", Branch: "GH-31",
		Worktree: &model.Worktree{Path: repository.Path, HeadOID: "pr-head", UpdatedAt: now},
		PullRequest: &model.PullRequest{
			Number: pullRequestNumber, URL: fmt.Sprintf("https://github.com/%s/pull/%d", projectID, pullRequestNumber),
			HeadRefName: "GH-31", HeadRefOID: "pr-head", BaseRefName: "main", UpdatedAt: now,
		},
		Signals:   model.Signals{Worktree: model.WorktreeClean, PullRequest: model.PullRequestOpen},
		UpdatedAt: now,
	}
	record := ProjectRecord{
		Version: CacheVersion, ProjectID: projectID, Revision: 1, Stage: "ready", UpdatedAt: now,
		Snapshot: model.Snapshot{
			SchemaVersion: model.SchemaVersion,
			Projects:      []model.Project{{Name: repository.Name, Path: repository.Path, GitHub: projectID, Base: "main", Remote: "origin"}},
			Lanes:         []model.Lane{oldLane}, Errors: []model.ScanError{}, Warnings: []model.ScanError{},
		},
	}
	localLane := oldLane
	localLane.ID = "git:" + projectID + "@GH-31"
	localLane.PullRequest = nil
	localLane.Signals.PullRequest = model.PullRequestNone
	snapshot := record.Snapshot
	snapshot.Lanes = []model.Lane{localLane}
	return record, snapshot
}

func checkoutRepository(projectID string) config.Repository {
	return config.Repository{Name: "repo", Path: "/repo", GitHub: projectID, Base: "main", Remote: "origin"}
}
