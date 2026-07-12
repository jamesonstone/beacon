package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
)

type RepositoryProvider func(context.Context) ([]config.Repository, error)
type ProjectScanner func(context.Context, config.Repository, bool, func(string)) (model.Snapshot, error)
type ProjectBatchScanner func(context.Context, []config.Repository, bool, func(string, string)) (map[string]model.Snapshot, error)
type ProjectBatchProber func(context.Context, []config.Repository, string, string, int) (map[string]ProbeResult, map[string]error)

type ProjectProber interface {
	Probe(context.Context, config.Repository, string) (ProbeResult, error)
}

type Engine struct {
	Config       config.Config
	Paths        Paths
	Cache        Cache
	Repositories RepositoryProvider
	ScanProject  ProjectScanner
	ScanBatch    ProjectBatchScanner
	Prober       ProjectProber
	ProbeBatch   ProjectBatchProber
	Tracker      tracking.Manager
	Now          func() time.Time
	StartedAt    time.Time

	mutex       sync.RWMutex
	records     map[string]ProjectRecord
	revisions   map[string]uint64
	stages      map[string]string
	refreshing  bool
	scanID      string
	activeAll   bool
	active      map[string]struct{}
	pendingAll  bool
	pending     map[string]struct{}
	hub         *eventHub
	cacheErrors []error
}

func NewEngine(cfg config.Config, paths Paths, cache Cache, repositories RepositoryProvider, scanner ProjectScanner, prober ProjectProber, tracker tracking.Manager) *Engine {
	records, failures := cache.LoadAll()
	if cfg.Path != "" && paths.State != "" {
		assembled := Assemble(records, cfg.Path, paths.State, time.Now().UTC())
		reconciled, err := tracker.ApplyAt(assembled, paths.State)
		if err != nil {
			failures = append(failures, fmt.Errorf("apply cached tracking state: %w", err))
		} else {
			trackingByProject := make(map[string]model.TrackingState, len(reconciled.Projects))
			for _, project := range reconciled.Projects {
				trackingByProject[project.GitHub] = project.TrackingState
			}
			for index := range records {
				if len(records[index].Snapshot.Projects) == 0 {
					continue
				}
				if state, found := trackingByProject[records[index].ProjectID]; found {
					records[index].Snapshot.Projects[0].TrackingState = state
					if state == model.TrackingUntracked {
						records[index].LastProbeAt = time.Now().UTC()
					}
				}
			}
		}
	}
	byID := make(map[string]ProjectRecord, len(records))
	revisions := make(map[string]uint64, len(records))
	stages := make(map[string]string, len(records))
	for _, record := range records {
		byID[record.ProjectID] = record
		revisions[record.ProjectID] = record.Revision
		stages[record.ProjectID] = "cached"
	}
	return &Engine{
		Config: cfg, Paths: paths, Cache: cache, Repositories: repositories,
		ScanProject: scanner, Prober: prober, Tracker: tracker, Now: time.Now,
		StartedAt: time.Now().UTC(), records: byID, revisions: revisions, stages: stages,
		hub: newEventHub(), cacheErrors: failures,
	}
}

func (e *Engine) Snapshot() model.Snapshot {
	e.mutex.RLock()
	records := make([]ProjectRecord, 0, len(e.records))
	for _, record := range e.records {
		records = append(records, record)
	}
	cacheErrors := append([]error{}, e.cacheErrors...)
	e.mutex.RUnlock()
	sort.Slice(records, func(i, j int) bool { return records[i].ProjectID < records[j].ProjectID })
	snapshot := Assemble(records, e.Config.Path, e.Paths.State, e.now())
	for _, cacheError := range cacheErrors {
		snapshot.Warnings = append(snapshot.Warnings, model.ScanError{Stage: "cache", Message: cacheError.Error()})
	}
	snapshot.Summary.Warnings = len(snapshot.Warnings)
	return snapshot
}

func (e *Engine) Projects() []ProjectStatus {
	e.mutex.RLock()
	records := make(map[string]ProjectRecord, len(e.records))
	stages := make(map[string]string, len(e.stages))
	for projectID, record := range e.records {
		records[projectID] = record
		stages[projectID] = e.stages[projectID]
	}
	e.mutex.RUnlock()
	projects := make([]ProjectStatus, 0, len(records))
	for projectID, record := range records {
		if len(record.Snapshot.Projects) == 0 {
			continue
		}
		project := record.Snapshot.Projects[0]
		entry, _, _ := e.Tracker.Entry(e.Config.Path, projectID)
		projects = append(projects, ProjectStatus{
			ProjectID: projectID, Name: project.Name, Path: project.Path,
			Tracking: project.TrackingState, Stage: stages[projectID],
			Revision: record.Revision, UpdatedAt: record.UpdatedAt,
			MutedAt: entry.UntrackedAt, LastProbeAt: entry.LastProbeAt,
		})
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].ProjectID < projects[j].ProjectID })
	return projects
}

func (e *Engine) Status() Status {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return Status{
		Running: true, PID: os.Getpid(), StartedAt: e.StartedAt,
		Refreshing: e.refreshing, ScanID: e.scanID,
		ProjectCount: len(e.records), Socket: e.Paths.Socket,
	}
}

func (e *Engine) Subscribe() (<-chan Event, func()) { return e.hub.Subscribe() }

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
	for _, repository := range repositories {
		if project != "" && project != repository.GitHub && project != repository.Name {
			continue
		}
		if !force && !e.due(repository) {
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
	for _, record := range records {
		if len(record.Snapshot.Projects) == 0 {
			return true
		}
		if record.Snapshot.Projects[0].TrackingState != model.TrackingUntracked {
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

func (e *Engine) refreshProject(ctx context.Context, scanID string, repository config.Repository, force bool) {
	record, cached := e.record(repository.GitHub)
	entry, muted, entryErr := e.Tracker.Entry(e.Config.Path, repository.GitHub)
	if entryErr != nil {
		e.failProject(scanID, repository.GitHub, e.revision(repository.GitHub)+1, entryErr)
		return
	}
	var changedProbe *ProbeResult
	if muted && cached && e.Prober != nil {
		probe, err := e.Prober.Probe(ctx, repository, e.probeAuthor())
		if err != nil {
			e.failProject(scanID, repository.GitHub, record.Revision+1, err)
			return
		}
		now := e.now()
		if entry.ProbeBaseline == "" || entry.ProbeFormat != probe.Format || probe.Combined == entry.ProbeBaseline {
			if err := e.Tracker.UpdateProbe(e.Config.Path, repository.GitHub, probe.Format, probe.Combined, probe.Local, probe.Remote, now); err != nil {
				e.failProject(scanID, repository.GitHub, record.Revision+1, err)
				return
			}
			record.LastProbeAt = now
			record.UpdatedAt = now
			record.Revision++
			record.Stage = "ready"
			if err := e.Cache.Write(record); err != nil {
				e.failProject(scanID, repository.GitHub, record.Revision, err)
				return
			}
			e.storeRecord(record)
			e.publish(EventProjectUpdated, scanID, repository.GitHub, record.Revision, "ready", "muted probe unchanged", pointer(e.Snapshot()))
			return
		}
		changedProbe = &probe
		_ = force
	}

	revision := e.revision(repository.GitHub) + 1
	snapshot, err := e.ScanProject(ctx, repository, true, func(stage string) {
		e.setStage(repository.GitHub, stage)
		eventType := EventProjectLocalReady
		if stage == "github" {
			eventType = EventProjectUpdated
		}
		e.publishProject(eventType, scanID, repository, revision, stage, "", nil)
	})
	if err != nil {
		e.failProject(scanID, repository.GitHub, revision, err)
		return
	}
	e.finishScannedProject(scanID, repository, revision, record, cached, muted, entry, changedProbe, snapshot)
}

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
		previous := record.Snapshot.Projects[0].TrackingState
		record.Snapshot.Projects[0].TrackingState = project.TrackingState
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

func (e *Engine) due(repository config.Repository) bool {
	record, exists := e.record(repository.GitHub)
	if !exists || len(record.Snapshot.Projects) == 0 {
		return true
	}
	if record.Snapshot.Projects[0].TrackingState != model.TrackingUntracked {
		return e.now().Sub(record.UpdatedAt) >= e.Config.Settings.TrackedRefreshInterval
	}
	return record.LastProbeAt.IsZero() || e.now().Sub(record.LastProbeAt) >= e.Config.Settings.UntrackedProbeInterval
}

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

func reactivationReason(entry tracking.Entry, probe ProbeResult) string {
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
