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

actor LegacyNotesAgent: AgentClientProtocol {
    private let general = AgentNotes(
        content: "General spark\nsecond line", path: "/tmp/beacon/notes.md",
        updatedAt: "2026-07-14T14:00:00Z", id: "general", title: "General"
    )
    private(set) var calls: [String] = []

    func snapshot() async throws -> AgentEvent { throw TestError.failed }
    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error> {
        AsyncThrowingStream { $0.finish() }
    }
    func refresh(project: String?) async throws -> String { throw TestError.failed }
    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent { throw TestError.failed }
    func status() async throws -> AgentStatusDetails { throw TestError.failed }

    func notes() async throws -> AgentEvent {
        calls.append("notes")
        return event(type: "notes", notes: general)
    }

    func notesWorkspace() async throws -> AgentEvent {
        calls.append("workspace")
        throw AgentClientError.command("unknown agent request: get_notes_workspace")
    }

    func setNotes(_ content: String, noteID: String) async throws -> AgentEvent {
        calls.append("set:\(noteID)")
        throw AgentClientError.command("unknown field \"note_id\"")
    }

    func createNote(_ content: String) async throws -> AgentEvent {
        calls.append("create")
        throw AgentClientError.command("unknown agent request: create_note")
    }

    func openNote(_ noteID: String) async throws -> AgentEvent {
        calls.append("open:\(noteID)")
        throw AgentClientError.command("unknown agent request: open_note")
    }

    func closeNote(_ noteID: String) async throws -> AgentEvent {
        calls.append("close:\(noteID)")
        throw AgentClientError.command("unknown agent request: close_note")
    }

    private func event(type: String, notes: AgentNotes?) -> AgentEvent {
        AgentEvent(
            protocolVersion: 1, requestID: nil, type: type, scanID: nil,
            projectID: nil, revision: nil, stage: "ready",
            generatedAt: "2026-07-14T15:00:00Z", message: nil,
            snapshot: nil, projects: nil, status: nil, notes: notes
        )
    }
}

actor ScriptedNotesFallbackClient: CLIClientProtocol {
    private var workspaces: [AgentNotesWorkspace]
    private(set) var calls: [String] = []

    init(workspaces: [AgentNotesWorkspace]) {
        self.workspaces = workspaces
    }

    func scan() async throws -> BeaconSnapshot { throw TestError.failed }
    func setProjectTracked(_ github: String, tracked: Bool) async throws {}

    func notesWorkspace() async throws -> AgentNotesWorkspace {
        try next("workspace")
    }

    func setNotes(_ content: String, noteID: String) async throws -> AgentNotesWorkspace {
        try next("set:\(noteID)")
    }

    func createNote(_ content: String) async throws -> AgentNotesWorkspace {
        try next("create")
    }

    func openNote(_ noteID: String) async throws -> AgentNotesWorkspace {
        try next("open:\(noteID)")
    }

    func closeNote(_ noteID: String) async throws -> AgentNotesWorkspace {
        try next("close:\(noteID)")
    }

    func deleteNote(_ noteID: String) async throws -> AgentNotesWorkspace {
        try next("delete:\(noteID)")
    }

    private func next(_ call: String) throws -> AgentNotesWorkspace {
        calls.append(call)
        guard !workspaces.isEmpty else { throw TestError.failed }
        return workspaces.removeFirst()
    }
}

func legacyFallbackWorkspace(
    activeID: String,
    openIDs: [String],
    detailContent: String = "[labcore] generate endpoints refactor\n\n",
    includeDetail: Bool = true
) -> AgentNotesWorkspace {
    let general = AgentNotes(
        content: "General spark\nsecond line", path: "/tmp/beacon/notes.md",
        updatedAt: "2026-07-14T14:00:00Z", id: "general", title: "General"
    )
    let detail = AgentNotes(
        content: detailContent, path: "/tmp/beacon/notes/detail-1.md",
        updatedAt: "2026-07-14T15:00:00Z", id: "detail-1",
        title: "[labcore] generate endpoints refactor"
    )
    let openSet = Set(openIDs)
    var tabs = [
        AgentNoteTab(
            id: "general", title: "General", path: general.path,
            createdAt: nil, updatedAt: general.updatedAt, openedAt: nil,
            isOpen: true, pinned: true
        ),
    ]
    if includeDetail {
        tabs.append(AgentNoteTab(
            id: "detail-1", title: detail.title ?? "Untitled", path: detail.path,
            createdAt: nil, updatedAt: detail.updatedAt, openedAt: nil,
            isOpen: openSet.contains("detail-1"), pinned: nil
        ))
    }
    return AgentNotesWorkspace(
        version: 1, activeID: activeID, openIDs: openIDs,
        tabs: tabs,
        active: activeID == "general" ? general : (includeDetail ? detail : nil)
    )
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

actor NotesWorkspaceAgent: AgentClientProtocol {
    private var documents: [String: AgentNotes]
    private var openIDs: [String]
    private var activeID: String
    private let detailOrder: [String]
    private let failSaves: Bool
    private(set) var savedNoteIDs: [String] = []
    private(set) var openedNoteIDs: [String] = []
    private(set) var closedNoteIDs: [String] = []
    private(set) var deletedNoteIDs: [String] = []

    init(
        openIDs: [String] = ["general", "detail-1", "detail-2"],
        activeID: String = "general",
        failSaves: Bool = false
    ) {
        let general = AgentNotes(
            content: "General spark\nsecond line", path: "/tmp/beacon/notes.md",
            updatedAt: "2026-07-14T14:00:00Z", id: "general", title: "General"
        )
        let first = AgentNotes(
            content: "First detail\nbody", path: "/tmp/beacon/notes/detail-1.md",
            updatedAt: "2026-07-14T14:01:00Z", id: "detail-1", title: "First detail"
        )
        let second = AgentNotes(
            content: "Second detail\nbody", path: "/tmp/beacon/notes/detail-2.md",
            updatedAt: "2026-07-14T14:02:00Z", id: "detail-2", title: "Second detail"
        )
        documents = ["general": general, "detail-1": first, "detail-2": second]
        self.openIDs = openIDs
        self.activeID = activeID
        detailOrder = ["detail-2", "detail-1"]
        self.failSaves = failSaves
    }

    func snapshot() async throws -> AgentEvent { throw TestError.failed }
    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error> {
        AsyncThrowingStream { $0.finish() }
    }
    func refresh(project: String?) async throws -> String { throw TestError.failed }
    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent { throw TestError.failed }
    func status() async throws -> AgentStatusDetails { throw TestError.failed }

    func notes() async throws -> AgentEvent {
        event(type: "notes", notes: documents["general"])
    }

    func notesWorkspace() async throws -> AgentEvent {
        workspaceEvent(type: "notes_workspace")
    }

    func notes(noteID: String) async throws -> AgentEvent {
        event(type: "notes", notes: documents[noteID])
    }

    func setNotes(_ content: String) async throws -> AgentEvent {
        try await setNotes(content, noteID: "general")
    }

    func setNotes(_ content: String, noteID: String) async throws -> AgentEvent {
        if failSaves { throw AgentClientError.command("simulated save failure") }
        guard let current = documents[noteID] else { throw TestError.failed }
        let title = content.components(separatedBy: .newlines).first?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        documents[noteID] = AgentNotes(
            content: content, path: current.path, updatedAt: "2026-07-14T15:00:00Z",
            id: noteID, title: title?.isEmpty == false ? title : "Untitled",
            createdAt: current.createdAt, openedAt: current.openedAt
        )
        savedNoteIDs.append(noteID)
        return workspaceEvent(type: "notes_updated")
    }

    func createNote(_ content: String) async throws -> AgentEvent {
        let id = "detail-3"
        let title = content.components(separatedBy: .newlines).first?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        documents[id] = AgentNotes(
            content: content, path: "/tmp/beacon/notes/\(id).md", updatedAt: "2026-07-14T15:00:00Z",
            id: id, title: title?.isEmpty == false ? title : "Untitled"
        )
        openIDs.removeAll { $0 == "new" }
        if !openIDs.contains(id) { openIDs.append(id) }
        activeID = id
        return workspaceEvent(type: "notes_workspace_updated")
    }

    func openNote(_ noteID: String) async throws -> AgentEvent {
        openedNoteIDs.append(noteID)
        if !openIDs.contains(noteID) { openIDs.append(noteID) }
        activeID = noteID
        return workspaceEvent(type: "notes_workspace_updated")
    }

    func closeNote(_ noteID: String) async throws -> AgentEvent {
        closedNoteIDs.append(noteID)
        guard let index = openIDs.firstIndex(of: noteID), noteID != "general" else { throw TestError.failed }
        openIDs.remove(at: index)
        if activeID == noteID {
            activeID = index > 0 ? openIDs[index - 1] : "general"
        }
        return workspaceEvent(type: "notes_workspace_updated")
    }

    func deleteNote(_ noteID: String) async throws -> AgentEvent {
        deletedNoteIDs.append(noteID)
        guard noteID != "general", noteID != "new", documents[noteID] != nil else { throw TestError.failed }
        let index = openIDs.firstIndex(of: noteID)
        openIDs.removeAll { $0 == noteID }
        documents[noteID] = nil
        if activeID == noteID {
            activeID = index.flatMap { $0 > 0 && $0 - 1 < openIDs.count ? openIDs[$0 - 1] : nil } ?? "general"
        }
        return workspaceEvent(type: "notes_workspace_updated")
    }

    private func workspaceEvent(type: String) -> AgentEvent {
        let workspace = workspace()
        return event(type: type, notes: workspace.active, notesWorkspace: workspace)
    }

    private func workspace() -> AgentNotesWorkspace {
        let openSet = Set(openIDs)
        var tabs = [AgentNoteTab(
            id: "general", title: "General", path: documents["general"]?.path,
            createdAt: nil, updatedAt: documents["general"]?.updatedAt, openedAt: nil,
            isOpen: true, pinned: true
        )]
        if openSet.contains("new") {
            tabs.append(AgentNoteTab(
                id: "new", title: "New Tab", path: nil, createdAt: nil,
                updatedAt: nil, openedAt: nil, isOpen: true, pinned: nil
            ))
        }
        let knownOrder = detailOrder + (documents.keys.contains("detail-3") ? ["detail-3"] : [])
        tabs += knownOrder.compactMap { id in
            guard let note = documents[id] else { return nil }
            return AgentNoteTab(
                id: id, title: note.title ?? "Untitled", path: note.path,
                createdAt: note.createdAt, updatedAt: note.updatedAt, openedAt: note.openedAt,
                isOpen: openSet.contains(id), pinned: nil
            )
        }
        return AgentNotesWorkspace(
            version: 1, activeID: activeID, openIDs: openIDs, tabs: tabs,
            active: activeID == "new" ? nil : documents[activeID]
        )
    }

    private func event(
        type: String,
        notes: AgentNotes?,
        notesWorkspace: AgentNotesWorkspace? = nil
    ) -> AgentEvent {
        AgentEvent(
            protocolVersion: 1, requestID: nil, type: type, scanID: nil,
            projectID: nil, revision: nil, stage: "ready",
            generatedAt: "2026-07-14T15:00:00Z", message: nil,
            snapshot: nil, projects: nil, status: nil, notes: notes,
            notesWorkspace: notesWorkspace
        )
    }
}
