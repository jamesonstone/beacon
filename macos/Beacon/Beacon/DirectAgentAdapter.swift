import Foundation

actor DirectAgentAdapter: AgentClientProtocol {
    let client: CLIClientProtocol
    var latest: BeaconSnapshot?

    init(client: CLIClientProtocol) {
        self.client = client
    }

    func snapshot() async throws -> AgentEvent {
        if latest == nil { latest = try await client.scan() }
        return event(type: "snapshot", snapshot: latest)
    }

    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error> {
        let initial = try await snapshot()
        return AsyncThrowingStream { continuation in
            continuation.yield(initial)
            continuation.finish()
        }
    }

    func refresh(project: String?) async throws -> String {
        latest = try await client.scan()
        return "direct"
    }

    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent {
        try await client.setProjectTracked(github, tracked: tracked)
        latest = try await client.scan()
        return event(type: "tracking_changed", snapshot: latest)
    }

    func notes() async throws -> AgentEvent {
        event(type: "notes", notes: try await client.notes())
    }

    func notesWorkspace() async throws -> AgentEvent {
        let workspace = try await client.notesWorkspace()
        return event(type: "notes_workspace", notes: workspace.active, notesWorkspace: workspace)
    }

    func notes(noteID: String) async throws -> AgentEvent {
        event(type: "notes", notes: try await client.notes(noteID: noteID))
    }

    func setNotes(_ content: String) async throws -> AgentEvent {
        event(type: "notes_updated", notes: try await client.setNotes(content))
    }

    func setNotes(_ content: String, noteID: String) async throws -> AgentEvent {
        let workspace = try await client.setNotes(content, noteID: noteID)
        return event(type: "notes_updated", notes: workspace.active, notesWorkspace: workspace)
    }

    func createNote(_ content: String) async throws -> AgentEvent {
        let workspace = try await client.createNote(content)
        return event(type: "notes_workspace_updated", notes: workspace.active, notesWorkspace: workspace)
    }

    func openNote(_ noteID: String) async throws -> AgentEvent {
        let workspace = try await client.openNote(noteID)
        return event(type: "notes_workspace_updated", notes: workspace.active, notesWorkspace: workspace)
    }

    func closeNote(_ noteID: String) async throws -> AgentEvent {
        let workspace = try await client.closeNote(noteID)
        return event(type: "notes_workspace_updated", notes: workspace.active, notesWorkspace: workspace)
    }

    func deleteNote(_ noteID: String) async throws -> AgentEvent {
        let workspace = try await client.deleteNote(noteID)
        return event(type: "notes_workspace_updated", notes: workspace.active, notesWorkspace: workspace)
    }

    func repositorySync(refresh: Bool) async throws -> AgentEvent {
        event(type: "repository_sync", repositorySync: try await client.repositorySync(refresh: refresh))
    }

    func syncRepositories(_ projectIDs: [String]) async throws -> AgentEvent {
        event(type: "repository_sync", repositorySync: try await client.syncRepositories(projectIDs))
    }

    func status() async throws -> AgentStatusDetails {
        AgentStatusDetails(
            running: true,
            pid: 0,
            startedAt: nil,
            refreshing: false,
            scanID: nil,
            projectCount: latest?.projects.count ?? 0,
            socket: "direct"
        )
    }

    private func event(
        type: String,
        snapshot: BeaconSnapshot? = nil,
        notes: AgentNotes? = nil,
        notesWorkspace: AgentNotesWorkspace? = nil,
        repositorySync: RepositorySyncReport? = nil
    ) -> AgentEvent {
        AgentEvent(
            protocolVersion: 1,
            requestID: nil,
            type: type,
            scanID: nil,
            projectID: nil,
            revision: nil,
            stage: "ready",
            generatedAt: snapshot?.generatedAt ?? "",
            message: nil,
            snapshot: snapshot,
            projects: nil,
            status: nil,
            notes: notes,
            notesWorkspace: notesWorkspace,
            repositorySync: repositorySync
        )
    }
}
