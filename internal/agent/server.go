package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/notes"
	"github.com/jamesonstone/beacon/internal/reposync"
)

type Server struct {
	Paths  Paths
	Engine *Engine
	Notes  notes.WorkspaceStore

	mutex  sync.Mutex
	cancel context.CancelFunc
}

func (s *Server) Serve(ctx context.Context) error {
	if s.Engine == nil {
		return errors.New("agent engine is required")
	}
	if err := s.Paths.EnsureRuntime(); err != nil {
		return err
	}
	release, err := acquirePIDLock(s.Paths.PID)
	if err != nil {
		return err
	}
	defer release()
	_ = os.Remove(s.Paths.Socket)
	listener, err := net.Listen("unix", s.Paths.Socket)
	if err != nil {
		return fmt.Errorf("listen on agent socket %s: %w", s.Paths.Socket, err)
	}
	defer listener.Close()
	defer os.Remove(s.Paths.Socket)
	if err := os.Chmod(s.Paths.Socket, 0o600); err != nil {
		return fmt.Errorf("secure agent socket: %w", err)
	}
	serverContext, cancel := context.WithCancel(ctx)
	s.mutex.Lock()
	s.cancel = cancel
	s.mutex.Unlock()
	defer cancel()
	go func() {
		<-serverContext.Done()
		_ = listener.Close()
	}()
	go s.Engine.RunSchedule(serverContext)
	go s.heartbeats(serverContext)

	for {
		connection, err := listener.Accept()
		if err != nil {
			if serverContext.Err() != nil {
				return nil
			}
			return fmt.Errorf("accept agent connection: %w", err)
		}
		go s.handle(serverContext, connection)
	}
}

func (s *Server) Stop() {
	s.mutex.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mutex.Unlock()
}

func (s *Server) handle(ctx context.Context, connection net.Conn) {
	defer connection.Close()
	request, err := DecodeRequest(connection)
	if err != nil {
		_ = Encode(connection, Event{ProtocolVersion: ProtocolVersion, Type: EventProjectFailed, GeneratedAt: time.Now().UTC(), Message: err.Error()})
		return
	}
	response := func(event Event) {
		event.ProtocolVersion = ProtocolVersion
		event.RequestID = request.RequestID
		if event.GeneratedAt.IsZero() {
			event.GeneratedAt = time.Now().UTC()
		}
		_ = Encode(connection, event)
	}
	switch request.Type {
	case RequestGetSnapshot:
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventSnapshot, Snapshot: &snapshot, Projects: s.Engine.Projects()})
	case RequestListProjects:
		response(Event{Type: EventSnapshot, Projects: s.Engine.Projects()})
	case RequestGetAgentStatus:
		status := s.Engine.Status()
		response(Event{Type: EventAgentStatus, Status: &status})
	case RequestGetNotes:
		document, notesErr := s.notesStore().LoadNote(s.Paths.Notes, request.NoteID)
		if notesErr != nil {
			response(Event{Type: EventProjectFailed, Stage: "notes", Message: notesErr.Error()})
			return
		}
		response(Event{Type: EventNotes, Notes: &document})
	case RequestGetNotesWorkspace:
		workspace, notesErr := s.notesStore().LoadWorkspace(s.Paths.Notes)
		if notesErr != nil {
			response(Event{Type: EventProjectFailed, Stage: "notes", Message: notesErr.Error()})
			return
		}
		response(Event{Type: EventNotesWorkspace, NotesWorkspace: &workspace, Notes: workspace.Active})
	case RequestSetNotes, RequestAppendNotes:
		store := s.notesStore()
		selected, notesErr := store.LoadNote(s.Paths.Notes, request.NoteID)
		if notesErr != nil {
			response(Event{Type: EventProjectFailed, Stage: "notes", Message: notesErr.Error()})
			return
		}
		var workspace notes.Workspace
		if request.Type == RequestAppendNotes {
			workspace, notesErr = store.AppendNote(s.Paths.Notes, selected.ID, request.Content)
		} else {
			workspace, notesErr = store.WriteNote(s.Paths.Notes, selected.ID, request.Content)
		}
		if notesErr != nil {
			response(Event{Type: EventProjectFailed, Stage: "notes", Message: notesErr.Error()})
			return
		}
		document, notesErr := store.LoadNote(s.Paths.Notes, selected.ID)
		if notesErr != nil {
			response(Event{Type: EventProjectFailed, Stage: "notes", Message: notesErr.Error()})
			return
		}
		event := Event{
			ProtocolVersion: ProtocolVersion, Type: EventNotesUpdated,
			GeneratedAt: time.Now().UTC(), Notes: &document, NotesWorkspace: &workspace,
		}
		s.Engine.hub.Publish(event)
		response(event)
	case RequestCreateNote, RequestOpenNote, RequestCloseNote, RequestDeleteNote:
		store := s.notesStore()
		var workspace notes.Workspace
		var notesErr error
		switch request.Type {
		case RequestCreateNote:
			workspace, notesErr = store.CreateNote(s.Paths.Notes, request.Content)
		case RequestOpenNote:
			workspace, notesErr = store.OpenNote(s.Paths.Notes, request.NoteID)
		case RequestCloseNote:
			workspace, notesErr = store.CloseNote(s.Paths.Notes, request.NoteID)
		case RequestDeleteNote:
			workspace, notesErr = store.DeleteNote(s.Paths.Notes, request.NoteID)
		}
		if notesErr != nil {
			response(Event{Type: EventProjectFailed, Stage: "notes", Message: notesErr.Error()})
			return
		}
		event := Event{
			ProtocolVersion: ProtocolVersion, Type: EventWorkspaceUpdated,
			GeneratedAt: time.Now().UTC(), NotesWorkspace: &workspace, Notes: workspace.Active,
		}
		s.Engine.hub.Publish(event)
		response(event)
	case RequestGetRepositorySync, RequestSyncRepositories:
		if s.Engine.RepositorySync == nil {
			response(Event{Type: EventProjectFailed, Stage: "repository-sync", Message: "repository sync is unavailable"})
			return
		}
		repositories, repositoryErr := s.Engine.Repositories(ctx)
		if repositoryErr != nil {
			response(Event{Type: EventProjectFailed, Stage: "repository-sync", Message: repositoryErr.Error()})
			return
		}
		var report reposync.Report
		if request.Type == RequestSyncRepositories {
			report = s.Engine.RepositorySync.Apply(ctx, repositories, request.ProjectIDs)
		} else {
			report = s.Engine.RepositorySync.Check(ctx, repositories, request.Refresh)
		}
		response(Event{Type: EventRepositorySync, Stage: "ready", RepositorySync: &report})
	case RequestRefreshAll, RequestRefreshProject:
		project := request.ProjectID
		if request.Type == RequestRefreshAll {
			project = ""
		}
		scanID, refreshErr := s.Engine.Refresh(ctx, project, true)
		if refreshErr != nil {
			response(Event{Type: EventProjectFailed, ProjectID: project, Stage: "failed", Message: refreshErr.Error()})
			return
		}
		response(Event{Type: EventProjectQueued, ScanID: scanID, ProjectID: project, Stage: "queued"})
	case RequestSetTrackingState:
		if err := s.Engine.SetTracking(ctx, request.ProjectID, request.TrackingState); err != nil {
			response(Event{Type: EventProjectFailed, ProjectID: request.ProjectID, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventTrackingChanged, ProjectID: request.ProjectID, Stage: "ready", Snapshot: &snapshot})
	case RequestSetTrackingBatch:
		if err := s.Engine.SetTrackingBatch(request.ProjectIDs, request.TrackingState); err != nil {
			response(Event{Type: EventProjectFailed, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventTrackingChanged, Stage: "ready", Snapshot: &snapshot})
	case RequestSetSelection:
		if err := s.Engine.SetSelection(request.ProjectIDs); err != nil {
			response(Event{Type: EventProjectFailed, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventTrackingChanged, Stage: "ready", Snapshot: &snapshot})
	case RequestSetLaneAttention:
		if err := s.Engine.SetLaneAttention(request.LaneID, model.AttentionState(request.AttentionState)); err != nil {
			response(Event{Type: EventProjectFailed, ProjectID: request.LaneID, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventWorkingSetChanged, ProjectID: request.LaneID, Stage: "ready", Snapshot: &snapshot})
	case RequestSetLanePinned:
		if err := s.Engine.SetLanePinned(request.LaneID, request.Pinned); err != nil {
			response(Event{Type: EventProjectFailed, ProjectID: request.LaneID, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventWorkingSetChanged, ProjectID: request.LaneID, Stage: "ready", Snapshot: &snapshot})
	case RequestSetLaneNote:
		if err := s.Engine.SetLaneNote(request.LaneID, request.Note); err != nil {
			response(Event{Type: EventProjectFailed, ProjectID: request.LaneID, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventWorkingSetChanged, ProjectID: request.LaneID, Stage: "ready", Snapshot: &snapshot})
	case RequestAddLaneTag:
		if err := s.Engine.AddLaneTag(request.LaneID, request.Tag); err != nil {
			response(Event{Type: EventProjectFailed, ProjectID: request.LaneID, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventWorkingSetChanged, ProjectID: request.LaneID, Stage: "ready", Snapshot: &snapshot})
	case RequestRemoveLaneTag:
		if err := s.Engine.RemoveLaneTag(request.LaneID, request.Tag); err != nil {
			response(Event{Type: EventProjectFailed, ProjectID: request.LaneID, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventWorkingSetChanged, ProjectID: request.LaneID, Stage: "ready", Snapshot: &snapshot})
	case RequestMarkLaneSeen:
		if err := s.Engine.MarkLaneSeen(request.LaneID); err != nil {
			response(Event{Type: EventProjectFailed, ProjectID: request.LaneID, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventWorkingSetChanged, ProjectID: request.LaneID, Stage: "ready", Snapshot: &snapshot})
	case RequestAddManualLane:
		id, err := s.Engine.AddManualLane(request.Title)
		if err != nil {
			response(Event{Type: EventProjectFailed, Stage: "failed", Message: err.Error()})
			return
		}
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventWorkingSetChanged, ProjectID: id, Stage: "ready", Snapshot: &snapshot})
	case RequestSubscribe:
		events, unsubscribe := s.Engine.Subscribe()
		defer unsubscribe()
		snapshot := s.Engine.Snapshot()
		response(Event{Type: EventSnapshot, Snapshot: &snapshot, Projects: s.Engine.Projects()})
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if err := Encode(connection, event); err != nil {
					return
				}
			}
		}
	case RequestShutdown:
		response(Event{Type: EventAgentStatus, Message: "stopping"})
		go s.Stop()
	default:
		response(Event{Type: EventProjectFailed, Stage: "failed", Message: "unknown agent request: " + request.Type})
	}
}
