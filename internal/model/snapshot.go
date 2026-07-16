package model

import "time"

type TrackingState string

const (
	TrackingTracked   TrackingState = "tracked"
	TrackingUntracked TrackingState = "untracked"
)

type FollowState string

const (
	FollowFollowing FollowState = "following"
	FollowRecent    FollowState = "recent"
	FollowQuiet     FollowState = "quiet"
)

type Tracking struct {
	Path            string   `json:"path"`
	AutoReactivated []string `json:"auto_reactivated"`
}

type RemoteEvidence struct {
	PullRequests []PullRequest `json:"pull_requests"`
	Issues       []Issue       `json:"issues"`
	Errors       []ScanError   `json:"errors"`
	Warnings     []ScanError   `json:"warnings"`
}

type RemoteCollection struct {
	Repositories map[string]RemoteEvidence `json:"repositories"`
	Errors       []ScanError               `json:"errors"`
	Warnings     []ScanError               `json:"warnings"`
}

type Groups struct {
	Ready     []string `json:"ready"`
	Action    []string `json:"action"`
	Waiting   []string `json:"waiting"`
	Idle      []string `json:"idle"`
	Untracked []string `json:"untracked"`
}

type WorkingSet struct {
	Path    string   `json:"path"`
	Active  []string `json:"active"`
	Waiting []string `json:"waiting"`
	Recent  []string `json:"recent"`
	Parked  []string `json:"parked"`
}

type Snapshot struct {
	SchemaVersion int         `json:"schema_version"`
	GeneratedAt   time.Time   `json:"generated_at"`
	ConfigPath    string      `json:"config_path"`
	Tracking      Tracking    `json:"tracking"`
	WorkingSet    WorkingSet  `json:"working_set"`
	Refresh       []Refresh   `json:"refresh"`
	Summary       Summary     `json:"summary"`
	Groups        Groups      `json:"groups"`
	Projects      []Project   `json:"projects"`
	Lanes         []Lane      `json:"lanes"`
	Errors        []ScanError `json:"errors"`
	Warnings      []ScanError `json:"warnings"`
}
