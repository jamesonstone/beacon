import Foundation

struct HyperliteItem: Equatable, Identifiable {
    let lane: WorkLane
    let projectName: String
    let group: HyperliteGroup
    let attention: Bool
    let ageDate: Date?
    let workingDate: Date?

    var id: String { lane.id }
}

enum HyperliteGroup: String, Equatable {
    case active
    case waiting
    case recent
}

enum HyperlitePresentation {
    static func items(snapshot: BeaconSnapshot, activity: ExternalActivitySnapshot) -> [HyperliteItem] {
        let projects = Dictionary(uniqueKeysWithValues: snapshot.projects.map { ($0.github, $0.name) })
        let lanes = Dictionary(uniqueKeysWithValues: snapshot.lanes.map { ($0.id, $0) })
        let groups: [(HyperliteGroup, [String])] = [
            (.active, snapshot.workingSet?.active ?? snapshot.groups.ready),
            (.waiting, snapshot.workingSet?.waiting ?? snapshot.groups.waiting),
            (.recent, snapshot.workingSet?.recent ?? snapshot.groups.action),
        ]
        var seen = Set<String>()
        var result: [HyperliteItem] = []
        for (group, ids) in groups {
            for id in ids {
                guard seen.insert(id).inserted, let lane = lanes[id] else { continue }
                let workingDate = activity.records
                    .filter { $0.state == "working" && ($0.laneID == lane.id || ($0.laneID == nil && $0.projectID == lane.github)) }
                    .compactMap { parseDate($0.observedAt) }
                    .min()
                result.append(HyperliteItem(lane: lane, projectName: projects[lane.github] ?? lane.repository, group: group, attention: needsAttention(lane), ageDate: parseDate(lane.updatedAt), workingDate: workingDate))
            }
        }
        return result.sorted {
            if $0.attention != $1.attention { return $0.attention && !$1.attention }
            if groupRank($0.group) != groupRank($1.group) { return groupRank($0.group) < groupRank($1.group) }
            return ($0.ageDate ?? .distantPast) > ($1.ageDate ?? .distantPast)
        }
    }

    static func needsAttention(_ lane: WorkLane) -> Bool {
        if lane.reviewReady || !lane.blockers.isEmpty || !lane.warnings.isEmpty { return true }
        let action = lane.nextAction.lowercased()
        return !["wait", "waiting", "monitor", "none", "idle", "unknown"].contains { action.contains($0) }
    }

    static func ageLabel(for date: Date?, now: Date = Date()) -> String {
        guard let date else { return "age unknown" }
        let seconds = max(0, Int(now.timeIntervalSince(date)))
        if seconds < 60 { return "now" }
        if seconds < 3_600 { return "\(seconds / 60)m" }
        if seconds < 86_400 { return "\(seconds / 3_600)h" }
        return "\(seconds / 86_400)d"
    }

    private static func groupRank(_ group: HyperliteGroup) -> Int {
        switch group { case .active: 0; case .waiting: 1; case .recent: 2 }
    }

    private static func parseDate(_ value: String) -> Date? {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.date(from: value) ?? ISO8601DateFormatter().date(from: value)
    }
}

enum HyperlitePresentationDate {
    static func parse(_ value: String) -> Date? {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.date(from: value) ?? ISO8601DateFormatter().date(from: value)
    }
}
