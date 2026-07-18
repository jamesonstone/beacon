import XCTest
@testable import Beacon

struct StubClient: CLIClientProtocol {
    let result: Result<BeaconSnapshot, Error>
    func scan() async throws -> BeaconSnapshot { try result.get() }
    func setProjectTracked(_ github: String, tracked: Bool) async throws {}
}

actor ExternalActivityClientStub: CLIClientProtocol {
    let initial: ExternalActivitySnapshot
    let pruned: ExternalActivitySnapshot
    let statuses: [String: IntegrationHealthStatus]
    private(set) var pruneCalls = 0

    init(
        initial: ExternalActivitySnapshot,
        pruned: ExternalActivitySnapshot = .empty,
        statuses: [String: IntegrationHealthStatus] = [:]
    ) {
        self.initial = initial
        self.pruned = pruned
        self.statuses = statuses
    }

    func scan() async throws -> BeaconSnapshot { TestSnapshots.empty }
    func setProjectTracked(_ github: String, tracked: Bool) async throws {}
    func externalActivity() async throws -> ExternalActivitySnapshot { initial }
    func pruneExternalActivity() async throws -> ExternalActivitySnapshot {
        pruneCalls += 1
        return pruned
    }
    func integrationStatus(_ provider: String) async throws -> IntegrationHealthStatus {
        if let status = statuses[provider] { return status }
        return IntegrationHealthStatus(
            provider: provider,
            state: "not_installed",
            settingsPath: "/tmp/settings.json",
            message: nil
        )
    }
}

actor SequenceClient: CLIClientProtocol {
    var results: [Result<BeaconSnapshot, Error>]

    init(results: [Result<BeaconSnapshot, Error>]) {
        self.results = results
    }

    func scan() async throws -> BeaconSnapshot {
        try results.removeFirst().get()
    }

    func setProjectTracked(_ github: String, tracked: Bool) async throws {}
}

struct TrackingCall: Equatable {
    let github: String
    let tracked: Bool
}

struct LaneAttentionCall: Equatable {
    let id: String
    let state: String
}

actor RecordingLaneAttentionAgent: AgentClientProtocol {
    let mutationEvent: AgentEvent
    private(set) var calls: [LaneAttentionCall] = []
    private(set) var laneOrderCalls: [[String]] = []

    init(mutationEvent: AgentEvent) {
        self.mutationEvent = mutationEvent
    }

    func snapshot() async throws -> AgentEvent { mutationEvent }
    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error> {
        AsyncThrowingStream { $0.finish() }
    }
    func refresh(project: String?) async throws -> String { "scan" }
    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent { mutationEvent }
    func status() async throws -> AgentStatusDetails {
        AgentStatusDetails(
            running: true, pid: 1, startedAt: nil, refreshing: false,
            scanID: nil, projectCount: 1, socket: "/socket"
        )
    }
    func setLaneAttention(_ id: String, state: String) async throws -> AgentEvent {
        calls.append(LaneAttentionCall(id: id, state: state))
        return mutationEvent
    }
    func reorderLanes(_ ids: [String]) async throws -> AgentEvent {
        laneOrderCalls.append(ids)
        return mutationEvent
    }
}

actor RecordingClient: CLIClientProtocol {
    var results: [Result<BeaconSnapshot, Error>]
    private(set) var trackingCalls: [TrackingCall] = []
    private(set) var maximumConcurrentTrackingCalls = 0
    private var activeTrackingCalls = 0
    private var trackingResults: [Result<Void, Error>]
    private let trackingDelay: Duration

    init(
        results: [Result<BeaconSnapshot, Error>],
        trackingResults: [Result<Void, Error>] = [],
        trackingDelay: Duration = .zero
    ) {
        self.results = results
        self.trackingResults = trackingResults
        self.trackingDelay = trackingDelay
    }

    func scan() async throws -> BeaconSnapshot {
        try results.removeFirst().get()
    }

    func setProjectTracked(_ github: String, tracked: Bool) async throws {
        trackingCalls.append(TrackingCall(github: github, tracked: tracked))
        activeTrackingCalls += 1
        maximumConcurrentTrackingCalls = max(maximumConcurrentTrackingCalls, activeTrackingCalls)
        defer { activeTrackingCalls -= 1 }
        if trackingDelay > .zero {
            try await Task.sleep(for: trackingDelay)
        }
        if !trackingResults.isEmpty {
            try trackingResults.removeFirst().get()
        }
    }
}

enum TestError: Error { case failed }
