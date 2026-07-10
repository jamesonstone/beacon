import AppKit
import Foundation

struct ProjectLaneGroup: Identifiable, Equatable {
    let id: String
    let name: String
    let progress: ProgressDetails?
    let lanes: [WorkLane]
}

@MainActor
final class AppState: ObservableObject {
    @Published private(set) var snapshot: BeaconSnapshot?
    @Published private(set) var isScanning = false
    @Published private(set) var lastError: String?

    private let client: CLIClientProtocol
    private var pollingTask: Task<Void, Never>?

    init(client: CLIClientProtocol = CLIClient()) {
        self.client = client
    }

    var readyCount: Int { snapshot?.summary.reviewReady ?? 0 }

    var inProgressCount: Int {
        guard let groups = snapshot?.groups else { return 0 }
        return groups.ready.count + groups.action.count + groups.waiting.count
    }

    func start() {
        guard pollingTask == nil else { return }
        pollingTask = Task { [weak self] in
            await self?.scan()
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(60))
                guard !Task.isCancelled else { return }
                await self?.scan()
            }
        }
    }

    func stop() {
        pollingTask?.cancel()
        pollingTask = nil
    }

    func scan() async {
        guard !isScanning else { return }
        isScanning = true
        defer { isScanning = false }
        do {
            let latest = try await client.scan()
            guard latest.schemaVersion == 2 else {
                throw CLIClientError.invalidOutput("unsupported schema version \(latest.schemaVersion)")
            }
            snapshot = latest
            lastError = nil
        } catch {
            lastError = error.localizedDescription
        }
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

    func open(_ lane: WorkLane) {
        if let target = Self.openTarget(for: lane) {
            NSWorkspace.shared.open(target)
        }
    }

    func openTopItem() {
        guard let snapshot else { return }
        let identifiers = snapshot.groups.ready
            + snapshot.groups.action
            + snapshot.groups.waiting
            + snapshot.groups.idle
        guard let lane = lanes(for: identifiers).first else { return }
        open(lane)
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
