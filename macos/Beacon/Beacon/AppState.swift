import AppKit
import Foundation

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
            guard latest.schemaVersion == 1 else {
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

    func open(_ lane: WorkLane) {
        if let pullRequest = lane.pullRequest, let url = URL(string: pullRequest.url) {
            NSWorkspace.shared.open(url)
        } else if let worktree = lane.worktree {
            NSWorkspace.shared.open(URL(fileURLWithPath: worktree.path))
        }
    }

    func openTopItem() {
        guard let lane = snapshot?.lanes.first else { return }
        open(lane)
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
}
