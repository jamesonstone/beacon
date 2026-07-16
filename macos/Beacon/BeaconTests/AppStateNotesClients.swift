import Foundation
@testable import Beacon

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
