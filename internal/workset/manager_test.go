package workset

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestReconcileBuildsSmallLaneWorkingSet(t *testing.T) {
	manager, now := testManager(t)
	snapshot := testSnapshot(now,
		testLane("dirty", "owner/repo", "feature-a", model.WorktreeDirty, model.PublicationPublished, now.Add(-time.Hour)),
		testLane("old-pr", "owner/repo", "feature-b", model.WorktreeClean, model.PublicationPublished, now.Add(-10*24*time.Hour)),
	)
	snapshot.Lanes[1].PullRequest = &model.PullRequest{Number: 2, UpdatedAt: now.Add(-10 * 24 * time.Hour)}
	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 1 || updated.WorkingSet.Active[0] != "dirty" || len(updated.WorkingSet.Recent) != 0 {
		t.Fatalf("working set = %#v", updated.WorkingSet)
	}
	if updated.Lanes[1].Attention != nil {
		t.Fatal("inactive PR entered working set")
	}
	updated, err = manager.SetPinned(updated, "old-pr", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 2 || updated.Lanes[1].Attention == nil || !updated.Lanes[1].Attention.Pinned {
		t.Fatalf("pinned working set = %#v", updated.WorkingSet)
	}
	updated, err = manager.SetPinned(updated, "old-pr", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 1 || updated.Lanes[1].Attention != nil {
		t.Fatalf("unpinned inactive PR remained in working set = %#v", updated.WorkingSet)
	}
}

func TestParkedLaneIgnoresUnrelatedChangeAndReactivatesForOwnDelta(t *testing.T) {
	manager, now := testManager(t)
	snapshot := testSnapshot(now,
		testLane("one", "owner/repo", "one", model.WorktreeDirty, model.PublicationPublished, now),
		testLane("two", "owner/repo", "two", model.WorktreeDirty, model.PublicationPublished, now),
	)
	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.SetAttention(updated, "one", model.AttentionParked)
	if err != nil {
		t.Fatal(err)
	}
	updated.Lanes[1].Worktree.HeadOID = "other-change"
	updated, err = manager.Reconcile(updated)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Lanes[0].Attention.State != model.AttentionParked {
		t.Fatalf("unrelated change resumed lane: %#v", updated.Lanes[0].Attention)
	}
	updated.Lanes[0].Worktree.HeadOID = "own-change"
	updated, err = manager.Reconcile(updated)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Lanes[0].Attention.State != model.AttentionActive || updated.Lanes[0].Attention.ReactivationReason == "" {
		t.Fatalf("own change did not reactivate: %#v", updated.Lanes[0].Attention)
	}
}

func TestManualLaneNoteSeenAndPark(t *testing.T) {
	manager, now := testManager(t)
	snapshot := testSnapshot(now)
	updated, id, err := manager.AddManual(snapshot, "Plan launch")
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.SetNote(updated, id, "check the launch checklist")
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.MarkSeen(updated, id)
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.SetAttention(updated, id, model.AttentionParked)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Parked) != 1 || updated.WorkingSet.Parked[0] != id {
		t.Fatalf("parked = %#v", updated.WorkingSet)
	}
	updated, err = manager.SetAttention(updated, id, model.AttentionActive)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 1 || updated.WorkingSet.Active[0] != id {
		t.Fatalf("resumed = %#v", updated.WorkingSet)
	}
	var manual *model.Lane
	for index := range updated.Lanes {
		if updated.Lanes[index].ID == id {
			manual = &updated.Lanes[index]
		}
	}
	if manual == nil || manual.Attention.Note != "check the launch checklist" || manual.Attention.LastSeenAt.IsZero() || manual.NextAction != model.ActionContinueWork {
		t.Fatalf("manual lane = %#v", manual)
	}
}

func TestReconcileMigratesUntrackedProjectLanesToParked(t *testing.T) {
	manager, now := testManager(t)
	lane := testLane("old-lane", "owner/repo", "old", model.WorktreeClean, model.PublicationPublished, now.Add(-30*24*time.Hour))
	snapshot := testSnapshot(now, lane)
	snapshot.Projects[0].TrackingState = model.TrackingUntracked

	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Parked) != 0 || updated.Lanes[0].Attention != nil {
		t.Fatalf("outside project leaked into working set = %#v", updated.WorkingSet)
	}
	state, err := (FileStore{}).Load(updated.WorkingSet.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Migrated || len(state.Entries) != 1 || state.Entries[0].State != model.AttentionParked {
		t.Fatalf("migrated state = %#v", state)
	}
}

func TestNoteBecomesStaleWhenEvidenceChangesAfterNote(t *testing.T) {
	manager, now := testManager(t)
	snapshot := testSnapshot(now, testLane("lane", "owner/repo", "feature", model.WorktreeDirty, model.PublicationPublished, now))
	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.SetNote(updated, "lane", "finish validation")
	if err != nil {
		t.Fatal(err)
	}
	manager.Now = func() time.Time { return now.Add(time.Hour) }
	updated.Lanes[0].Worktree.HeadOID = "new-head"
	updated, err = manager.Reconcile(updated)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Lanes[0].Attention.NoteStale || updated.Lanes[0].Attention.Delta != "new commit observed" {
		t.Fatalf("attention = %#v", updated.Lanes[0].Attention)
	}
}

func TestExplicitResumeKeepsInactiveLaneInWorkingSet(t *testing.T) {
	manager, now := testManager(t)
	lane := testLane("old", "owner/repo", "old", model.WorktreeClean, model.PublicationPublished, now.Add(-30*24*time.Hour))
	snapshot := testSnapshot(now, lane)
	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.SetAttention(updated, lane.ID, model.AttentionActive)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 1 || updated.WorkingSet.Active[0] != lane.ID {
		t.Fatalf("resumed working set = %#v", updated.WorkingSet)
	}
	frequent, err := manager.FrequentRepositories()
	if err != nil {
		t.Fatal(err)
	}
	if _, found := frequent[lane.GitHub]; !found {
		t.Fatalf("resumed repository not scheduled frequently: %#v", frequent)
	}
}

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
