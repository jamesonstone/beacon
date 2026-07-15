package agent

import (
	"errors"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
)

func (e *Engine) completeScan(scanID string) {
	e.mutex.Lock()
	if e.scanID == scanID {
		e.refreshing = false
		e.activeAll = false
		e.active = nil
		e.pendingAll = false
		e.pending = nil
		e.scanID = ""
	}
	e.mutex.Unlock()
	e.clearCheckoutConfirmationBudget(scanID)
	e.publish(EventScanCompleted, scanID, "", 0, "ready", "", pointer(e.Snapshot()))
}

func (e *Engine) failProject(scanID, projectID string, revision uint64, err error) {
	e.mutex.Lock()
	e.stages[projectID] = "failed"
	if revision > e.revisions[projectID] {
		e.revisions[projectID] = revision
	}
	e.mutex.Unlock()
	e.publish(EventProjectFailed, scanID, projectID, revision, "failed", err.Error(), pointer(e.Snapshot()))
}

func (e *Engine) publish(eventType, scanID, projectID string, revision uint64, stage, message string, snapshot *model.Snapshot) {
	e.hub.Publish(Event{
		ProtocolVersion: ProtocolVersion, Type: eventType, ScanID: scanID,
		ProjectID: projectID, Revision: revision, Stage: stage,
		GeneratedAt: e.now(), Message: message, Snapshot: snapshot,
	})
}

func (e *Engine) publishProject(eventType, scanID string, repository config.Repository, revision uint64, stage, message string, snapshot *model.Snapshot) {
	status := e.projectStatus(repository, revision, stage)
	e.hub.Publish(Event{
		ProtocolVersion: ProtocolVersion, Type: eventType, ScanID: scanID,
		ProjectID: repository.GitHub, Revision: revision, Stage: stage,
		GeneratedAt: e.now(), Message: message, Snapshot: snapshot,
		Projects: []ProjectStatus{status},
	})
}

func (e *Engine) projectStatus(repository config.Repository, revision uint64, stage string) ProjectStatus {
	trackingState := model.TrackingTracked
	updatedAt := e.now()
	if record, found := e.record(repository.GitHub); found {
		updatedAt = record.UpdatedAt
		if len(record.Snapshot.Projects) > 0 {
			trackingState = record.Snapshot.Projects[0].TrackingState
		}
	}
	entry, muted, _ := e.Tracker.Entry(e.Config.Path, repository.GitHub)
	if muted {
		trackingState = model.TrackingUntracked
	}
	return ProjectStatus{
		ProjectID: repository.GitHub, Name: repository.Name, Path: repository.Path,
		Tracking: trackingState, Stage: stage, Revision: revision, UpdatedAt: updatedAt,
		MutedAt: entry.UntrackedAt, LastProbeAt: entry.LastProbeAt,
	}
}

func (e *Engine) record(projectID string) (ProjectRecord, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	record, found := e.records[projectID]
	return record, found
}

func (e *Engine) revision(projectID string) uint64 {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	revision := e.revisions[projectID]
	if record, found := e.records[projectID]; found && record.Revision > revision {
		revision = record.Revision
	}
	return revision
}

func (e *Engine) storeRecord(record ProjectRecord) {
	e.mutex.Lock()
	e.records[record.ProjectID] = record
	e.revisions[record.ProjectID] = record.Revision
	e.stages[record.ProjectID] = record.Stage
	e.mutex.Unlock()
}

func (e *Engine) setStage(projectID, stage string) {
	e.mutex.Lock()
	e.stages[projectID] = stage
	e.mutex.Unlock()
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func (e *Engine) probeAuthor() string {
	if e.Config.Settings.GitHubScope == config.GitHubScopeAll {
		return ""
	}
	return e.Config.Settings.GitHubAuthor
}

func projectActivityReason(entry tracking.Entry, probe ProbeResult) string {
	switch {
	case entry.ProbeLocal != "" && entry.ProbeLocal != probe.Local:
		return "new local changes"
	case entry.ProbeRemote != "" && entry.ProbeRemote != probe.Remote:
		return "new GitHub activity"
	default:
		return "material project evidence changed"
	}
}

func snapshotCollectionError(snapshot model.Snapshot) error {
	if len(snapshot.Errors) == 0 {
		return nil
	}
	messages := make([]string, 0, len(snapshot.Errors))
	for _, scanError := range snapshot.Errors {
		message := scanError.Stage + ": " + scanError.Message
		if scanError.Repository != "" {
			message = scanError.Repository + " " + message
		}
		messages = append(messages, message)
	}
	return errors.New(strings.Join(messages, "; "))
}

func pointer[T any](value T) *T { return &value }
