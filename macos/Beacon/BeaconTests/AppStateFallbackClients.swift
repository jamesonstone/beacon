import Foundation
@testable import Beacon

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

    func setNotePinned(_ noteID: String, pinned: Bool) async throws -> AgentEvent {
        calls.append("pin:\(noteID):\(pinned)")
        throw AgentClientError.command("unknown agent request: set_note_pinned")
    }

    func reorderPinnedNotes(_ noteIDs: [String]) async throws -> AgentEvent {
        calls.append("reorder:\(noteIDs.joined(separator: ","))")
        throw AgentClientError.command("unknown agent request: reorder_pinned_notes")
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

    func setNotePinned(_ noteID: String, pinned: Bool) async throws -> AgentNotesWorkspace {
        try next("pin:\(noteID):\(pinned)")
    }

    func reorderPinnedNotes(_ noteIDs: [String]) async throws -> AgentNotesWorkspace {
        try next("reorder:\(noteIDs.joined(separator: ","))")
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
        version: 1, activeID: activeID, openIDs: openIDs, pinnedIDs: ["general"],
        tabs: tabs,
        active: activeID == "general" ? general : (includeDetail ? detail : nil)
    )
}
