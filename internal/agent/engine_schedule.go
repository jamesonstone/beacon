package agent

import (
	"context"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

func (e *Engine) RunSchedule(ctx context.Context) {
	interval := e.Config.Settings.TrackedRefreshInterval
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if e.hasDueCachedProject() {
		_, _ = e.Refresh(ctx, "", false)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if e.hasDueCachedProject() {
				_, _ = e.Refresh(ctx, "", false)
			}
		}
	}
}

func (e *Engine) hasDueCachedProject() bool {
	e.mutex.RLock()
	records := make([]ProjectRecord, 0, len(e.records))
	for _, record := range e.records {
		records = append(records, record)
	}
	e.mutex.RUnlock()
	if len(records) == 0 {
		return true
	}
	frequent := e.frequentRepositories()
	for _, record := range records {
		if len(record.Snapshot.Projects) == 0 {
			return true
		}
		project := record.Snapshot.Projects[0]
		_, laneNeedsFrequentObservation := frequent[project.GitHub]
		if e.WorkingSet != nil {
			interval := e.Config.Settings.UntrackedProbeInterval
			if laneNeedsFrequentObservation {
				interval = e.Config.Settings.TrackedRefreshInterval
			}
			if e.now().Sub(record.UpdatedAt) >= interval {
				return true
			}
			continue
		}
		if project.TrackingState != model.TrackingUntracked {
			if e.now().Sub(record.UpdatedAt) >= e.Config.Settings.TrackedRefreshInterval {
				return true
			}
			continue
		}
		if record.LastProbeAt.IsZero() || e.now().Sub(record.LastProbeAt) >= e.Config.Settings.UntrackedProbeInterval {
			return true
		}
	}
	return false
}

func (e *Engine) due(repository config.Repository, frequent map[string]struct{}) bool {
	record, exists := e.record(repository.GitHub)
	if !exists || len(record.Snapshot.Projects) == 0 {
		return true
	}
	_, laneNeedsFrequentObservation := frequent[repository.GitHub]
	if e.WorkingSet != nil {
		interval := e.Config.Settings.UntrackedProbeInterval
		if laneNeedsFrequentObservation {
			interval = e.Config.Settings.TrackedRefreshInterval
		}
		return e.now().Sub(record.UpdatedAt) >= interval
	}
	if record.Snapshot.Projects[0].TrackingState != model.TrackingUntracked {
		return e.now().Sub(record.UpdatedAt) >= e.Config.Settings.TrackedRefreshInterval
	}
	return record.LastProbeAt.IsZero() || e.now().Sub(record.LastProbeAt) >= e.Config.Settings.UntrackedProbeInterval
}

func (e *Engine) frequentRepositories() map[string]struct{} {
	if e.WorkingSet == nil {
		return map[string]struct{}{}
	}
	candidates, err := e.WorkingSet.FrequentRepositories()
	if err != nil {
		return map[string]struct{}{}
	}
	repositories := make(map[string]struct{}, len(candidates))
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	for github := range candidates {
		record, found := e.records[github]
		if !found || len(record.Snapshot.Projects) == 0 {
			continue
		}
		project := record.Snapshot.Projects[0]
		following := project.FollowState == model.FollowFollowing
		if project.FollowState == "" {
			following = project.TrackingState != model.TrackingUntracked
		}
		if following {
			repositories[github] = struct{}{}
		}
	}
	return repositories
}
