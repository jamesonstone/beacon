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

    var followedProjects: [BeaconProject] {
        (snapshot?.projects ?? []).filter { presentedFollowState(for: $0) == "following" }
    }

    var recentProjects: [BeaconProject] {
        (snapshot?.projects ?? []).filter { presentedFollowState(for: $0) == "recent" }
    }

    var quietProjects: [BeaconProject] {
        (snapshot?.projects ?? []).filter { presentedFollowState(for: $0) == "quiet" }
    }

    var trackedProjects: [BeaconProject] { followedProjects }
    var untrackedProjects: [BeaconProject] { recentProjects + quietProjects }

    var inProgressCount: Int {
        if let working = snapshot?.workingSet {
            return working.active.count + working.waiting.count + working.recent.count
        }
        guard let groups = snapshot?.groups else { return 0 }
        return groups.ready.count + groups.action.count + groups.waiting.count
    }

    var followedProjectCount: Int { followedProjects.count }

    var recentProjectCount: Int { recentProjects.count }

    var quietProjectCount: Int { quietProjects.count }

    var queuedTrackingCount: Int { mutatingProjects.count }

    var untrackedProjectCount: Int { untrackedProjects.count }

    func projects(in followState: String, matching query: String = "") -> [BeaconProject] {
        let projects: [BeaconProject]
        switch followState {
        case "following": projects = followedProjects
        case "recent": projects = recentProjects
        default: projects = quietProjects
        }
        let normalizedQuery = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalizedQuery.isEmpty else { return projects }
        return projects.filter {
            $0.name.lowercased().contains(normalizedQuery)
                || $0.github.lowercased().contains(normalizedQuery)
                || $0.path.lowercased().contains(normalizedQuery)
        }
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

    func projectGroup(for lane: WorkLane) -> ProjectLaneGroup {
        if let project = snapshot?.projects.first(where: {
            $0.github == lane.github || $0.laneIDs.contains(lane.id)
        }) {
            return ProjectLaneGroup(
                id: project.github,
                name: project.name,
                progress: project.progress,
                lanes: [lane]
            )
        }
        return ProjectLaneGroup(id: lane.github, name: lane.repository, progress: nil, lanes: [lane])
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
