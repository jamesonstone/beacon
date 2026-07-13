package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
	"github.com/jamesonstone/beacon/internal/workset"
)

func TestSchedulerBoundsConcurrencyAndCoalescesDuplicates(t *testing.T) {
	var active atomic.Int32
	var maximum atomic.Int32
	var calls atomic.Int32
	Scheduler{MaxParallel: 2}.Run(context.Background(), []string{"a", "b", "a", "c"}, func(context.Context, string) {
		current := active.Add(1)
		for {
			previous := maximum.Load()
			if current <= previous || maximum.CompareAndSwap(previous, current) {
				break
			}
		}
		calls.Add(1)
		time.Sleep(15 * time.Millisecond)
		active.Add(-1)
	})
	if calls.Load() != 3 || maximum.Load() > 2 {
		t.Fatalf("calls=%d maximum=%d", calls.Load(), maximum.Load())
	}
}

func TestProtocolRejectsUnsupportedAndMalformedRequests(t *testing.T) {
	valid := []byte(`{"protocol_version":1,"request_id":"one","type":"get_snapshot"}` + "\n")
	request, err := DecodeRequest(bytes.NewReader(valid))
	if err != nil || request.Type != RequestGetSnapshot {
		t.Fatalf("request=%#v err=%v", request, err)
	}
	for _, payload := range []string{
		`{"protocol_version":2,"request_id":"one","type":"get_snapshot"}`,
		`{"protocol_version":1,"request_id":"","type":"get_snapshot"}`,
		`{"protocol_version":1,"request_id":"one","type":"get_snapshot","unknown":true}`,
	} {
		if _, err := DecodeRequest(bytes.NewBufferString(payload)); err == nil {
			t.Fatalf("accepted invalid request %s", payload)
		}
	}
}

func TestSlowSubscriberRetainsTerminalSnapshotEvent(t *testing.T) {
	hub := newEventHub()
	events, unsubscribe := hub.Subscribe()
	defer unsubscribe()
	for index := 0; index < 80; index++ {
		hub.Publish(Event{ProtocolVersion: ProtocolVersion, Type: EventProjectQueued, ProjectID: fmt.Sprintf("project-%d", index)})
	}
	snapshot := model.Snapshot{SchemaVersion: model.SchemaVersion}
	hub.Publish(Event{ProtocolVersion: ProtocolVersion, Type: EventScanCompleted, ScanID: "scan", Snapshot: &snapshot})
	for index := 0; index < 64; index++ {
		event := <-events
		if event.Type == EventScanCompleted && event.ScanID == "scan" && event.Snapshot != nil {
			return
		}
	}
	t.Fatal("terminal snapshot event was dropped for a slow subscriber")
}

func TestServerClientSnapshotStatusAndShutdown(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "beacon-agent-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	paths := testPaths(root)
	cfg := config.Config{Path: filepath.Join(root, "config.yaml"), Settings: config.Settings{MaxParallel: 1, TrackedRefreshInterval: time.Hour, UntrackedProbeInterval: time.Hour}}
	engine := NewEngine(cfg, paths, Cache{Directory: paths.Projects}, func(context.Context) ([]config.Repository, error) {
		return []config.Repository{}, nil
	}, nil, nil, tracking.Manager{})
	server := &Server{Paths: paths, Engine: engine}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(context.Background()) }()
	waitForFile(t, paths.Socket)
	client := Client{Socket: paths.Socket}
	event, err := client.Request(context.Background(), Request{Type: RequestGetSnapshot})
	if err != nil || event.Snapshot == nil || event.Snapshot.SchemaVersion != model.SchemaVersion {
		t.Fatalf("snapshot event=%#v err=%v", event, err)
	}
	statusEvent, err := client.Request(context.Background(), Request{Type: RequestGetAgentStatus})
	if err != nil || statusEvent.Status == nil || !statusEvent.Status.Running {
		t.Fatalf("status event=%#v err=%v", statusEvent, err)
	}
	if _, err := client.Request(context.Background(), Request{Type: RequestShutdown}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-serverDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop")
	}
}

func TestServerClientReadsWritesAndPublishesSignalNotes(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "beacon-notes-agent-")
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
	done := make(chan error, 1)
	go func() { done <- server.Serve(context.Background()) }()
	waitForFile(t, paths.Socket)
	client := Client{Socket: paths.Socket}
	events, eventErrors, err := client.Subscribe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	<-events // initial snapshot

	written, err := client.Request(context.Background(), Request{Type: RequestSetNotes, Content: "# Signal Log\n\nShip it."})
	if err != nil || written.Notes == nil || written.Notes.Content != "# Signal Log\n\nShip it." {
		t.Fatalf("written event=%#v err=%v", written, err)
	}
	select {
	case event := <-events:
		if event.Type != EventNotesUpdated || event.Notes == nil || event.Notes.Content != written.Notes.Content {
			t.Fatalf("published event=%#v", event)
		}
	case err := <-eventErrors:
		t.Fatalf("subscription error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("notes update was not published")
	}
	appended, err := client.Request(context.Background(), Request{Type: RequestAppendNotes, Content: "- verify orbit"})
	if err != nil || appended.Notes == nil || appended.Notes.Content != "# Signal Log\n\nShip it.\n- verify orbit\n" {
		t.Fatalf("appended event=%#v err=%v", appended, err)
	}
	select {
	case event := <-events:
		if event.Type != EventNotesUpdated || event.Notes == nil || event.Notes.Content != appended.Notes.Content {
			t.Fatalf("published append event=%#v", event)
		}
	case err := <-eventErrors:
		t.Fatalf("subscription error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("notes append was not published")
	}
	loaded, err := client.Request(context.Background(), Request{Type: RequestGetNotes})
	if err != nil || loaded.Notes == nil || loaded.Notes.Path != paths.Notes || loaded.Notes.UpdatedAt.IsZero() {
		t.Fatalf("loaded event=%#v err=%v", loaded, err)
	}

	if _, err := client.Request(context.Background(), Request{Type: RequestShutdown}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
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
