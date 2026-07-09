import Foundation

struct BeaconSnapshot: Codable, Equatable {
    let schemaVersion: Int
    let generatedAt: String
    let configPath: String
    let refresh: [RefreshResult]
    let summary: SnapshotSummary
    let groups: LaneGroups
    let lanes: [WorkLane]
    let errors: [ScanError]

    enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case generatedAt = "generated_at"
        case configPath = "config_path"
        case refresh, summary, groups, lanes, errors
    }
}

struct SnapshotSummary: Codable, Equatable {
    let total: Int
    let reviewReady: Int
    let needsAction: Int
    let waiting: Int
    let idle: Int
    let errors: Int

    enum CodingKeys: String, CodingKey {
        case total, waiting, idle, errors
        case reviewReady = "review_ready"
        case needsAction = "needs_action"
    }
}

struct LaneGroups: Codable, Equatable {
    let ready: [String]
    let action: [String]
    let waiting: [String]
    let idle: [String]
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

struct WorkLane: Codable, Equatable, Identifiable {
    let id: String
    let repository: String
    let github: String
    let base: String
    let branch: String
    let worktree: WorktreeDetails?
    let pullRequest: PullRequestDetails?
    let signals: LaneSignals
    let reviewReady: Bool
    let nextAction: String
    let reasons: [String]
    let warnings: [String]
    let blockers: [String]
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id, repository, github, base, branch, worktree, signals, reasons, warnings, blockers
        case pullRequest = "pull_request"
        case reviewReady = "review_ready"
        case nextAction = "next_action"
        case updatedAt = "updated_at"
    }
}

struct WorktreeDetails: Codable, Equatable {
    let path: String
    let headOID: String
    let upstream: String?
    let staged: Int
    let unstaged: Int
    let untracked: Int
    let conflicted: Int
    let ahead: Int
    let behind: Int
    let aheadBase: Int
    let behindBase: Int
    let detached: Bool
    let locked: Bool
    let prunable: Bool
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case path, upstream, staged, unstaged, untracked, conflicted, ahead, behind, detached, locked, prunable
        case headOID = "head_oid"
        case aheadBase = "ahead_base"
        case behindBase = "behind_base"
        case updatedAt = "updated_at"
    }
}

struct PullRequestDetails: Codable, Equatable {
    let number: Int
    let title: String
    let url: String
    let headRefName: String
    let headRefOID: String
    let baseRefName: String
    let isDraft: Bool
    let updatedAt: String
    let reviewDecision: String?
    let mergeStateStatus: String?
    let mergeable: String?
    let ciState: String

    enum CodingKeys: String, CodingKey {
        case number, title, url, mergeable
        case headRefName = "head_ref_name"
        case headRefOID = "head_ref_oid"
        case baseRefName = "base_ref_name"
        case isDraft = "is_draft"
        case updatedAt = "updated_at"
        case reviewDecision = "review_decision"
        case mergeStateStatus = "merge_state_status"
        case ciState = "ci_state"
    }
}

struct LaneSignals: Codable, Equatable {
    let worktree: String
    let publication: String
    let pullRequest: String
    let ci: String
    let review: String
    let merge: String
    let freshness: String

    enum CodingKeys: String, CodingKey {
        case worktree, publication, ci, review, merge, freshness
        case pullRequest = "pull_request"
    }
}
