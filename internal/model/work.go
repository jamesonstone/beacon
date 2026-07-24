package model

import "time"

const WorkScanSchemaVersion = 1

type WorkState string

const (
	WorkConflict    WorkState = "conflict"
	WorkCIFailed    WorkState = "ci_failed"
	WorkFeedback    WorkState = "feedback"
	WorkDirty       WorkState = "dirty"
	WorkUnpublished WorkState = "unpublished"
	WorkDraft       WorkState = "draft"
	WorkPullRequest WorkState = "pull_request"
	WorkBranch      WorkState = "branch"
	WorkIdle        WorkState = "idle"
)

type WorkScanSummary struct {
	Projects        int `json:"projects"`
	ActiveProjects  int `json:"active_projects"`
	WorkItems       int `json:"work_items"`
	IdleProjects    int `json:"idle_projects"`
	UnknownProjects int `json:"unknown_projects"`
	Errors          int `json:"errors"`
	Warnings        int `json:"warnings"`
}

type WorktreeSummary struct {
	Path       string `json:"path"`
	Staged     int    `json:"staged"`
	Unstaged   int    `json:"unstaged"`
	Untracked  int    `json:"untracked"`
	Conflicted int    `json:"conflicted"`
	Ahead      int    `json:"ahead"`
	Behind     int    `json:"behind"`
	AheadBase  int    `json:"ahead_base"`
	BehindBase int    `json:"behind_base"`
	Detached   bool   `json:"detached"`
}

type WorkPullRequestSummary struct {
	Number int         `json:"number"`
	Title  string      `json:"title"`
	URL    string      `json:"url"`
	Draft  bool        `json:"draft"`
	CI     CIState     `json:"ci"`
	Review ReviewState `json:"review"`
}

type WorkItem struct {
	Repository     string                  `json:"repository"`
	GitHub         string                  `json:"github"`
	RepositoryPath string                  `json:"repository_path"`
	Branch         string                  `json:"branch"`
	Base           string                  `json:"base"`
	State          WorkState               `json:"state"`
	Publication    PublicationState        `json:"publication"`
	NextAction     Action                  `json:"next_action"`
	UpdatedAt      time.Time               `json:"updated_at,omitzero"`
	Worktree       *WorktreeSummary        `json:"worktree,omitempty"`
	PullRequest    *WorkPullRequestSummary `json:"pull_request,omitempty"`
}

type WorkScan struct {
	SchemaVersion int             `json:"schema_version"`
	GeneratedAt   time.Time       `json:"generated_at"`
	Summary       WorkScanSummary `json:"summary"`
	Items         []WorkItem      `json:"items"`
	Errors        []ScanError     `json:"errors"`
	Warnings      []ScanError     `json:"warnings"`
}
