package agent

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/notes"
	"github.com/jamesonstone/beacon/internal/reposync"
)

const ProtocolVersion = 1

const (
	RequestGetSnapshot       = "get_snapshot"
	RequestSubscribe         = "subscribe"
	RequestRefreshAll        = "refresh_all"
	RequestRefreshProject    = "refresh_project"
	RequestSetTrackingState  = "set_tracking_state"
	RequestSetTrackingBatch  = "set_tracking_batch"
	RequestSetSelection      = "set_tracking_selection"
	RequestListProjects      = "list_projects"
	RequestGetAgentStatus    = "get_agent_status"
	RequestShutdown          = "shutdown"
	RequestSetLaneAttention  = "set_lane_attention"
	RequestSetLanePinned     = "set_lane_pinned"
	RequestSetLaneNote       = "set_lane_note"
	RequestAddLaneTag        = "add_lane_tag"
	RequestRemoveLaneTag     = "remove_lane_tag"
	RequestMarkLaneSeen      = "mark_lane_seen"
	RequestAddManualLane     = "add_manual_lane"
	RequestGetNotes          = "get_notes"
	RequestSetNotes          = "set_notes"
	RequestAppendNotes       = "append_notes"
	RequestGetNotesWorkspace = "get_notes_workspace"
	RequestCreateNote        = "create_note"
	RequestOpenNote          = "open_note"
	RequestCloseNote         = "close_note"
	RequestDeleteNote        = "delete_note"
	RequestGetRepositorySync = "get_repository_sync"
	RequestSyncRepositories  = "sync_repositories"
)

const (
	EventSnapshot           = "snapshot"
	EventProjectDiscovered  = "project_discovered"
	EventProjectQueued      = "project_queued"
	EventProjectLocalReady  = "project_local_ready"
	EventProjectUpdated     = "project_updated"
	EventProjectFailed      = "project_failed"
	EventTrackingChanged    = "tracking_changed"
	EventProjectReactivated = "project_reactivated"
	EventScanCompleted      = "scan_completed"
	EventHeartbeat          = "heartbeat"
	EventAgentStatus        = "agent_status"
	EventWorkingSetChanged  = "working_set_changed"
	EventNotes              = "notes"
	EventNotesUpdated       = "notes_updated"
	EventNotesWorkspace     = "notes_workspace"
	EventWorkspaceUpdated   = "notes_workspace_updated"
	EventRepositorySync     = "repository_sync"
)

type Request struct {
	ProtocolVersion int      `json:"protocol_version"`
	RequestID       string   `json:"request_id"`
	Type            string   `json:"type"`
	ProjectID       string   `json:"project_id,omitempty"`
	ProjectIDs      []string `json:"project_ids,omitempty"`
	TrackingState   string   `json:"tracking_state,omitempty"`
	LaneID          string   `json:"lane_id,omitempty"`
	AttentionState  string   `json:"attention_state,omitempty"`
	Pinned          bool     `json:"pinned,omitempty"`
	Note            string   `json:"note,omitempty"`
	Tag             string   `json:"tag,omitempty"`
	Title           string   `json:"title,omitempty"`
	Content         string   `json:"content,omitempty"`
	NoteID          string   `json:"note_id,omitempty"`
	Refresh         bool     `json:"refresh,omitempty"`
}

type ProjectStatus struct {
	ProjectID   string              `json:"project_id"`
	Name        string              `json:"name"`
	Path        string              `json:"path"`
	Tracking    model.TrackingState `json:"tracking_state"`
	Stage       string              `json:"stage"`
	Revision    uint64              `json:"revision"`
	UpdatedAt   time.Time           `json:"updated_at"`
	MutedAt     time.Time           `json:"muted_at,omitempty"`
	LastProbeAt time.Time           `json:"last_probe_at,omitempty"`
}

type Status struct {
	Running      bool      `json:"running"`
	PID          int       `json:"pid"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	Refreshing   bool      `json:"refreshing"`
	ScanID       string    `json:"scan_id,omitempty"`
	ProjectCount int       `json:"project_count"`
	Socket       string    `json:"socket"`
}

type Event struct {
	ProtocolVersion int              `json:"protocol_version"`
	RequestID       string           `json:"request_id,omitempty"`
	Type            string           `json:"type"`
	ScanID          string           `json:"scan_id,omitempty"`
	ProjectID       string           `json:"project_id,omitempty"`
	Revision        uint64           `json:"revision,omitempty"`
	Stage           string           `json:"stage,omitempty"`
	GeneratedAt     time.Time        `json:"generated_at"`
	Message         string           `json:"message,omitempty"`
	Snapshot        *model.Snapshot  `json:"snapshot,omitempty"`
	Projects        []ProjectStatus  `json:"projects,omitempty"`
	Status          *Status          `json:"status,omitempty"`
	Notes           *notes.Document  `json:"notes,omitempty"`
	NotesWorkspace  *notes.Workspace `json:"notes_workspace,omitempty"`
	RepositorySync  *reposync.Report `json:"repository_sync,omitempty"`
}

func Encode(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func DecodeRequest(reader io.Reader) (Request, error) {
	var request Request
	decoder := json.NewDecoder(bufio.NewReader(reader))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf("decode agent request: %w", err)
	}
	if request.ProtocolVersion != ProtocolVersion {
		return Request{}, fmt.Errorf("unsupported protocol version %d", request.ProtocolVersion)
	}
	if request.RequestID == "" || request.Type == "" {
		return Request{}, errors.New("agent request requires request_id and type")
	}
	return request, nil
}

type EventDecoder struct {
	decoder *json.Decoder
}

func NewEventDecoder(reader io.Reader) *EventDecoder {
	decoder := json.NewDecoder(bufio.NewReader(reader))
	decoder.DisallowUnknownFields()
	return &EventDecoder{decoder: decoder}
}

func (d *EventDecoder) Next() (Event, error) {
	var event Event
	if err := d.decoder.Decode(&event); err != nil {
		if errors.Is(err, io.EOF) {
			return Event{}, io.EOF
		}
		return Event{}, fmt.Errorf("decode agent event: %w", err)
	}
	if event.ProtocolVersion != ProtocolVersion {
		return Event{}, fmt.Errorf("unsupported protocol version %d", event.ProtocolVersion)
	}
	if event.Type == "" {
		return Event{}, errors.New("agent event requires type")
	}
	return event, nil
}
