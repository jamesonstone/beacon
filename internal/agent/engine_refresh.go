package agent

import (
	"context"
	"fmt"
	"sort"

	"github.com/jamesonstone/beacon/internal/config"
)

func (e *Engine) Refresh(ctx context.Context, project string, force bool) (string, error) {
	e.mutex.Lock()
	if e.refreshing {
		scanID := e.scanID
		e.queueRefreshLocked(project, force)
		e.mutex.Unlock()
		return scanID, nil
	}
	e.refreshing = true
	e.scanID = newID()
	scanID := e.scanID
	e.activeAll = force && project == ""
	e.active = make(map[string]struct{})
	e.pending = make(map[string]struct{})
	if project != "" {
		e.active[project] = struct{}{}
	}
	e.mutex.Unlock()
	go e.startRefresh(ctx, scanID, project, force)
	return scanID, nil
}

func (e *Engine) startRefresh(ctx context.Context, scanID, project string, force bool) {
	repositories, err := e.Repositories(ctx)
	if err != nil {
		e.failProject(scanID, project, 0, err)
		e.completeScan(scanID)
		return
	}
	selected := e.selectRepositories(repositories, project, force)
	if project != "" && len(selected) == 0 {
		e.failProject(scanID, project, 0, fmt.Errorf("project not found: %s", project))
		e.completeScan(scanID)
		return
	}
	e.markActive(selected)
	e.publishQueued(scanID, selected)
	batch := selected
	for {
		e.runBatch(ctx, scanID, batch, force)
		var pendingForce bool
		batch, pendingForce = e.nextBatch(scanID, repositories)
		if batch == nil {
			return
		}
		force = pendingForce
		e.publishQueued(scanID, batch)
	}
}

func (e *Engine) queueRefreshLocked(project string, force bool) {
	if e.activeAll || e.pendingAll {
		return
	}
	if project == "" {
		if force {
			e.pendingAll = true
			e.pending = make(map[string]struct{})
		}
		return
	}
	if _, running := e.active[project]; running {
		return
	}
	e.pending[project] = struct{}{}
}

func (e *Engine) selectRepositories(repositories []config.Repository, project string, force bool) map[string]config.Repository {
	selected := make(map[string]config.Repository)
	frequent := e.frequentRepositories()
	for _, repository := range repositories {
		if project != "" && project != repository.GitHub && project != repository.Name {
			continue
		}
		if !force && !e.due(repository, frequent) {
			continue
		}
		selected[repository.GitHub] = repository
	}
	return selected
}

func (e *Engine) markActive(repositories map[string]config.Repository) {
	e.mutex.Lock()
	for projectID, repository := range repositories {
		e.active[projectID] = struct{}{}
		e.active[repository.Name] = struct{}{}
	}
	e.mutex.Unlock()
}

func (e *Engine) publishQueued(scanID string, repositories map[string]config.Repository) {
	projectIDs := make([]string, 0, len(repositories))
	for projectID := range repositories {
		projectIDs = append(projectIDs, projectID)
	}
	sort.Strings(projectIDs)
	for _, projectID := range projectIDs {
		repository := repositories[projectID]
		revision := e.revision(projectID) + 1
		e.publishProject(EventProjectDiscovered, scanID, repository, revision, "cached", repository.Name, nil)
		e.setStage(projectID, "queued")
		e.publishProject(EventProjectQueued, scanID, repository, revision, "queued", repository.Name, nil)
	}
}

func (e *Engine) runBatch(ctx context.Context, scanID string, repositories map[string]config.Repository, force bool) {
	if e.ScanBatch != nil && e.ProbeBatch != nil {
		e.runCollectedBatch(ctx, scanID, repositories, force)
	} else {
		e.runProjectBatch(ctx, scanID, repositories, force)
	}
	e.mutex.Lock()
	for projectID := range repositories {
		delete(e.active, projectID)
		delete(e.active, repositories[projectID].Name)
	}
	e.mutex.Unlock()
}

func (e *Engine) runProjectBatch(ctx context.Context, scanID string, repositories map[string]config.Repository, force bool) {
	projectIDs := make([]string, 0, len(repositories))
	for projectID := range repositories {
		projectIDs = append(projectIDs, projectID)
	}
	sort.Strings(projectIDs)
	Scheduler{MaxParallel: e.Config.Settings.MaxParallel}.Run(ctx, projectIDs, func(jobContext context.Context, projectID string) {
		e.refreshProject(jobContext, scanID, repositories[projectID], force)
	})
}

func (e *Engine) nextBatch(scanID string, repositories []config.Repository) (map[string]config.Repository, bool) {
	e.mutex.Lock()
	if e.scanID != scanID {
		e.mutex.Unlock()
		return nil, false
	}
	pendingAll := e.pendingAll
	pending := e.pending
	e.pendingAll = false
	e.pending = make(map[string]struct{})
	if !pendingAll && len(pending) == 0 {
		e.refreshing = false
		e.activeAll = false
		e.active = nil
		e.scanID = ""
		e.mutex.Unlock()
		e.publish(EventScanCompleted, scanID, "", 0, "ready", "", pointer(e.Snapshot()))
		return nil, false
	}
	if pendingAll {
		e.activeAll = true
	}
	selected := make(map[string]config.Repository)
	matched := make(map[string]struct{}, len(pending))
	for _, repository := range repositories {
		if pendingAll {
			selected[repository.GitHub] = repository
			continue
		}
		if _, found := pending[repository.GitHub]; found {
			selected[repository.GitHub] = repository
			matched[repository.GitHub] = struct{}{}
			continue
		}
		if _, found := pending[repository.Name]; found {
			selected[repository.GitHub] = repository
			matched[repository.Name] = struct{}{}
		}
	}
	for projectID, repository := range selected {
		e.active[projectID] = struct{}{}
		e.active[repository.Name] = struct{}{}
	}
	e.mutex.Unlock()
	for projectID := range pending {
		if _, found := matched[projectID]; !found {
			e.failProject(scanID, projectID, e.revision(projectID)+1, fmt.Errorf("project not found: %s", projectID))
		}
	}
	return selected, true
}
