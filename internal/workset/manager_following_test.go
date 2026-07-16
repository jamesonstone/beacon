package workset

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestOutsideProjectLanesStayOutOfFollowingWithoutLosingDurableState(t *testing.T) {
	manager, now := testManager(t)
	lane := testLane("lane", "owner/repo", "feature", model.WorktreeDirty, model.PublicationPublished, now)
	snapshot := testSnapshot(now, lane)

	followed, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(followed.WorkingSet.Active) != 1 {
		t.Fatalf("followed working set = %#v", followed.WorkingSet)
	}

	snapshot.Projects[0].TrackingState = model.TrackingUntracked
	snapshot.Projects[0].FollowState = model.FollowQuiet
	outside, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(outside.WorkingSet.Active) != 0 || outside.Lanes[0].Attention != nil {
		t.Fatalf("outside lane remained in Following = %#v", outside.WorkingSet)
	}
	state, err := (FileStore{}).Load(outside.WorkingSet.Path)
	if err != nil || len(state.Entries) != 1 || state.Entries[0].ID != lane.ID {
		t.Fatalf("durable lane state = %#v, err = %v", state, err)
	}

	snapshot.Projects[0].TrackingState = model.TrackingTracked
	snapshot.Projects[0].FollowState = model.FollowFollowing
	restored, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.WorkingSet.Active) != 1 || restored.WorkingSet.Active[0] != lane.ID {
		t.Fatalf("restored working set = %#v", restored.WorkingSet)
	}
}

func TestPinnedInactiveRemotePullRequestSurvivesDefaultEnrichmentFilter(t *testing.T) {
	manager, now := testManager(t)
	lane := testLane("gh:owner/repo#2", "owner/repo", "old-pr", model.WorktreeNotLocal, model.PublicationPublished, now.Add(-30*24*time.Hour))
	lane.Worktree = nil
	lane.PullRequest = &model.PullRequest{Number: 2, UpdatedAt: now.Add(-30 * 24 * time.Hour)}
	snapshot := testSnapshot(now, lane)
	updated, err := manager.SetPinned(snapshot, lane.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	updated.Lanes = nil
	updated.Projects[0].LaneIDs = nil
	updated, err = manager.Reconcile(updated)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 1 || len(updated.Lanes) != 1 || updated.Lanes[0].PullRequest == nil {
		t.Fatalf("retained snapshot = %#v", updated)
	}
	if got := updated.Lanes[0].PullRequest.URL; got != "https://github.com/owner/repo/pull/2" {
		t.Fatalf("retained PR URL = %q", got)
	}
}

func TestStaleDirtyLaneStartsParkedAndCleanBaseStaysOut(t *testing.T) {
	manager, now := testManager(t)
	staleDirty := testLane("stale-dirty", "owner/repo", "old", model.WorktreeDirty, model.PublicationPublished, now.Add(-30*24*time.Hour))
	cleanBase := testLane("base", "owner/repo", "main", model.WorktreeClean, model.PublicationBase, now.Add(-time.Hour))
	cleanBase.Base = "main"
	updated, err := manager.Reconcile(testSnapshot(now, staleDirty, cleanBase))
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Parked) != 1 || updated.WorkingSet.Parked[0] != staleDirty.ID {
		t.Fatalf("working set = %#v", updated.WorkingSet)
	}
	if updated.Lanes[1].Attention != nil {
		t.Fatalf("clean base entered focus: %#v", updated.Lanes[1].Attention)
	}
}

func testManager(t *testing.T) (Manager, time.Time) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	now := time.Date(2026, 7, 12, 20, 0, 0, 0, time.UTC)
	return Manager{Store: FileStore{}, Now: func() time.Time { return now }}, now
}

func testSnapshot(now time.Time, lanes ...model.Lane) model.Snapshot {
	return model.Snapshot{SchemaVersion: model.SchemaVersion, GeneratedAt: now, Projects: []model.Project{{Name: "repo", GitHub: "owner/repo", TrackingState: model.TrackingTracked, LaneIDs: laneIDs(lanes), Errors: []model.ScanError{}, Warnings: []model.ScanError{}}}, Lanes: lanes, Errors: []model.ScanError{}, Warnings: []model.ScanError{}}
}

func laneIDs(lanes []model.Lane) []string {
	values := make([]string, len(lanes))
	for index := range lanes {
		values[index] = lanes[index].ID
	}
	return values
}

func testLane(id, github, branch string, worktree model.WorktreeState, publication model.PublicationState, updated time.Time) model.Lane {
	return model.Lane{ID: id, Repository: "repo", GitHub: github, Branch: branch, Worktree: &model.Worktree{Path: "/repo/" + branch, HeadOID: "head", StatusHash: "status", UpdatedAt: updated}, Signals: model.Signals{Worktree: worktree, Publication: publication, PullRequest: model.PullRequestNone, CI: model.CINone, Review: model.ReviewNone, Merge: model.MergeClean, Freshness: model.FreshnessCurrent, Issue: model.IssueNone}, NextAction: model.ActionInspectLocal, UpdatedAt: updated, Reasons: []string{}, Warnings: []string{}, Blockers: []string{}}
}
