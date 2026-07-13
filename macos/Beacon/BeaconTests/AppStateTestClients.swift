import XCTest
@testable import Beacon

struct StubClient: CLIClientProtocol {
    let result: Result<BeaconSnapshot, Error>
    func scan() async throws -> BeaconSnapshot { try result.get() }
    func setProjectTracked(_ github: String, tracked: Bool) async throws {}
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

actor ScriptedAgent: AgentClientProtocol {
    let events: [AgentEvent]
    let terminalError: Error?
    let statusDetails: AgentStatusDetails
    private(set) var refreshCalls = 0

    init(
        events: [AgentEvent],
        terminalError: Error? = nil,
        statusDetails: AgentStatusDetails = AgentStatusDetails(
            running: true, pid: 1, startedAt: nil, refreshing: false,
            scanID: nil, projectCount: 1, socket: "/socket"
        )
    ) {
        self.events = events
        self.terminalError = terminalError
        self.statusDetails = statusDetails
    }

    func snapshot() async throws -> AgentEvent {
        try XCTUnwrap(events.first)
    }

    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error> {
        let values = events
        let failure = terminalError
        return AsyncThrowingStream { continuation in
            for event in values { continuation.yield(event) }
            if let failure {
                continuation.finish(throwing: failure)
            } else {
                continuation.finish()
            }
        }
    }

    func refresh(project: String?) async throws -> String {
        refreshCalls += 1
        return "scan"
    }
    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent {
        try XCTUnwrap(events.first)
    }
    func status() async throws -> AgentStatusDetails {
        statusDetails
    }
}
