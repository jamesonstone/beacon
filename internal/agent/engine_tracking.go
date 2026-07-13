package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func (e *Engine) SetTracking(ctx context.Context, projectID, state string) error {
	_ = ctx
	return e.SetTrackingBatch([]string{projectID}, state)
}

func (e *Engine) SetTrackingBatch(projectIDs []string, state string) error {
	tracked := state == "tracked"
	if !tracked && state != "muted" {
		return fmt.Errorf("tracking state must be tracked or muted: %q", state)
	}
	snapshot := e.Snapshot()
	updated, err := e.Tracker.SetTracked(snapshot, projectIDs, tracked)
	if err != nil {
		return err
	}
	if err := e.applyTrackingSnapshot(updated); err != nil {
		return err
	}
	e.publish(EventTrackingChanged, "", "", 0, "ready", state, pointer(e.Snapshot()))
	return nil
}

func (e *Engine) SetSelection(projectIDs []string) error {
	updated, err := e.Tracker.SetSelection(e.Snapshot(), projectIDs)
	if err != nil {
		return err
	}
	if err := e.applyTrackingSnapshot(updated); err != nil {
		return err
	}
	e.publish(EventTrackingChanged, "", "", 0, "ready", "selection", pointer(e.Snapshot()))
	return nil
}

func (e *Engine) applyTrackingSnapshot(updated model.Snapshot) error {
	byID := make(map[string]model.Project, len(updated.Projects))
	for _, project := range updated.Projects {
		byID[project.GitHub] = project
	}
	e.mutex.Lock()
	for id, record := range e.records {
		project, found := byID[id]
		if !found || len(record.Snapshot.Projects) == 0 {
			continue
		}
		cachedProject := &record.Snapshot.Projects[0]
		previous := cachedProject.TrackingState
		cachedProject.TrackingState = project.TrackingState
		cachedProject.FollowState = project.FollowState
		cachedProject.LastActivityAt = project.LastActivityAt
		cachedProject.ActivityReason = project.ActivityReason
		if previous != model.TrackingUntracked && project.TrackingState == model.TrackingUntracked {
			record.LastProbeAt = e.now()
		}
		if project.TrackingState != model.TrackingUntracked {
			record.LastProbeAt = time.Time{}
		}
		record.Revision++
		record.UpdatedAt = e.now()
		e.records[id] = record
		e.revisions[id] = record.Revision
		if err := e.Cache.Write(record); err != nil {
			e.mutex.Unlock()
			return err
		}
	}
	e.mutex.Unlock()
	return nil
}
