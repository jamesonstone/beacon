package model

import "time"

type AttentionState string

const (
	AttentionActive  AttentionState = "active"
	AttentionWaiting AttentionState = "waiting"
	AttentionRecent  AttentionState = "recent"
	AttentionParked  AttentionState = "parked"
)

type LaneObservation struct {
	HeadOID         string           `json:"head_oid,omitempty"`
	StatusHash      string           `json:"status_hash,omitempty"`
	Worktree        WorktreeState    `json:"worktree"`
	Publication     PublicationState `json:"publication"`
	PullRequest     int              `json:"pull_request,omitempty"`
	CI              CIState          `json:"ci"`
	Review          ReviewState      `json:"review"`
	Merge           MergeState       `json:"merge"`
	Unresolved      int              `json:"unresolved_feedback"`
	ObservedAt      time.Time        `json:"observed_at"`
	RemoteUpdatedAt time.Time        `json:"remote_updated_at,omitempty"`
}

type LaneAttention struct {
	State              AttentionState  `json:"state"`
	Pinned             bool            `json:"pinned"`
	Manual             bool            `json:"manual"`
	Title              string          `json:"title,omitempty"`
	Tags               []string        `json:"tags"`
	Note               string          `json:"note,omitempty"`
	NoteUpdatedAt      time.Time       `json:"note_updated_at,omitempty"`
	NoteStale          bool            `json:"note_stale"`
	LastSeenAt         time.Time       `json:"last_seen_at,omitempty"`
	Delta              string          `json:"delta"`
	ReactivationReason string          `json:"reactivation_reason,omitempty"`
	Previous           LaneObservation `json:"previous"`
	Current            LaneObservation `json:"current"`
}
