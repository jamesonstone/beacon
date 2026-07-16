package agent

import (
	"context"
	"time"

	"github.com/jamesonstone/beacon/internal/notes"
)

func (s *Server) notesStore() notes.WorkspaceStore {
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
