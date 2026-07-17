package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/notes"
	"github.com/jamesonstone/beacon/internal/reposync"
	"github.com/jamesonstone/beacon/internal/tracking"
	"github.com/jamesonstone/beacon/internal/workset"
)

func TestServerClientCreatesOpensAndClosesSignalNoteTabs(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "beacon-tabs-agent-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	paths := testPaths(root)
	cfg := config.Config{Path: paths.Config, Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Hour, UntrackedProbeInterval: time.Hour}}
	engine := NewEngine(cfg, paths, Cache{Directory: paths.Projects}, func(context.Context) ([]config.Repository, error) {
		return []config.Repository{}, nil
	}, nil, nil, tracking.Manager{})
	server := &Server{Paths: paths, Engine: engine}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	waitForFile(t, paths.Socket)
	client := Client{Socket: paths.Socket, Timeout: time.Second}
	events, eventErrors, err := client.Subscribe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	<-events

	created, err := client.Request(context.Background(), Request{Type: RequestCreateNote, Content: "Idea\n\nBody"})
	if err != nil || created.NotesWorkspace == nil || created.NotesWorkspace.Active == nil {
		t.Fatalf("created event=%#v err=%v", created, err)
	}
	id := created.NotesWorkspace.ActiveID
	if created.NotesWorkspace.Active.Title != "Idea" || created.NotesWorkspace.Active.ID != id {
		t.Fatalf("created workspace=%#v", created.NotesWorkspace)
	}
	assertWorkspaceEvent(t, events, eventErrors, id)

	closed, err := client.Request(context.Background(), Request{Type: RequestCloseNote, NoteID: id})
	if err != nil || closed.NotesWorkspace == nil || closed.NotesWorkspace.ActiveID != notes.GeneralID {
		t.Fatalf("closed event=%#v err=%v", closed, err)
	}
	assertWorkspaceEvent(t, events, eventErrors, notes.GeneralID)

	opened, err := client.Request(context.Background(), Request{Type: RequestOpenNote, NoteID: "Idea"})
	if err != nil || opened.NotesWorkspace == nil || opened.NotesWorkspace.ActiveID != id {
		t.Fatalf("opened event=%#v err=%v", opened, err)
	}
	assertWorkspaceEvent(t, events, eventErrors, id)

	pinned, err := client.Request(context.Background(), Request{
		Type: RequestSetNotePinned, NoteID: id, Pinned: true,
	})
	if err != nil || pinned.NotesWorkspace == nil || len(pinned.NotesWorkspace.PinnedIDs) != 2 ||
		pinned.NotesWorkspace.PinnedIDs[1] != id {
		t.Fatalf("pinned event=%#v err=%v", pinned, err)
	}
	assertWorkspaceEvent(t, events, eventErrors, id)

	blocked, err := client.Request(context.Background(), Request{Type: RequestCloseNote, NoteID: id})
	if err != nil || blocked.Type != EventProjectFailed {
		t.Fatalf("close pinned event=%#v err=%v", blocked, err)
	}

	reordered, err := client.Request(context.Background(), Request{Type: RequestReorderPinned, NoteIDs: []string{id}})
	if err != nil || reordered.NotesWorkspace == nil || reordered.NotesWorkspace.PinnedIDs[1] != id {
		t.Fatalf("reordered event=%#v err=%v", reordered, err)
	}
	assertWorkspaceEvent(t, events, eventErrors, id)

	unpinned, err := client.Request(context.Background(), Request{Type: RequestSetNotePinned, NoteID: id})
	if err != nil || unpinned.NotesWorkspace == nil || len(unpinned.NotesWorkspace.PinnedIDs) != 1 {
		t.Fatalf("unpinned event=%#v err=%v", unpinned, err)
	}
	assertWorkspaceEvent(t, events, eventErrors, id)

	loaded, err := client.Request(context.Background(), Request{Type: RequestGetNotesWorkspace})
	if err != nil || loaded.Type != EventNotesWorkspace || loaded.NotesWorkspace == nil || loaded.NotesWorkspace.ActiveID != id {
		t.Fatalf("loaded event=%#v err=%v", loaded, err)
	}

	deleted, err := client.Request(context.Background(), Request{Type: RequestDeleteNote, NoteID: id})
	if err != nil || deleted.NotesWorkspace == nil || deleted.NotesWorkspace.ActiveID != notes.GeneralID {
		t.Fatalf("deleted event=%#v err=%v", deleted, err)
	}
	for _, tab := range deleted.NotesWorkspace.Tabs {
		if tab.ID == id {
			t.Fatalf("deleted tab remains in workspace: %#v", deleted.NotesWorkspace.Tabs)
		}
	}
	assertWorkspaceEvent(t, events, eventErrors, notes.GeneralID)
	if _, err := client.Request(context.Background(), Request{Type: RequestShutdown}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop")
	}
}

func assertWorkspaceEvent(t *testing.T, events <-chan Event, eventErrors <-chan error, activeID string) {
	t.Helper()
	select {
	case event := <-events:
		if event.Type != EventWorkspaceUpdated || event.NotesWorkspace == nil || event.NotesWorkspace.ActiveID != activeID {
			t.Fatalf("workspace event=%#v", event)
		}
	case err := <-eventErrors:
		t.Fatalf("subscription error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("workspace update was not published")
	}
}

func TestServerClientChecksAndAppliesRepositorySync(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "beacon-sync-agent-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	paths := testPaths(root)
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo", Base: "main", Remote: "origin"}
	cfg := config.Config{Path: paths.Config, Settings: config.Settings{
		MaxParallel: 1, TrackedRefreshInterval: time.Hour, UntrackedProbeInterval: time.Hour,
	}}
	scanProject := func(context.Context, config.Repository, bool, func(string)) (model.Snapshot, error) {
		return model.Snapshot{SchemaVersion: model.SchemaVersion}, nil
	}
	engine := NewEngine(cfg, paths, Cache{Directory: paths.Projects}, func(context.Context) ([]config.Repository, error) {
		return []config.Repository{repository}, nil
	}, scanProject, nil, tracking.Manager{})
	synchronizer := &recordingRepositorySynchronizer{}
	engine.RepositorySync = synchronizer
	server := &Server{Paths: paths, Engine: engine}
	done := make(chan error, 1)
	go func() { done <- server.Serve(context.Background()) }()
	waitForFile(t, paths.Socket)
	client := Client{Socket: paths.Socket}

	checked, err := client.Request(context.Background(), Request{Type: RequestGetRepositorySync, Refresh: true})
	if err != nil || checked.RepositorySync == nil || !synchronizer.refresh {
		t.Fatalf("checked=%#v refresh=%t err=%v", checked, synchronizer.refresh, err)
	}
	applied, err := client.Request(context.Background(), Request{Type: RequestSyncRepositories, ProjectIDs: []string{"owner/repo"}})
	if err != nil || applied.RepositorySync == nil || len(synchronizer.projectIDs) != 1 || synchronizer.projectIDs[0] != "owner/repo" {
		t.Fatalf("applied=%#v project_ids=%#v err=%v", applied, synchronizer.projectIDs, err)
	}

	if _, err := client.Request(context.Background(), Request{Type: RequestShutdown}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

type recordingRepositorySynchronizer struct {
	refresh    bool
	projectIDs []string
}

func (s *recordingRepositorySynchronizer) Check(_ context.Context, repositories []config.Repository, refresh bool) reposync.Report {
	s.refresh = refresh
	return reposync.Report{Repositories: []reposync.Repository{{ProjectID: repositories[0].GitHub}}}
}

func (s *recordingRepositorySynchronizer) Apply(_ context.Context, _ []config.Repository, projectIDs []string) reposync.Report {
	s.projectIDs = append([]string(nil), projectIDs...)
	return reposync.Report{Repositories: []reposync.Repository{{ProjectID: projectIDs[0], Updated: true}}}
}

func TestServerClientMutatesWorkingSetThroughSharedAuthority(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "beacon-workset-agent-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	paths := testPaths(root)
	cfg := config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Hour, UntrackedProbeInterval: time.Hour}}
	engine := NewEngine(cfg, paths, Cache{Directory: paths.Projects}, func(context.Context) ([]config.Repository, error) {
		return []config.Repository{}, nil
	}, nil, nil, tracking.Manager{})
	engine.WorkingSet = &workset.Manager{Store: workset.FileStore{}}
	server := &Server{Paths: paths, Engine: engine}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(context.Background()) }()
	waitForFile(t, paths.Socket)
	client := Client{Socket: paths.Socket}

	added, err := client.Request(context.Background(), Request{Type: RequestAddManualLane, Title: "Plan migration"})
	if err != nil || added.Snapshot == nil || len(added.Snapshot.WorkingSet.Active) != 1 {
		t.Fatalf("add event=%#v err=%v", added, err)
	}
	laneID := added.Snapshot.WorkingSet.Active[0]
	noted, err := client.Request(context.Background(), Request{Type: RequestSetLaneNote, LaneID: laneID, Note: "compare storage contracts"})
	if err != nil || noted.Snapshot == nil || noted.Snapshot.Lanes[0].Attention.Note != "compare storage contracts" {
		t.Fatalf("note event=%#v err=%v", noted, err)
	}
	tagged, err := client.Request(context.Background(), Request{Type: RequestAddLaneTag, LaneID: laneID, Tag: "manual test"})
	if err != nil || tagged.Snapshot == nil || len(tagged.Snapshot.Lanes[0].Attention.Tags) != 1 {
		t.Fatalf("tag event=%#v err=%v", tagged, err)
	}
	untagged, err := client.Request(context.Background(), Request{Type: RequestRemoveLaneTag, LaneID: laneID, Tag: "manual test"})
	if err != nil || untagged.Snapshot == nil || len(untagged.Snapshot.Lanes[0].Attention.Tags) != 0 {
		t.Fatalf("untag event=%#v err=%v", untagged, err)
	}
	parked, err := client.Request(context.Background(), Request{Type: RequestSetLaneAttention, LaneID: laneID, AttentionState: string(model.AttentionParked)})
	if err != nil || parked.Snapshot == nil || len(parked.Snapshot.WorkingSet.Parked) != 1 {
		t.Fatalf("park event=%#v err=%v", parked, err)
	}

	if _, err := client.Request(context.Background(), Request{Type: RequestShutdown}); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}
