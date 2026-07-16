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
