package agent

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/githubscan"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/tracking"
)

type batchProjectState struct {
	repository   config.Repository
	record       ProjectRecord
	cached       bool
	entry        tracking.Entry
	muted        bool
	changedProbe *ProbeResult
	revision     uint64
}

type unchangedBatchProbe struct {
	state *batchProjectState
	probe ProbeResult
}

func (e *Engine) runCollectedBatch(
	ctx context.Context,
	scanID string,
	repositories map[string]config.Repository,
	force bool,
) {
	states := make(map[string]*batchProjectState, len(repositories))
	probeRepositories := make([]config.Repository, 0, len(repositories))
	scanRepositories := make([]config.Repository, 0, len(repositories))
	entries, err := e.Tracker.Entries(e.Config.Path)
	if err != nil {
		for _, projectID := range sortedRepositoryIDs(repositories) {
			e.failProject(scanID, projectID, e.revision(projectID)+1, err)
		}
		return
	}
	for _, projectID := range sortedRepositoryIDs(repositories) {
		repository := repositories[projectID]
		record, cached := e.record(projectID)
		entry, muted := entries[projectID]
		state := &batchProjectState{
			repository: repository, record: record, cached: cached,
			entry: entry, muted: muted, revision: e.revision(projectID) + 1,
		}
		states[projectID] = state
		if muted && cached && !force {
			probeRepositories = append(probeRepositories, repository)
			continue
		}
		scanRepositories = append(scanRepositories, repository)
	}

	unchanged := make([]unchangedBatchProbe, 0, len(probeRepositories))
	if len(probeRepositories) > 0 {
		probes, failures := e.ProbeBatch(
			ctx, probeRepositories, string(e.Config.Settings.GitHubScope),
			e.probeAuthor(), e.Config.Settings.MaxParallel,
		)
		for _, repository := range probeRepositories {
			state := states[repository.GitHub]
			if err := failures[repository.GitHub]; err != nil {
				e.failProject(scanID, repository.GitHub, state.revision, err)
				continue
			}
			probe, found := probes[repository.GitHub]
			if !found {
				e.failProject(scanID, repository.GitHub, state.revision, errors.New("batch probe returned no project evidence"))
				continue
			}
			if state.entry.ProbeBaseline == "" || state.entry.ProbeFormat != probe.Format || state.entry.ProbeBaseline == probe.Combined {
				unchanged = append(unchanged, unchangedBatchProbe{state: state, probe: probe})
				continue
			}
			state.changedProbe = &probe
			scanRepositories = append(scanRepositories, repository)
		}
	}
	if len(unchanged) > 0 {
		now := e.now()
		updates := make([]tracking.ProbeUpdate, 0, len(unchanged))
		for _, result := range unchanged {
			updates = append(updates, tracking.ProbeUpdate{
				GitHub: result.state.repository.GitHub, Format: result.probe.Format,
				Baseline: result.probe.Combined, Local: result.probe.Local,
				Remote: result.probe.Remote, At: now,
			})
		}
		if err := e.Tracker.UpdateProbes(e.Config.Path, updates); err != nil {
			for _, result := range unchanged {
				e.failProject(scanID, result.state.repository.GitHub, result.state.revision, err)
			}
		} else {
			for _, result := range unchanged {
				e.finishUnchangedProbe(scanID, result.state, now)
			}
		}
	}

	if len(scanRepositories) == 0 {
		return
	}
	sort.Slice(scanRepositories, func(i, j int) bool {
		return scanRepositories[i].GitHub < scanRepositories[j].GitHub
	})
	scanContext := ctx
	followed := make([]string, 0, len(scanRepositories))
	for _, repository := range scanRepositories {
		state := states[repository.GitHub]
		if state != nil && (!state.muted || force && len(scanRepositories) == 1) {
			followed = append(followed, repository.GitHub)
		}
	}
	if len(followed) > 0 {
		scanContext = githubscan.WithInactivePullRequestRepositories(scanContext, followed)
	}
	snapshots, err := e.ScanBatch(scanContext, scanRepositories, force, func(projectID, stage string) {
		state := states[projectID]
		if state == nil {
			return
		}
		e.setStage(projectID, stage)
		eventType := EventProjectLocalReady
		if stage == "github" {
			eventType = EventProjectUpdated
		}
		e.publishProject(eventType, scanID, state.repository, state.revision, stage, "", nil)
	})
	if err != nil {
		for _, repository := range scanRepositories {
			e.failProject(scanID, repository.GitHub, states[repository.GitHub].revision, err)
		}
		return
	}
	for _, repository := range scanRepositories {
		state := states[repository.GitHub]
		snapshot, found := snapshots[repository.GitHub]
		if !found {
			e.failProject(scanID, repository.GitHub, state.revision, errors.New("batch scan returned no project snapshot"))
			continue
		}
		e.finishScannedProject(
			ctx, scanID, repository, state.revision, state.record, state.cached,
			state.muted, state.entry, state.changedProbe, snapshot,
		)
	}
}

func (e *Engine) finishUnchangedProbe(scanID string, state *batchProjectState, now time.Time) {
	state.record.LastProbeAt = now
	state.record.UpdatedAt = now
	state.record.Revision++
	state.record.Stage = "ready"
	if err := e.Cache.Write(state.record); err != nil {
		e.failProject(scanID, state.repository.GitHub, state.record.Revision, err)
		return
	}
	e.storeRecord(state.record)
	e.publish(
		EventProjectUpdated, scanID, state.repository.GitHub, state.record.Revision,
		"ready", "muted batch probe unchanged", pointer(e.Snapshot()),
	)
}

func (e *Engine) finishScannedProject(
	ctx context.Context,
	scanID string,
	repository config.Repository,
	revision uint64,
	record ProjectRecord,
	cached, muted bool,
	entry tracking.Entry,
	changedProbe *ProbeResult,
	snapshot model.Snapshot,
) {
	if e.revision(repository.GitHub) >= revision {
		e.publish(EventProjectUpdated, scanID, repository.GitHub, e.revision(repository.GitHub), "ready", "superseded scan result ignored", pointer(e.Snapshot()))
		return
	}
	if collectionErr := snapshotCollectionError(snapshot); collectionErr != nil {
		if !cached && len(snapshot.Projects) == 1 && snapshot.Projects[0].GitHub == repository.GitHub {
			failed := ProjectRecord{
				Version: CacheVersion, ProjectID: repository.GitHub, Revision: revision,
				Stage: "failed", UpdatedAt: e.now(), Snapshot: snapshot,
			}
			if err := e.Cache.Write(failed); err == nil {
				e.storeRecord(failed)
			}
		}
		e.failProject(scanID, repository.GitHub, revision, collectionErr)
		return
	}
	if len(snapshot.Projects) != 1 || snapshot.Projects[0].GitHub != repository.GitHub {
		e.failProject(scanID, repository.GitHub, revision, errors.New("project scan returned mismatched identity"))
		return
	}

	reason := ""
	lastProbeAt := record.LastProbeAt
	if muted && changedProbe != nil {
		now := e.now()
		reason = projectActivityReason(entry, *changedProbe)
		if err := e.Tracker.RecordActivity(e.Config.Path, repository.GitHub, reason, now); err != nil {
			e.failProject(scanID, repository.GitHub, revision, err)
			return
		}
		if err := e.Tracker.UpdateProbe(
			e.Config.Path, repository.GitHub, changedProbe.Format,
			changedProbe.Combined, changedProbe.Local, changedProbe.Remote, now,
		); err != nil {
			e.failProject(scanID, repository.GitHub, revision, err)
			return
		}
		snapshot.Projects[0].FollowState = model.FollowRecent
		snapshot.Projects[0].LastActivityAt = now
		snapshot.Projects[0].ActivityReason = reason
		lastProbeAt = now
	}
	confirmations := e.enrichCheckoutWarnings(ctx, scanID, repository, record, cached, !muted, &snapshot)
	updated := ProjectRecord{
		Version: CacheVersion, ProjectID: repository.GitHub, Revision: revision,
		Stage: "ready", UpdatedAt: e.now(), LastProbeAt: lastProbeAt,
		CheckoutConfirmations: confirmations, Snapshot: snapshot,
	}
	if err := e.Cache.Write(updated); err != nil {
		e.failProject(scanID, repository.GitHub, revision, err)
		return
	}
	e.storeRecord(updated)
	e.publish(EventProjectUpdated, scanID, repository.GitHub, revision, "ready", reason, pointer(e.Snapshot()))
}

func sortedRepositoryIDs(repositories map[string]config.Repository) []string {
	identifiers := make([]string, 0, len(repositories))
	for identifier := range repositories {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)
	return identifiers
}
