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
)

type Server struct {
	Paths  Paths
	Engine *Engine
	Notes  notes.Store

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
		document, notesErr := s.notesStore().Load(s.Paths.Notes)
		if notesErr != nil {
			response(Event{Type: EventProjectFailed, Stage: "notes", Message: notesErr.Error()})
			return
		}
		response(Event{Type: EventNotes, Notes: &document})
	case RequestSetNotes, RequestAppendNotes:
		store := s.notesStore()
		var document notes.Document
		var notesErr error
		if request.Type == RequestAppendNotes {
			document, notesErr = store.Append(s.Paths.Notes, request.Content)
		} else {
			document, notesErr = store.Write(s.Paths.Notes, request.Content)
		}
		if notesErr != nil {
			response(Event{Type: EventProjectFailed, Stage: "notes", Message: notesErr.Error()})
			return
		}
		event := Event{
			ProtocolVersion: ProtocolVersion, Type: EventNotesUpdated,
			GeneratedAt: time.Now().UTC(), Notes: &document,
		}
		s.Engine.hub.Publish(event)
		response(event)
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

func (s *Server) notesStore() notes.Store {
	if s.Notes != nil {
		return s.Notes
	}
	return notes.FileStore{}
}

func (s *Server) heartbeats(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.Engine.hub.Publish(Event{ProtocolVersion: ProtocolVersion, Type: EventHeartbeat, GeneratedAt: now.UTC()})
		}
	}
}
