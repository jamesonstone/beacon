import Foundation

struct HyperliteWorkScan: Codable, Equatable {
    let schemaVersion: Int
    let generatedAt: Date
    let summary: HyperliteWorkSummary
    let items: [HyperliteWorkItem]
    let errors: [HyperliteDiagnostic]
    let warnings: [HyperliteDiagnostic]

    enum CodingKeys: String, CodingKey {
        case summary, items, errors, warnings
        case schemaVersion = "schema_version"
        case generatedAt = "generated_at"
    }
}

struct HyperliteWorkSummary: Codable, Equatable {
    let projects: Int
    let activeProjects: Int
    let workItems: Int
    let idleProjects: Int
    let unknownProjects: Int

    enum CodingKeys: String, CodingKey {
        case projects, workItems = "work_items", idleProjects = "idle_projects", unknownProjects = "unknown_projects"
        case activeProjects = "active_projects"
    }
}

struct HyperliteWorkItem: Codable, Equatable, Identifiable {
    let repository: String
    let github: String
    let repositoryPath: String
    let branch: String
    let base: String
    let state: String
    let publication: String
    let nextAction: String
    let updatedAt: Date?
    let pullRequest: HyperlitePullRequest?

    var id: String { "\(github):\(branch):\(repositoryPath)" }

    enum CodingKeys: String, CodingKey {
        case repository, github, branch, base, state, publication
        case repositoryPath = "repository_path"
        case nextAction = "next_action"
        case updatedAt = "updated_at"
        case pullRequest = "pull_request"
    }

    var title: String {
        if let pullRequest { return "PR #\(pullRequest.number) · \(branch)" }
        return branch.isEmpty ? base : branch
    }

    var needsAttention: Bool {
        ["conflict", "ci_failed", "feedback", "dirty", "unpublished"].contains(state)
    }
}

struct HyperlitePullRequest: Codable, Equatable {
    let number: Int
    let title: String
    let url: String
    let draft: Bool
    let ci: String
    let review: String
}

struct HyperliteDiagnostic: Codable, Equatable {
    let repository: String
    let stage: String
    let message: String
}

enum HyperlitePresentation {
    static let supportedAgeWindows = [3, 5, 7, 10, 30]

    static func items(scan: HyperliteWorkScan, maxAgeDays: Int, now: Date = Date()) -> [HyperliteWorkItem] {
        let ageDays = min(30, max(3, maxAgeDays))
        let cutoff = now.addingTimeInterval(-Double(ageDays) * 86_400)
        return scan.items.filter { item in
            guard let updatedAt = item.updatedAt else { return false }
            return updatedAt >= cutoff && updatedAt <= now
        }.sorted {
            if $0.needsAttention != $1.needsAttention { return $0.needsAttention && !$1.needsAttention }
            return ($0.updatedAt ?? .distantPast) > ($1.updatedAt ?? .distantPast)
        }
    }

    static func ageLabel(for date: Date?, now: Date = Date()) -> String {
        guard let date else { return "age unknown" }
        let seconds = max(0, Int(now.timeIntervalSince(date)))
        if seconds < 60 { return "now" }
        if seconds < 3_600 { return "\(seconds / 60)m" }
        if seconds < 86_400 { return "\(seconds / 3_600)h" }
        return "\(seconds / 86_400)d"
    }
}
