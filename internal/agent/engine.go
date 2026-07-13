package agent

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
	"github.com/jamesonstone/beacon/internal/workset"
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
	WorkingSet   *workset.Manager
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
	if e.WorkingSet != nil {
		reconciled, err := e.WorkingSet.Reconcile(snapshot)
		if err != nil {
			snapshot.Warnings = append(snapshot.Warnings, model.ScanError{Stage: "working-set", Message: err.Error()})
			snapshot.Summary.Warnings = len(snapshot.Warnings)
		} else {
			snapshot = reconciled
		}
	}
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
