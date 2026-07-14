import Foundation

struct AgentProjectStatus: Codable, Equatable {
    let projectID: String
    let name: String
    let path: String
    let trackingState: String
    let stage: String
    let revision: UInt64
    let updatedAt: String
    let mutedAt: String?
    let lastProbeAt: String?

    enum CodingKeys: String, CodingKey {
        case name, path, stage, revision
        case projectID = "project_id"
        case trackingState = "tracking_state"
        case updatedAt = "updated_at"
        case mutedAt = "muted_at"
        case lastProbeAt = "last_probe_at"
    }
}

struct AgentStatusDetails: Codable, Equatable {
    let running: Bool
    let pid: Int
    let startedAt: String?
    let refreshing: Bool
    let scanID: String?
    let projectCount: Int
    let socket: String

    enum CodingKeys: String, CodingKey {
        case running, pid, refreshing, socket
        case startedAt = "started_at"
        case scanID = "scan_id"
        case projectCount = "project_count"
    }
}

struct AgentNotes: Codable, Equatable {
    let id: String?
    let title: String?
    let content: String
    let path: String
    let createdAt: String?
    let updatedAt: String?
    let openedAt: String?

    init(
        content: String,
        path: String,
        updatedAt: String?,
        id: String? = nil,
        title: String? = nil,
        createdAt: String? = nil,
        openedAt: String? = nil
    ) {
        self.id = id
        self.title = title
        self.content = content
        self.path = path
        self.createdAt = createdAt
        self.updatedAt = updatedAt
        self.openedAt = openedAt
    }

    enum CodingKeys: String, CodingKey {
        case id, title, content, path
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case openedAt = "opened_at"
    }
}

struct AgentNoteTab: Codable, Equatable, Identifiable {
    let id: String
    let title: String
    let path: String?
    let createdAt: String?
    let updatedAt: String?
    let openedAt: String?
    let isOpen: Bool
    let pinned: Bool?

    enum CodingKeys: String, CodingKey {
        case id, title, path, pinned
        case isOpen = "open"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case openedAt = "opened_at"
    }
}

struct AgentNotesWorkspace: Codable, Equatable {
    let version: Int
    let activeID: String
    let openIDs: [String]
    let tabs: [AgentNoteTab]
    let active: AgentNotes?

    enum CodingKeys: String, CodingKey {
        case version, tabs, active
        case activeID = "active_id"
        case openIDs = "open_ids"
    }
}

struct RepositorySyncReport: Codable, Equatable {
    let checkedAt: String
    let fetchAttempted: Bool
    let repositories: [RepositorySyncItem]

    enum CodingKeys: String, CodingKey {
        case repositories
        case checkedAt = "checked_at"
        case fetchAttempted = "fetch_attempted"
    }
}

struct RepositorySyncItem: Codable, Equatable, Identifiable {
    let projectID: String
    let name: String
    let path: String
    let base: String
    let remote: String
    let currentBranch: String?
    let baseWorktree: String?
    let currentAhead: Int
    let currentBehind: Int
    let defaultAhead: Int
    let defaultBehind: Int
    let dirty: Bool
    let detached: Bool
    let needsUpdate: Bool
    let canUpdate: Bool
    let fetched: Bool
    let updated: Bool
    let state: String
    let action: String
    let reason: String
    let error: String?

    var id: String { projectID }

    enum CodingKeys: String, CodingKey {
        case name, path, base, remote, dirty, detached, fetched, updated, state, action, reason, error
        case projectID = "project_id"
        case currentBranch = "current_branch"
        case baseWorktree = "base_worktree"
        case currentAhead = "current_ahead"
        case currentBehind = "current_behind"
        case defaultAhead = "default_ahead"
        case defaultBehind = "default_behind"
        case needsUpdate = "needs_update"
        case canUpdate = "can_update"
    }
}

struct DependencyLimitReport: Codable, Equatable {
    let checkedAt: String
    let dependencies: [DependencyLimit]

    enum CodingKeys: String, CodingKey {
        case dependencies
        case checkedAt = "checked_at"
    }

    var highestUsagePercent: Int {
        dependencies.flatMap(\.buckets).map(\.usagePercent).max() ?? 0
    }

    var hasUsage: Bool {
        dependencies.flatMap(\.buckets).contains { $0.limit > 0 && $0.used > 0 }
    }

    var usageLevel: DependencyUsageLevel {
        DependencyLimitPresentation.level(percent: highestUsagePercent, hasUsage: hasUsage)
    }
}

struct DependencyLimit: Codable, Equatable, Identifiable {
    let name: String
    let buckets: [DependencyLimitBucket]

    var id: String { name }
}

struct DependencyLimitBucket: Codable, Equatable, Identifiable {
    let id: String
    let name: String
    let limit: Int
    let used: Int
    let remaining: Int
    let resetAt: String

    enum CodingKeys: String, CodingKey {
        case id, name, limit, used, remaining
        case resetAt = "reset_at"
    }

    var usagePercent: Int {
        DependencyLimitPresentation.percentage(used: used, limit: limit)
    }
}

enum DependencyUsageLevel: String, Equatable {
    case unmeasured
    case healthy
    case warning
    case critical
}

enum DependencyLimitPresentation {
    static func percentage(used: Int, limit: Int) -> Int {
        guard used > 0, limit > 0 else { return 0 }
        let roundedUp = Int(ceil((Double(used) / Double(limit)) * 100))
        return min(100, max(1, roundedUp))
    }

    static func level(percent: Int, hasUsage: Bool) -> DependencyUsageLevel {
        guard hasUsage, percent > 0 else { return .unmeasured }
        if percent < 50 { return .healthy }
        if percent <= 75 { return .warning }
        return .critical
    }
}

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
    func notes() async throws -> AgentEvent
    func setNotes(_ content: String) async throws -> AgentEvent
    func notesWorkspace() async throws -> AgentEvent
    func notes(noteID: String) async throws -> AgentEvent
    func setNotes(_ content: String, noteID: String) async throws -> AgentEvent
    func createNote(_ content: String) async throws -> AgentEvent
    func openNote(_ noteID: String) async throws -> AgentEvent
    func closeNote(_ noteID: String) async throws -> AgentEvent
    func deleteNote(_ noteID: String) async throws -> AgentEvent
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
    func notes() async throws -> AgentEvent { throw AgentClientError.command("signal notes are unavailable") }
    func setNotes(_ content: String) async throws -> AgentEvent { throw AgentClientError.command("signal notes are unavailable") }
    func notesWorkspace() async throws -> AgentEvent { throw AgentClientError.command("signal note tabs are unavailable") }
    func notes(noteID: String) async throws -> AgentEvent {
        guard noteID == "general" else { throw AgentClientError.command("signal note tabs are unavailable") }
        return try await notes()
    }
    func setNotes(_ content: String, noteID: String) async throws -> AgentEvent {
        guard noteID == "general" else { throw AgentClientError.command("signal note tabs are unavailable") }
        return try await setNotes(content)
    }
    func createNote(_ content: String) async throws -> AgentEvent { throw AgentClientError.command("signal note tabs are unavailable") }
    func openNote(_ noteID: String) async throws -> AgentEvent { throw AgentClientError.command("signal note tabs are unavailable") }
    func closeNote(_ noteID: String) async throws -> AgentEvent { throw AgentClientError.command("signal note tabs are unavailable") }
    func deleteNote(_ noteID: String) async throws -> AgentEvent { throw AgentClientError.command("signal note deletion is unavailable") }
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
