import Foundation

@MainActor
extension AppState {
    func listenForAgent() async {
        while !Task.isCancelled {
            do {
                let stream = try await agent.subscribe()
                agentAvailable = true
                lastError = nil
                await loadNotes()
                await checkRepositorySync(refresh: false)
                for try await event in stream {
                    guard !Task.isCancelled else { return }
                    apply(event)
                    if event.type == "snapshot" || event.type == "heartbeat" {
                        reconcile(try await agent.status())
                    }
                }
            } catch {
                agentAvailable = false
                lastError = error.localizedDescription
            }
            try? await Task.sleep(for: .seconds(2))
        }
    }

    func apply(_ event: AgentEvent) {
        guard event.protocolVersion == 1 else {
            lastError = "Beacon agent returned invalid data: unsupported protocol \(event.protocolVersion)"
            return
        }
        if activeScanID == nil, let eventScanID = event.scanID,
           !eventScanID.isEmpty, event.type != "scan_completed" {
            activeScanID = eventScanID
            isScanning = true
        }
        if let activeScanID, let eventScanID = event.scanID,
           !eventScanID.isEmpty, eventScanID != activeScanID,
           event.type != "heartbeat" {
            return
        }
        if let projectID = event.projectID, let revision = event.revision {
            guard revision >= revisions[projectID, default: 0] else { return }
            revisions[projectID] = revision
            if let existing = projectStatuses[projectID] {
                projectStatuses[projectID] = AgentProjectStatus(
                    projectID: existing.projectID,
                    name: existing.name,
                    path: existing.path,
                    trackingState: existing.trackingState,
                    stage: event.stage ?? existing.stage,
                    revision: revision,
                    updatedAt: event.generatedAt,
                    mutedAt: existing.mutedAt,
                    lastProbeAt: existing.lastProbeAt
                )
            }
        }
        for status in event.projects ?? [] {
            guard status.revision >= revisions[status.projectID, default: 0] else { continue }
            revisions[status.projectID] = status.revision
            projectStatuses[status.projectID] = status
        }
        if let latest = event.snapshot {
            guard latest.schemaVersion == 3 else {
                lastError = "Beacon CLI returned invalid JSON: unsupported schema version \(latest.schemaVersion)"
                return
            }
            snapshot = latest
            if event.type != "project_failed" {
                lastError = nil
            }
        }
        if let workspace = event.notesWorkspace {
            notesUseFallback = false
            applyNotesWorkspace(workspace)
        } else if let notes = event.notes {
            applyNotesDocument(notes, noteID: notes.id ?? activeNoteID)
        }
        if event.type == "project_failed" {
            lastError = event.message ?? "Project refresh failed — showing previous result"
        }
        if event.type == "scan_completed", activeScanID == nil || activeScanID == event.scanID {
            isScanning = false
            activeScanID = nil
        }
    }

    func reconcile(_ status: AgentStatusDetails) {
        agentAvailable = status.running
        isScanning = status.refreshing
        activeScanID = status.refreshing ? status.scanID : nil
    }
}
