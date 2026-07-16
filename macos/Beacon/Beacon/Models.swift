import Foundation

struct ExternalActivitySnapshot: Codable, Equatable {
    let version: Int
    let records: [ExternalActivityRecord]
    let nextExpiry: String?

    enum CodingKeys: String, CodingKey {
        case version, records
        case nextExpiry = "next_expiry"
    }

    static let empty = ExternalActivitySnapshot(version: 1, records: [], nextExpiry: nil)
}
struct ExternalActivityRecord: Codable, Equatable {
    let provider: String
    let state: String
    let sessionKey: String
    let projectID: String
    let laneID: String?
    let observedAt: String
    let expiresAt: String

    enum CodingKeys: String, CodingKey {
        case provider, state
        case sessionKey = "session_key"
        case projectID = "project_id"
        case laneID = "lane_id"
        case observedAt = "observed_at"
        case expiresAt = "expires_at"
    }
}

struct IntegrationHealthStatus: Codable, Equatable, Identifiable {
    var id: String { provider }
    let provider: String
    let state: String
    let settingsPath: String
    let message: String?

    enum CodingKeys: String, CodingKey {
        case provider, state, message
        case settingsPath = "settings_path"
    }
}

struct ExternalActivityChip: Equatable {
    let label: String
    let state: String
    let sessionCount: Int
}

enum ExternalActivityPresentation {
    static func chip(for records: [ExternalActivityRecord]) -> ExternalActivityChip? {
        guard !records.isEmpty else { return nil }
        let state = records.map(\.state).max { priority($0) < priority($1) } ?? "turn_finished"
        let providers = Set(records.map(\.provider))
        let provider: String
        if providers.count > 1 {
            provider = "Agents"
        } else if providers.first == "claude-code" {
            provider = "Claude Code"
        } else {
            provider = "Codex"
        }
        let stateLabel: String
        switch state {
        case "needs_attention": stateLabel = "Needs attention"
        case "working": stateLabel = "Working"
        default: stateLabel = "Turn finished"
        }
        var components = [provider, stateLabel]
        if records.count > 1 {
            components.append(String(records.count))
        }
        return ExternalActivityChip(
            label: components.joined(separator: " · "),
            state: state,
            sessionCount: records.count
        )
    }

    private static func priority(_ state: String) -> Int {
        switch state {
        case "needs_attention": return 3
        case "working": return 2
        case "turn_finished": return 1
        default: return 0
        }
    }
}

struct BeaconSnapshot: Codable, Equatable {
    let schemaVersion: Int
    let generatedAt: String
    let configPath: String
    let tracking: TrackingDetails?
    var workingSet: WorkingSetGroups? = nil
    let refresh: [RefreshResult]
    let summary: SnapshotSummary
    let groups: LaneGroups
    let projects: [BeaconProject]
    let lanes: [WorkLane]
    let errors: [ScanError]

    enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case generatedAt = "generated_at"
        case configPath = "config_path"
        case refresh, summary, groups, projects, lanes, errors, tracking
        case workingSet = "working_set"
    }
}

struct SnapshotSummary: Codable, Equatable {
    let projects: Int
    let trackedProjects: Int?
    let untrackedProjects: Int?
    var followingProjects: Int? = nil
    var recentProjects: Int? = nil
    var quietProjects: Int? = nil
    let total: Int
    let reviewReady: Int
    let needsAction: Int
    let waiting: Int
    let idle: Int
    let errors: Int
    let openIssues: Int
    let unresolvedFeedback: Int
    var activeLanes: Int? = nil
    var recentLanes: Int? = nil
    var parkedLanes: Int? = nil

    enum CodingKeys: String, CodingKey {
        case projects, total, waiting, idle, errors
        case trackedProjects = "tracked_projects"
        case untrackedProjects = "untracked_projects"
        case followingProjects = "following_projects"
        case recentProjects = "recent_projects"
        case quietProjects = "quiet_projects"
        case reviewReady = "review_ready"
        case needsAction = "needs_action"
        case openIssues = "open_issues"
        case unresolvedFeedback = "unresolved_feedback"
        case activeLanes = "active_lanes"
        case recentLanes = "recent_lanes"
        case parkedLanes = "parked_lanes"
    }
}

struct WorkingSetGroups: Codable, Equatable {
    let path: String
    let active: [String]
    let waiting: [String]
    let recent: [String]
    let parked: [String]
}

struct LaneGroups: Codable, Equatable {
    let ready: [String]
    let action: [String]
    let waiting: [String]
    let idle: [String]
    let untracked: [String]?
}

struct TrackingDetails: Codable, Equatable {
    let path: String
    let autoReactivated: [String]

    enum CodingKeys: String, CodingKey {
        case path
        case autoReactivated = "auto_reactivated"
    }
}

struct RefreshResult: Codable, Equatable {
    let repository: String
    let attempted: Bool
    let refreshed: Bool
    let at: String?
    let error: String?
}

struct ScanError: Codable, Equatable, Identifiable {
    var id: String { "\(repository ?? "global"):\(stage):\(message)" }
    let repository: String?
    let stage: String
    let message: String
}

struct BeaconProject: Codable, Equatable, Identifiable {
    var id: String { github }
    let name: String
    let path: String
    let github: String
    let base: String
    let remote: String
    let trackingState: String?
    var followState: String? = nil
    var lastActivityAt: String? = nil
    var activityReason: String? = nil
    let progress: ProgressDetails?
    let laneIDs: [String]
    let errors: [ScanError]

    enum CodingKeys: String, CodingKey {
        case name, path, github, base, remote, progress, errors
        case trackingState = "tracking_state"
        case followState = "follow_state"
        case lastActivityAt = "last_activity_at"
        case activityReason = "activity_reason"
        case laneIDs = "lane_ids"
    }

    var effectiveFollowState: String {
        if let followState, !followState.isEmpty { return followState }
        return trackingState == "untracked" ? "quiet" : "following"
    }

    var isFollowed: Bool { effectiveFollowState == "following" }
    var isTracked: Bool { isFollowed }
}

struct WorkLane: Codable, Equatable, Identifiable {
    let id: String
    let repository: String
    let github: String
    let base: String
    let branch: String
    let worktree: WorktreeDetails?
    let pullRequest: PullRequestDetails?
    let issue: IssueDetails?
    let progress: ProgressDetails?
    let signals: LaneSignals
    let reviewReady: Bool
    let nextAction: String
    let reasons: [String]
    let warnings: [String]
    let blockers: [String]
    let updatedAt: String
    var attention: LaneAttentionDetails? = nil
    var checkoutWarning: CheckoutWarningDetails? = nil

    enum CodingKeys: String, CodingKey {
        case id, repository, github, base, branch, worktree, issue, progress, signals, reasons, warnings, blockers, attention
        case pullRequest = "pull_request"
        case reviewReady = "review_ready"
        case nextAction = "next_action"
        case updatedAt = "updated_at"
        case checkoutWarning = "checkout_warning"
    }
}

struct CheckoutWarningDetails: Codable, Equatable {
    let kind: String
    let severity: String
    let pullRequestNumber: Int
    let pullRequestURL: String?
    let branch: String
    let base: String
    let mergedAt: String
    let confirmedAt: String
    let message: String

    enum CodingKeys: String, CodingKey {
        case kind, severity, branch, base, message
        case pullRequestNumber = "pull_request_number"
        case pullRequestURL = "pull_request_url"
        case mergedAt = "merged_at"
        case confirmedAt = "confirmed_at"
    }
}

struct LaneAttentionDetails: Codable, Equatable {
    let state: String
    let pinned: Bool
    let manual: Bool
    let title: String?
    let tags: [String]?
    let note: String?
    let noteUpdatedAt: String?
    let noteStale: Bool
    let lastSeenAt: String?
    let delta: String
    let reactivationReason: String?

    enum CodingKeys: String, CodingKey {
        case state, pinned, manual, title, tags, note, delta
        case noteUpdatedAt = "note_updated_at"
        case noteStale = "note_stale"
        case lastSeenAt = "last_seen_at"
        case reactivationReason = "reactivation_reason"
    }
}
