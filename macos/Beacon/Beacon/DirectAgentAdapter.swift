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

    func setNotes(_ content: String) async throws -> AgentEvent {
        event(type: "notes_updated", notes: try await client.setNotes(content))
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
            repositorySync: repositorySync
        )
    }
}
