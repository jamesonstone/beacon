import Foundation

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
    let checks: CheckSummary
    let feedback: FeedbackSummary
    let closingIssues: [IssueDetails]

    enum CodingKeys: String, CodingKey {
        case number, title, url, mergeable, checks, feedback
        case headRefName = "head_ref_name"
        case headRefOID = "head_ref_oid"
        case baseRefName = "base_ref_name"
        case isDraft = "is_draft"
        case updatedAt = "updated_at"
        case reviewDecision = "review_decision"
        case mergeStateStatus = "merge_state_status"
        case ciState = "ci_state"
        case closingIssues = "closing_issues"
    }
}

struct CheckSummary: Codable, Equatable {
    let total: Int
    let success: Int
    let pending: Int
    let failure: Int
    let skipped: Int
    let unknown: Int
}

struct FeedbackSummary: Codable, Equatable {
    let comments: Int
    let reviews: Int
    let approvals: Int
    let changesRequested: Int
    let unresolvedThreads: Int

    enum CodingKeys: String, CodingKey {
        case comments, reviews, approvals
        case changesRequested = "changes_requested"
        case unresolvedThreads = "unresolved_threads"
    }
}

struct IssueDetails: Codable, Equatable {
    let number: Int
    let title: String
    let url: String
    let labels: [String]
    let assignees: [String]
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case number, title, url, labels, assignees
        case updatedAt = "updated_at"
    }
}

struct ProgressDetails: Codable, Equatable {
    let source: String
    let featureID: String
    let feature: String
    let phase: String
    let summary: String
    let path: String

    enum CodingKeys: String, CodingKey {
        case source, feature, phase, summary, path
        case featureID = "feature_id"
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
    let issue: String

    enum CodingKeys: String, CodingKey {
        case worktree, publication, ci, review, merge, freshness, issue
        case pullRequest = "pull_request"
    }
}
