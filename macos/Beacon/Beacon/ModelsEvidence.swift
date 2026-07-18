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
    let body: String?
    let bodyTruncated: Bool?
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
        case number, title, body, url, mergeable, checks, feedback
        case bodyTruncated = "body_truncated"
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
    let threads: [ReviewThreadDetails]?
    let threadsTruncated: Bool?

    enum CodingKeys: String, CodingKey {
        case comments, reviews, approvals
        case changesRequested = "changes_requested"
        case unresolvedThreads = "unresolved_threads"
        case threads
        case threadsTruncated = "threads_truncated"
    }
}

struct ReviewThreadDetails: Codable, Equatable, Identifiable {
    let id: String
    let path: String
    let line: Int?
    let originalLine: Int?
    let outdated: Bool
    let comments: [ReviewCommentDetails]
    let commentsTruncated: Bool

    enum CodingKeys: String, CodingKey {
        case id, path, line, outdated, comments
        case originalLine = "original_line"
        case commentsTruncated = "comments_truncated"
    }

    var displayLine: Int? { line ?? originalLine }
}

struct ReviewCommentDetails: Codable, Equatable, Identifiable {
    let id: String
    let author: String
    let body: String
    let bodyTruncated: Bool
    let url: String
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id, author, body, url
        case bodyTruncated = "body_truncated"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct IssueDetails: Codable, Equatable {
    let number: Int
    let title: String
    let body: String?
    let bodyTruncated: Bool?
    let url: String
    let labels: [String]
    let assignees: [String]
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case number, title, body, url, labels, assignees
        case bodyTruncated = "body_truncated"
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
