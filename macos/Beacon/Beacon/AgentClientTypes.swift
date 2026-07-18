import Foundation

struct AgentEvent: Codable, Equatable {
    let protocolVersion: Int
    let requestID: String?
    let type: String
    let scanID: String?
    let projectID: String?
    let revision: UInt64?
    let stage: String?
    let generatedAt: String
    let message: String?
    let snapshot: BeaconSnapshot?
    let projects: [AgentProjectStatus]?
    let status: AgentStatusDetails?
    let notes: AgentNotes?
    let notesWorkspace: AgentNotesWorkspace?
    let repositorySync: RepositorySyncReport?

    init(
        protocolVersion: Int,
        requestID: String?,
        type: String,
        scanID: String?,
        projectID: String?,
        revision: UInt64?,
        stage: String?,
        generatedAt: String,
        message: String?,
        snapshot: BeaconSnapshot?,
        projects: [AgentProjectStatus]?,
        status: AgentStatusDetails?,
        notes: AgentNotes?,
        notesWorkspace: AgentNotesWorkspace? = nil,
        repositorySync: RepositorySyncReport? = nil
    ) {
        self.protocolVersion = protocolVersion
        self.requestID = requestID
        self.type = type
        self.scanID = scanID
        self.projectID = projectID
        self.revision = revision
        self.stage = stage
        self.generatedAt = generatedAt
        self.message = message
        self.snapshot = snapshot
        self.projects = projects
        self.status = status
        self.notes = notes
        self.notesWorkspace = notesWorkspace
        self.repositorySync = repositorySync
    }

    enum CodingKeys: String, CodingKey {
        case type, revision, stage, message, snapshot, projects, status, notes
        case notesWorkspace = "notes_workspace"
        case repositorySync = "repository_sync"
        case protocolVersion = "protocol_version"
        case requestID = "request_id"
        case scanID = "scan_id"
        case projectID = "project_id"
        case generatedAt = "generated_at"
    }
}

protocol AgentClientProtocol {
    func snapshot() async throws -> AgentEvent
    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error>
    func refresh(project: String?) async throws -> String
    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent
    func status() async throws -> AgentStatusDetails
    func setLaneAttention(_ id: String, state: String) async throws -> AgentEvent
    func setLanePinned(_ id: String, pinned: Bool) async throws -> AgentEvent
    func setLaneNote(_ id: String, note: String) async throws -> AgentEvent
    func addLaneTag(_ id: String, tag: String) async throws -> AgentEvent
    func removeLaneTag(_ id: String, tag: String) async throws -> AgentEvent
    func markLaneSeen(_ id: String) async throws -> AgentEvent
    func addManualLane(_ title: String) async throws -> AgentEvent
    func reorderLanes(_ ids: [String]) async throws -> AgentEvent
    func notes() async throws -> AgentEvent
    func setNotes(_ content: String) async throws -> AgentEvent
    func notesWorkspace() async throws -> AgentEvent
    func notes(noteID: String) async throws -> AgentEvent
    func setNotes(_ content: String, noteID: String) async throws -> AgentEvent
    func createNote(_ content: String) async throws -> AgentEvent
    func openNote(_ noteID: String) async throws -> AgentEvent
    func closeNote(_ noteID: String) async throws -> AgentEvent
    func deleteNote(_ noteID: String) async throws -> AgentEvent
    func setNotePinned(_ noteID: String, pinned: Bool) async throws -> AgentEvent
    func reorderPinnedNotes(_ noteIDs: [String]) async throws -> AgentEvent
    func repositorySync(refresh: Bool) async throws -> AgentEvent
    func syncRepositories(_ projectIDs: [String]) async throws -> AgentEvent
}

extension AgentClientProtocol {
    func setLaneAttention(_ id: String, state: String) async throws -> AgentEvent { throw AgentClientError.command("lane attention is unavailable") }
    func setLanePinned(_ id: String, pinned: Bool) async throws -> AgentEvent { throw AgentClientError.command("lane pinning is unavailable") }
    func setLaneNote(_ id: String, note: String) async throws -> AgentEvent { throw AgentClientError.command("lane notes are unavailable") }
    func addLaneTag(_ id: String, tag: String) async throws -> AgentEvent { throw AgentClientError.command("lane tags are unavailable") }
    func removeLaneTag(_ id: String, tag: String) async throws -> AgentEvent { throw AgentClientError.command("lane tags are unavailable") }
    func markLaneSeen(_ id: String) async throws -> AgentEvent { throw AgentClientError.command("lane acknowledgement is unavailable") }
    func addManualLane(_ title: String) async throws -> AgentEvent { throw AgentClientError.command("manual lanes are unavailable") }
    func reorderLanes(_ ids: [String]) async throws -> AgentEvent { throw AgentClientError.command("lane reordering is unavailable") }
    func notes() async throws -> AgentEvent { throw AgentClientError.command("Notes are unavailable") }
    func setNotes(_ content: String) async throws -> AgentEvent { throw AgentClientError.command("Notes are unavailable") }
    func notesWorkspace() async throws -> AgentEvent { throw AgentClientError.command("Note tabs are unavailable") }
    func notes(noteID: String) async throws -> AgentEvent {
        guard noteID == "general" else { throw AgentClientError.command("Note tabs are unavailable") }
        return try await notes()
    }
    func setNotes(_ content: String, noteID: String) async throws -> AgentEvent {
        guard noteID == "general" else { throw AgentClientError.command("Note tabs are unavailable") }
        return try await setNotes(content)
    }
    func createNote(_ content: String) async throws -> AgentEvent { throw AgentClientError.command("Note tabs are unavailable") }
    func openNote(_ noteID: String) async throws -> AgentEvent { throw AgentClientError.command("Note tabs are unavailable") }
    func closeNote(_ noteID: String) async throws -> AgentEvent { throw AgentClientError.command("Note tabs are unavailable") }
    func deleteNote(_ noteID: String) async throws -> AgentEvent { throw AgentClientError.command("Note deletion is unavailable") }
    func setNotePinned(_ noteID: String, pinned: Bool) async throws -> AgentEvent { throw AgentClientError.command("note pinning is unavailable") }
    func reorderPinnedNotes(_ noteIDs: [String]) async throws -> AgentEvent { throw AgentClientError.command("note reordering is unavailable") }
    func repositorySync(refresh: Bool) async throws -> AgentEvent { throw AgentClientError.command("repository sync is unavailable") }
    func syncRepositories(_ projectIDs: [String]) async throws -> AgentEvent { throw AgentClientError.command("repository sync is unavailable") }
}

enum AgentClientError: LocalizedError {
    case connection(String)
    case invalidResponse(String)
    case command(String)

    var errorDescription: String? {
        switch self {
        case .connection(let message): "Beacon background agent is unavailable: \(message)"
        case .invalidResponse(let message): "Beacon agent returned invalid data: \(message)"
        case .command(let message): "Beacon agent command failed: \(message)"
        }
    }
}
