package model

import "time"

const SchemaVersion = 3

type WorktreeState string
type PublicationState string
type PullRequestState string
type CIState string
type ReviewState string
type MergeState string
type FreshnessState string
type IssueState string
type Action string

const (
	WorktreeClean       WorktreeState = "clean"
	WorktreeDirty       WorktreeState = "dirty"
	WorktreeConflicted  WorktreeState = "conflicted"
	WorktreeUnavailable WorktreeState = "unavailable"
	WorktreeNotLocal    WorktreeState = "not_local"

	PublicationBase       PublicationState = "base"
	PublicationNoUpstream PublicationState = "no_upstream"
	PublicationUnpushed   PublicationState = "unpushed"
	PublicationPublished  PublicationState = "published"
	PublicationBehind     PublicationState = "behind"
	PublicationDiverged   PublicationState = "diverged"
	PublicationUnknown    PublicationState = "unknown"

	PullRequestNone  PullRequestState = "none"
	PullRequestDraft PullRequestState = "draft"
	PullRequestOpen  PullRequestState = "open"

	CISuccess CIState = "success"
	CIPending CIState = "pending"
	CIFailure CIState = "failure"
	CINone    CIState = "none"
	CIUnknown CIState = "unknown"

	ReviewNone             ReviewState = "none"
	ReviewRequired         ReviewState = "review_required"
	ReviewFeedbackPending  ReviewState = "feedback_pending"
	ReviewChangesRequested ReviewState = "changes_requested"
	ReviewApproved         ReviewState = "approved"
	ReviewUnknown          ReviewState = "unknown"

	MergeClean       MergeState = "clean"
	MergeBlocked     MergeState = "blocked"
	MergeConflicting MergeState = "conflicting"
	MergeUnknown     MergeState = "unknown"

	FreshnessCurrent FreshnessState = "current"
	FreshnessStale   FreshnessState = "stale"

	IssueNone IssueState = "none"
	IssueOpen IssueState = "open"

	ActionReviewPR        Action = "review_pr"
	ActionResolveConflict Action = "resolve_conflict"
	ActionFixCI           Action = "fix_ci"
	ActionAddressReview   Action = "address_review"
	ActionInspectLocal    Action = "inspect_local"
	ActionPushBranch      Action = "push_branch"
	ActionCreatePR        Action = "create_pr"
	ActionMarkReady       Action = "mark_ready"
	ActionWaitForCI       Action = "wait_for_ci"
	ActionManualTestMerge Action = "manual_test_then_merge"
	ActionMergePR         Action = "merge_pr"
	ActionStartIssue      Action = "start_issue"
	ActionRefreshState    Action = "refresh_state"
	ActionResumeOrClose   Action = "resume_or_close"
	ActionContinueWork    Action = "continue_work"
	ActionNone            Action = "none"
)

type CheckSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Pending int `json:"pending"`
	Failure int `json:"failure"`
	Skipped int `json:"skipped"`
	Unknown int `json:"unknown"`
}

type Feedback struct {
	Comments          int `json:"comments"`
	Reviews           int `json:"reviews"`
	Approvals         int `json:"approvals"`
	ChangesRequested  int `json:"changes_requested"`
	UnresolvedThreads int `json:"unresolved_threads"`
}

type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Labels    []string  `json:"labels"`
	Assignees []string  `json:"assignees"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Progress struct {
	Source    string `json:"source"`
	FeatureID string `json:"feature_id"`
	Feature   string `json:"feature"`
	Phase     string `json:"phase"`
	Summary   string `json:"summary"`
	Path      string `json:"path"`
}

type Worktree struct {
	Path       string    `json:"path"`
	HeadOID    string    `json:"head_oid"`
	Upstream   string    `json:"upstream,omitempty"`
	Staged     int       `json:"staged"`
	Unstaged   int       `json:"unstaged"`
	Untracked  int       `json:"untracked"`
	Conflicted int       `json:"conflicted"`
	Ahead      int       `json:"ahead"`
	Behind     int       `json:"behind"`
	AheadBase  int       `json:"ahead_base"`
	BehindBase int       `json:"behind_base"`
	Detached   bool      `json:"detached"`
	Locked     bool      `json:"locked"`
	Prunable   bool      `json:"prunable"`
	UpdatedAt  time.Time `json:"updated_at"`
	StatusHash string    `json:"-"`
}

type PullRequest struct {
	Number         int          `json:"number"`
	Title          string       `json:"title"`
	URL            string       `json:"url"`
	HeadRefName    string       `json:"head_ref_name"`
	HeadRefOID     string       `json:"head_ref_oid"`
	BaseRefName    string       `json:"base_ref_name"`
	IsDraft        bool         `json:"is_draft"`
	UpdatedAt      time.Time    `json:"updated_at"`
	ReviewDecision string       `json:"review_decision,omitempty"`
	MergeState     string       `json:"merge_state_status,omitempty"`
	Mergeable      string       `json:"mergeable,omitempty"`
	CI             CIState      `json:"ci_state"`
	Checks         CheckSummary `json:"checks"`
	Feedback       Feedback     `json:"feedback"`
	ClosingIssues  []Issue      `json:"closing_issues"`
}

type CheckoutWarning struct {
	Kind              string    `json:"kind"`
	Severity          string    `json:"severity"`
	PullRequestNumber int       `json:"pull_request_number"`
	PullRequestURL    string    `json:"pull_request_url,omitempty"`
	Branch            string    `json:"branch"`
	Base              string    `json:"base"`
	MergedAt          time.Time `json:"merged_at"`
	ConfirmedAt       time.Time `json:"confirmed_at"`
	Message           string    `json:"message"`
}

type Signals struct {
	Worktree    WorktreeState    `json:"worktree"`
	Publication PublicationState `json:"publication"`
	PullRequest PullRequestState `json:"pull_request"`
	CI          CIState          `json:"ci"`
	Review      ReviewState      `json:"review"`
	Merge       MergeState       `json:"merge"`
	Freshness   FreshnessState   `json:"freshness"`
	Issue       IssueState       `json:"issue"`
}

type Lane struct {
	ID              string           `json:"id"`
	Repository      string           `json:"repository"`
	GitHub          string           `json:"github"`
	Base            string           `json:"base"`
	Branch          string           `json:"branch"`
	Worktree        *Worktree        `json:"worktree,omitempty"`
	PullRequest     *PullRequest     `json:"pull_request,omitempty"`
	Issue           *Issue           `json:"issue,omitempty"`
	Progress        *Progress        `json:"progress,omitempty"`
	Signals         Signals          `json:"signals"`
	ReviewReady     bool             `json:"review_ready"`
	NextAction      Action           `json:"next_action"`
	Reasons         []string         `json:"reasons"`
	Warnings        []string         `json:"warnings"`
	Blockers        []string         `json:"blockers"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Attention       *LaneAttention   `json:"attention,omitempty"`
	CheckoutWarning *CheckoutWarning `json:"checkout_warning,omitempty"`
}

type ScanError struct {
	Repository string `json:"repository,omitempty"`
	Stage      string `json:"stage"`
	Message    string `json:"message"`
}

type Refresh struct {
	Repository string    `json:"repository"`
	Attempted  bool      `json:"attempted"`
	Refreshed  bool      `json:"refreshed"`
	At         time.Time `json:"at,omitempty"`
	Error      string    `json:"error,omitempty"`
}

type Summary struct {
	Projects           int `json:"projects"`
	TrackedProjects    int `json:"tracked_projects"`
	UntrackedProjects  int `json:"untracked_projects"`
	FollowingProjects  int `json:"following_projects"`
	RecentProjects     int `json:"recent_projects"`
	QuietProjects      int `json:"quiet_projects"`
	Total              int `json:"total"`
	ReviewReady        int `json:"review_ready"`
	NeedsAction        int `json:"needs_action"`
	Waiting            int `json:"waiting"`
	Idle               int `json:"idle"`
	Errors             int `json:"errors"`
	Warnings           int `json:"warnings"`
	OpenIssues         int `json:"open_issues"`
	UnresolvedFeedback int `json:"unresolved_feedback"`
	ActiveLanes        int `json:"active_lanes"`
	RecentLanes        int `json:"recent_lanes"`
	ParkedLanes        int `json:"parked_lanes"`
}

type Project struct {
	Name           string        `json:"name"`
	Path           string        `json:"path"`
	GitHub         string        `json:"github"`
	Base           string        `json:"base"`
	Remote         string        `json:"remote"`
	TrackingState  TrackingState `json:"tracking_state"`
	FollowState    FollowState   `json:"follow_state"`
	LastActivityAt time.Time     `json:"last_activity_at,omitzero"`
	ActivityReason string        `json:"activity_reason,omitempty"`
	Progress       *Progress     `json:"progress,omitempty"`
	LaneIDs        []string      `json:"lane_ids"`
	Errors         []ScanError   `json:"errors"`
	Warnings       []ScanError   `json:"warnings"`
}
