import AppKit
import Foundation

struct ProjectLaneGroup: Identifiable, Equatable {
    let id: String
    let name: String
    let progress: ProgressDetails?
    let lanes: [WorkLane]
}

extension AppState {
    var readyCount: Int { snapshot?.summary.reviewReady ?? 0 }

    var inProgressCount: Int {
        if let working = snapshot?.workingSet {
            return working.active.count + working.waiting.count + working.recent.count
        }
        guard let groups = snapshot?.groups else { return 0 }
        return groups.ready.count + groups.action.count + groups.waiting.count
    }

    var quietProjectCount: Int { quietProjectGroups().count }

    var queuedTrackingCount: Int { mutatingProjects.count }

    var untrackedProjectCount: Int {
        snapshot?.summary.untrackedProjects ?? untrackedProjects.count
    }

    func stage(for projectID: String) -> String {
        projectStatuses[projectID]?.stage ?? "cached"
    }

    func lanes(for identifiers: [String]) -> [WorkLane] {
        let lanesByID = Dictionary(uniqueKeysWithValues: (snapshot?.lanes ?? []).map { ($0.id, $0) })
        return identifiers.compactMap { lanesByID[$0] }
    }

    func projectGroups(for lanes: [WorkLane]) -> [ProjectLaneGroup] {
        var remaining = lanes
        var groups: [ProjectLaneGroup] = []

        for project in snapshot?.projects ?? [] {
            let matching = remaining.filter {
                $0.github == project.github || project.laneIDs.contains($0.id)
            }
            guard !matching.isEmpty else { continue }
            let matchingIDs = Set(matching.map(\.id))
            remaining.removeAll { matchingIDs.contains($0.id) }
            groups.append(ProjectLaneGroup(
                id: project.github,
                name: project.name,
                progress: project.progress,
                lanes: matching
            ))
        }

        while let first = remaining.first {
            let matching = remaining.filter { $0.github == first.github }
            let matchingIDs = Set(matching.map(\.id))
            remaining.removeAll { matchingIDs.contains($0.id) }
            groups.append(ProjectLaneGroup(
                id: first.github,
                name: first.repository,
                progress: nil,
                lanes: matching
            ))
        }
        return groups
    }

    func quietProjectGroups(matching query: String = "") -> [ProjectLaneGroup] {
        guard let snapshot else { return [] }
        let activeProjects = Set(lanes(for: snapshot.groups.ready + snapshot.groups.action + snapshot.groups.waiting).map(\.github))
        let quietLanes = lanes(for: snapshot.groups.idle).filter { !activeProjects.contains($0.github) }
        let groups = projectGroups(for: quietLanes)
        let normalizedQuery = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalizedQuery.isEmpty else { return groups }
        return groups.filter { group in
            group.name.lowercased().contains(normalizedQuery)
                || group.id.lowercased().contains(normalizedQuery)
                || group.lanes.contains { lane in
                    lane.repository.lowercased().contains(normalizedQuery)
                        || lane.branch.lowercased().contains(normalizedQuery)
                        || lane.pullRequest?.title.lowercased().contains(normalizedQuery) == true
                        || lane.issue?.title.lowercased().contains(normalizedQuery) == true
                }
        }
    }

    func open(_ lane: WorkLane) {
        if let target = Self.openTarget(for: lane) {
            NSWorkspace.shared.open(target)
        }
    }

    func openTopItem() {
        guard let lane = topLane() else { return }
        open(lane)
    }

    func topLane() -> WorkLane? {
        guard let snapshot else { return nil }
        let identifiers: [String]
        if let working = snapshot.workingSet {
            identifiers = working.active + working.waiting + working.recent
        } else {
            identifiers = snapshot.groups.ready + snapshot.groups.action + snapshot.groups.waiting
        }
        return lanes(for: identifiers).first { Self.openTarget(for: $0) != nil }
    }

    static func openTarget(for lane: WorkLane) -> URL? {
        if let pullRequest = lane.pullRequest, let url = webURL(pullRequest.url) {
            return url
        }
        if let issue = lane.issue, let url = webURL(issue.url) {
            return url
        }
        if let worktree = lane.worktree {
            return URL(fileURLWithPath: worktree.path)
        }
        return nil
    }

    func openConfig() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let config = home.appendingPathComponent(".config/beacon/config.yaml")
        if FileManager.default.fileExists(atPath: config.path) {
            NSWorkspace.shared.open(config)
        } else {
            NSWorkspace.shared.open(config.deletingLastPathComponent())
        }
    }

    private static func webURL(_ value: String) -> URL? {
        guard let url = URL(string: value), ["http", "https"].contains(url.scheme?.lowercased()) else {
            return nil
        }
        return url
    }
}
