package workset

import (
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestReconcileKeepsOpenPullRequestsInFollowedWorkingSet(t *testing.T) {
	manager, now := testManager(t)
	snapshot := testSnapshot(now,
		testLane("dirty", "owner/repo", "feature-a", model.WorktreeDirty, model.PublicationPublished, now.Add(-time.Hour)),
		testLane("old-pr", "owner/repo", "feature-b", model.WorktreeClean, model.PublicationPublished, now.Add(-10*24*time.Hour)),
		testLane("stale-dirty-pr", "owner/repo", "feature-c", model.WorktreeDirty, model.PublicationPublished, now.Add(-10*24*time.Hour)),
	)
	snapshot.Lanes[1].PullRequest = &model.PullRequest{Number: 2, UpdatedAt: now.Add(-10 * 24 * time.Hour)}
	snapshot.Lanes[2].PullRequest = &model.PullRequest{Number: 3, UpdatedAt: now.Add(-10 * 24 * time.Hour)}
	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 3 || updated.WorkingSet.Active[0] != "dirty" || updated.WorkingSet.Active[1] != "old-pr" || updated.WorkingSet.Active[2] != "stale-dirty-pr" || len(updated.WorkingSet.Recent) != 0 {
		t.Fatalf("working set = %#v", updated.WorkingSet)
	}
	if updated.Lanes[1].Attention == nil || updated.Lanes[1].Attention.State != model.AttentionActive {
		t.Fatalf("open PR attention = %#v", updated.Lanes[1].Attention)
	}
	updated, err = manager.SetPinned(updated, "old-pr", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 3 || updated.Lanes[1].Attention == nil || !updated.Lanes[1].Attention.Pinned {
		t.Fatalf("pinned working set = %#v", updated.WorkingSet)
	}
	updated, err = manager.SetPinned(updated, "old-pr", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 3 || updated.Lanes[1].Attention == nil || updated.Lanes[1].Attention.Pinned {
		t.Fatalf("unpinned open PR left working set = %#v", updated.WorkingSet)
	}
	updated.Lanes[1].PullRequest = nil
	updated.Lanes[1].Signals.PullRequest = model.PullRequestNone
	updated, err = manager.Reconcile(updated)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 2 || updated.WorkingSet.Active[0] != "dirty" || updated.WorkingSet.Active[1] != "stale-dirty-pr" || updated.Lanes[1].Attention != nil {
		t.Fatalf("closed PR remained in working set = %#v", updated.WorkingSet)
	}
}

func TestReconcileRepairsAutomaticallyParkedOpenPullRequest(t *testing.T) {
	manager, now := testManager(t)
	lane := testLane(
		"stale-dirty-pr", "owner/repo", "feature", model.WorktreeDirty,
		model.PublicationPublished, now.Add(-10*24*time.Hour),
	)
	lane.PullRequest = &model.PullRequest{Number: 2, UpdatedAt: now.Add(-10 * 24 * time.Hour)}
	observation := observe(lane, now)
	path, err := ResolvePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := (FileStore{}).Write(path, State{
		Version: Version, Migrated: true,
		Entries: []Entry{{
			ID: lane.ID, Repository: lane.Repository, GitHub: lane.GitHub,
			Branch: lane.Branch, State: model.AttentionParked,
			Previous: observation, Current: observation,
		}},
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	updated, err := manager.Reconcile(testSnapshot(now, lane))
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 1 || updated.WorkingSet.Active[0] != lane.ID || len(updated.WorkingSet.Parked) != 0 {
		t.Fatalf("automatically parked PR was not repaired: %#v", updated.WorkingSet)
	}

	updated, err = manager.SetAttention(updated, lane.ID, model.AttentionParked)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 0 || len(updated.WorkingSet.Parked) != 1 || updated.WorkingSet.Parked[0] != lane.ID {
		t.Fatalf("explicitly parked PR was reactivated: %#v", updated.WorkingSet)
	}
}

func TestReconcileKeepsOpenIssuesInFollowedWorkingSet(t *testing.T) {
	manager, now := testManager(t)
	lane := testLane(
		"old-issue", "owner/repo", "", model.WorktreeNotLocal,
		model.PublicationPublished, now.Add(-10*24*time.Hour),
	)
	lane.Worktree = nil
	lane.Issue = &model.Issue{Number: 50, UpdatedAt: now.Add(-10 * 24 * time.Hour)}
	lane.Signals.Issue = model.IssueOpen
	snapshot := testSnapshot(now, lane)

	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 1 || updated.WorkingSet.Active[0] != lane.ID || len(updated.WorkingSet.Recent) != 0 {
		t.Fatalf("working set = %#v", updated.WorkingSet)
	}
	if updated.Lanes[0].Attention == nil || updated.Lanes[0].Attention.State != model.AttentionActive {
		t.Fatalf("open issue attention = %#v", updated.Lanes[0].Attention)
	}

	updated.Lanes[0].Issue = nil
	updated.Lanes[0].Signals.Issue = model.IssueNone
	updated, err = manager.Reconcile(updated)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.WorkingSet.Active) != 0 || updated.Lanes[0].Attention != nil {
		t.Fatalf("closed issue remained in working set = %#v", updated.WorkingSet)
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
