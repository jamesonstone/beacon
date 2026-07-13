package workset

import (
	"fmt"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestLaneTagsAreNormalizedAndRemovable(t *testing.T) {
	manager, now := testManager(t)
	snapshot := testSnapshot(now, testLane("lane", "owner/repo", "feature", model.WorktreeDirty, model.PublicationPublished, now))
	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.AddTag(updated, "lane", "  manual test  ")
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.AddTag(updated, "lane", "MANUAL TEST")
	if err != nil {
		t.Fatal(err)
	}
	if got := updated.Lanes[0].Attention.Tags; len(got) != 1 || got[0] != "manual test" {
		t.Fatalf("tags = %#v", got)
	}
	updated, err = manager.RemoveTag(updated, "lane", "Manual Test")
	if err != nil {
		t.Fatal(err)
	}
	if got := updated.Lanes[0].Attention.Tags; len(got) != 0 {
		t.Fatalf("tags after removal = %#v", got)
	}
}

func TestLaneTagLimitsAreEnforced(t *testing.T) {
	manager, now := testManager(t)
	snapshot := testSnapshot(now, testLane("lane", "owner/repo", "feature", model.WorktreeDirty, model.PublicationPublished, now))
	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 12; index++ {
		updated, err = manager.AddTag(updated, "lane", fmt.Sprintf("tag-%d", index))
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := manager.AddTag(updated, "lane", "one-too-many"); err == nil {
		t.Fatal("expected tag-count validation error")
	}
}

func TestLaneTagsDoNotCreateAttentionOrChangePolicy(t *testing.T) {
	manager, now := testManager(t)
	lane := testLane("inactive", "owner/repo", "old", model.WorktreeClean, model.PublicationPublished, now.Add(-30*24*time.Hour))
	snapshot := testSnapshot(now, lane)
	updated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	updated, err = manager.AddTag(updated, lane.ID, "remember")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Lanes[0].Attention != nil || len(updated.WorkingSet.Active)+len(updated.WorkingSet.Waiting)+len(updated.WorkingSet.Recent) != 0 {
		t.Fatalf("tag created attention: lane=%#v groups=%#v", updated.Lanes[0], updated.WorkingSet)
	}
	state, err := (FileStore{}).Load(updated.WorkingSet.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 1 || len(state.Entries[0].Tags) != 1 || state.Entries[0].Tags[0] != "remember" {
		t.Fatalf("stored tags = %#v", state.Entries)
	}
}
