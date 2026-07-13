package tracking

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestManagerUntracksAndAutomaticallyReactivatesChangedEvidence(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	manager := Manager{Store: FileStore{}, Now: func() time.Time { return now }}
	snapshot := managerSnapshot(t)

	untracked, err := manager.SetTracked(snapshot, []string{"repo"}, false)
	if err != nil {
		t.Fatal(err)
	}
	assertUntracked(t, untracked)
	unchanged, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	assertUntracked(t, unchanged)

	snapshot.Lanes[0].Worktree.HeadOID = "new-head"
	reactivated, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if reactivated.Projects[0].TrackingState != model.TrackingTracked || len(reactivated.Tracking.AutoReactivated) != 1 {
		t.Fatalf("reactivated snapshot = %#v", reactivated)
	}
	state, err := (FileStore{}).Load(reactivated.Tracking.Path)
	if err != nil || len(state.Untracked) != 0 {
		t.Fatalf("persisted state = %#v, %v", state, err)
	}
}

func TestManagerDoesNotReactivateWhenCollectionFailed(t *testing.T) {
	manager := Manager{Store: FileStore{}, Now: func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) }}
	snapshot := managerSnapshot(t)
	if _, err := manager.SetTracked(snapshot, []string{"owner/repo"}, false); err != nil {
		t.Fatal(err)
	}
	snapshot.Lanes[0].Worktree.HeadOID = "changed"
	snapshot.Projects[0].Errors = []model.ScanError{{Repository: "repo", Stage: "github", Message: "timeout"}}
	result, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	assertUntracked(t, result)
}

func TestManagerDoesNotReactivateWhenGlobalCollectionFailed(t *testing.T) {
	manager := Manager{Store: FileStore{}, Now: func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) }}
	snapshot := managerSnapshot(t)
	if _, err := manager.SetTracked(snapshot, []string{"owner/repo"}, false); err != nil {
		t.Fatal(err)
	}
	snapshot.Lanes[0].Worktree.HeadOID = "changed-during-global-error"
	snapshot.Errors = []model.ScanError{{Stage: "github-search-prs", Message: "authentication failed"}}
	result, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	assertUntracked(t, result)
}

func TestManagerDefersBaselineUntilCollectionRecovers(t *testing.T) {
	manager := Manager{Store: FileStore{}, Now: func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) }}
	snapshot := managerSnapshot(t)
	snapshot.Errors = []model.ScanError{{Stage: "github-search-prs", Message: "authentication failed"}}
	untracked, err := manager.SetTracked(snapshot, []string{"owner/repo"}, false)
	if err != nil {
		t.Fatal(err)
	}
	assertUntracked(t, untracked)
	state, err := (FileStore{}).Load(untracked.Tracking.Path)
	if err != nil || len(state.Untracked) != 1 || state.Untracked[0].Baseline != "" {
		t.Fatalf("pending state=%#v err=%v", state, err)
	}

	snapshot.Errors = []model.ScanError{}
	initialized, err := manager.Reconcile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	assertUntracked(t, initialized)
	state, err = (FileStore{}).Load(initialized.Tracking.Path)
	if err != nil || len(state.Untracked) != 1 || !fingerprintPattern.MatchString(state.Untracked[0].Baseline) {
		t.Fatalf("initialized state=%#v err=%v", state, err)
	}
	if len(initialized.Tracking.AutoReactivated) != 0 {
		t.Fatalf("baseline initialization reactivated project: %#v", initialized.Tracking)
	}
}

func TestManagerManualTrackingAndSelectionAreIdempotent(t *testing.T) {
	manager := Manager{Store: FileStore{}, Now: func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) }}
	snapshot := managerSnapshot(t)
	untracked, err := manager.SetSelection(snapshot, []string{})
	if err != nil {
		t.Fatal(err)
	}
	assertUntracked(t, untracked)
	tracked, err := manager.SetTracked(untracked, []string{"owner/repo"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if tracked.Projects[0].TrackingState != model.TrackingTracked || tracked.Summary.TrackedProjects != 1 {
		t.Fatalf("tracked snapshot = %#v", tracked)
	}
	if _, err := manager.SetTracked(tracked, []string{"owner/repo"}, true); err != nil {
		t.Fatal(err)
	}
}

func TestManagerExplicitUntrackReplacesStaleBaseline(t *testing.T) {
	manager := Manager{Store: FileStore{}, Now: func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) }}
	snapshot := managerSnapshot(t)
	untracked, err := manager.SetTracked(snapshot, []string{"owner/repo"}, false)
	if err != nil {
		t.Fatal(err)
	}
	untracked.Lanes[0].Worktree.HeadOID = "new-head"
	reaffirmed, err := manager.SetTracked(untracked, []string{"owner/repo"}, false)
	if err != nil {
		t.Fatal(err)
	}
	assertUntracked(t, reaffirmed)
	unchanged, err := manager.Reconcile(untracked)
	if err != nil {
		t.Fatal(err)
	}
	assertUntracked(t, unchanged)
}

func TestManagerProjectAliasesMustExistAndBeUnique(t *testing.T) {
	manager := Manager{Store: FileStore{}, Now: func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) }}
	snapshot := managerSnapshot(t)
	duplicate := snapshot.Projects[0]
	duplicate.GitHub = "other/repo"
	snapshot.Projects = append(snapshot.Projects, duplicate)
	for _, target := range []string{"missing", "repo"} {
		if _, err := manager.SetTracked(snapshot, []string{target}, false); err == nil {
			t.Fatalf("target %q unexpectedly resolved", target)
		}
	}
}

func TestManagerDoesNotPublishReactivationWhenStateWriteFails(t *testing.T) {
	snapshot := managerSnapshot(t)
	project := snapshot.Projects[0]
	baseline, err := Fingerprint(project, snapshot.Lanes)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Lanes[0].Worktree.HeadOID = "new-head"
	store := &failingStore{
		state: State{Version: Version, Untracked: []Entry{{
			GitHub: project.GitHub, Name: project.Name, Path: project.Path,
			UntrackedAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC), Baseline: baseline,
		}}},
		writeErr: errors.New("disk full"),
	}
	_, err = (Manager{Store: store}).Reconcile(snapshot)
	if err == nil || !strings.Contains(err.Error(), "persist automatic project reactivation") {
		t.Fatalf("error = %v", err)
	}
}

func managerSnapshot(t *testing.T) model.Snapshot {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	project, lanes := trackingFixture()
	return model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		ConfigPath:    filepath.Join(t.TempDir(), "config.yaml"),
		Projects:      []model.Project{project},
		Lanes:         lanes,
		Groups: model.Groups{
			Ready: []string{"gh:owner/repo#2"}, Idle: []string{"git:owner/repo@main"}, Untracked: []string{},
		},
		Summary: model.Summary{Projects: 1, Total: 2, ReviewReady: 1, Idle: 1, OpenIssues: 1},
		Errors:  []model.ScanError{}, Warnings: []model.ScanError{},
	}
}

func assertUntracked(t *testing.T, snapshot model.Snapshot) {
	t.Helper()
	if snapshot.Projects[0].TrackingState != model.TrackingUntracked || snapshot.Summary.Projects != 1 || snapshot.Summary.TrackedProjects != 0 || snapshot.Summary.UntrackedProjects != 1 {
		t.Fatalf("untracked project summary = %#v %#v", snapshot.Projects, snapshot.Summary)
	}
	if len(snapshot.Groups.Ready) != 0 || len(snapshot.Groups.Idle) != 0 || len(snapshot.Groups.Untracked) != 2 || snapshot.Summary.Total != 0 {
		t.Fatalf("untracked groups = %#v %#v", snapshot.Groups, snapshot.Summary)
	}
}

type failingStore struct {
	state    State
	writeErr error
}

func (s *failingStore) Load(string) (State, error) { return s.state, nil }
func (s *failingStore) Write(string, State) error  { return s.writeErr }
