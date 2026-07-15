package agent

import (
	"context"

	"github.com/jamesonstone/beacon/internal/config"
)

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
	e.finishScannedProject(ctx, scanID, repository, revision, record, cached, muted, entry, changedProbe, snapshot)
}
