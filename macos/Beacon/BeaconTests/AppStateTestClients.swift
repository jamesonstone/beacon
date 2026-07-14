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
    private(set) var setNotesCalls = 0
    private(set) var repositoryCheckRefreshes: [Bool] = []
    private(set) var syncedProjectIDs: [[String]] = []
    private var signalNotes: AgentNotes
    private var repositorySyncReports: [RepositorySyncReport]
    private let repositorySyncError: Error?

    init(
        events: [AgentEvent],
        terminalError: Error? = nil,
        statusDetails: AgentStatusDetails = AgentStatusDetails(
            running: true, pid: 1, startedAt: nil, refreshing: false,
            scanID: nil, projectCount: 1, socket: "/socket"
        ),
        signalNotes: AgentNotes = AgentNotes(content: "", path: "/tmp/beacon/notes.md", updatedAt: nil),
        repositorySyncReports: [RepositorySyncReport] = [
            RepositorySyncReport(checkedAt: "2026-07-14T12:00:00Z", fetchAttempted: false, repositories: [])
        ],
        repositorySyncError: Error? = nil
    ) {
        self.events = events
        self.terminalError = terminalError
        self.statusDetails = statusDetails
        self.signalNotes = signalNotes
        self.repositorySyncReports = repositorySyncReports
        self.repositorySyncError = repositorySyncError
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

    func notes() async throws -> AgentEvent {
        notesEvent(type: "notes")
    }

    func setNotes(_ content: String) async throws -> AgentEvent {
        setNotesCalls += 1
        signalNotes = AgentNotes(content: content, path: signalNotes.path, updatedAt: "2026-07-13T14:00:00Z")
        return notesEvent(type: "notes_updated")
    }

    func repositorySync(refresh: Bool) async throws -> AgentEvent {
        repositoryCheckRefreshes.append(refresh)
        if let repositorySyncError { throw repositorySyncError }
        return repositorySyncEvent(nextRepositorySyncReport())
    }

    func syncRepositories(_ projectIDs: [String]) async throws -> AgentEvent {
        syncedProjectIDs.append(projectIDs)
        if let repositorySyncError { throw repositorySyncError }
        return repositorySyncEvent(nextRepositorySyncReport())
    }

    private func notesEvent(type: String) -> AgentEvent {
        AgentEvent(
            protocolVersion: 1, requestID: nil, type: type, scanID: nil,
            projectID: nil, revision: nil, stage: "ready",
            generatedAt: "2026-07-13T14:00:00Z", message: nil,
            snapshot: nil, projects: nil, status: nil, notes: signalNotes
        )
    }

    private func nextRepositorySyncReport() -> RepositorySyncReport {
        if repositorySyncReports.count > 1 {
            return repositorySyncReports.removeFirst()
        }
        return repositorySyncReports[0]
    }

    private func repositorySyncEvent(_ report: RepositorySyncReport) -> AgentEvent {
        AgentEvent(
            protocolVersion: 1, requestID: nil, type: "repository_sync", scanID: nil,
            projectID: nil, revision: nil, stage: "ready",
            generatedAt: report.checkedAt, message: nil,
            snapshot: nil, projects: nil, status: nil, notes: nil,
            repositorySync: report
        )
    }
}

actor RepositorySyncFallbackClient: CLIClientProtocol {
    private var reports: [RepositorySyncReport]
    private(set) var checkRefreshes: [Bool] = []
    private(set) var syncedProjectIDs: [[String]] = []

    init(reports: [RepositorySyncReport]) {
        self.reports = reports
    }

    func scan() async throws -> BeaconSnapshot { throw TestError.failed }
    func setProjectTracked(_ github: String, tracked: Bool) async throws {}

    func repositorySync(refresh: Bool) async throws -> RepositorySyncReport {
        checkRefreshes.append(refresh)
        return nextReport()
    }

    func syncRepositories(_ projectIDs: [String]) async throws -> RepositorySyncReport {
        syncedProjectIDs.append(projectIDs)
        return nextReport()
    }

    private func nextReport() -> RepositorySyncReport {
        if reports.count > 1 { return reports.removeFirst() }
        return reports[0]
    }
}

actor DependencyLimitsClient: CLIClientProtocol {
    let report: DependencyLimitReport
    private(set) var calls = 0

    init(report: DependencyLimitReport) {
        self.report = report
    }

    func scan() async throws -> BeaconSnapshot { throw TestError.failed }
    func setProjectTracked(_ github: String, tracked: Bool) async throws {}

    func dependencyLimits() async throws -> DependencyLimitReport {
        calls += 1
        return report
    }
}
