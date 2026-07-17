import Foundation

extension AgentClient {
    func notes() async throws -> AgentEvent {
        try await notes(noteID: "general")
    }

    func notesWorkspace() async throws -> AgentEvent {
        let event = try await request(type: "get_notes_workspace")
        guard event.type != "project_failed", event.notesWorkspace != nil else {
            throw AgentClientError.command(event.message ?? "load Notes tabs failed")
        }
        return event
    }

    func notes(noteID: String) async throws -> AgentEvent {
        let event = try await request(payload: Self.noteRequestData(type: "get_notes", noteID: noteID))
        guard event.type != "project_failed" else {
            throw AgentClientError.command(event.message ?? "load Notes failed")
        }
        return event
    }

    func setNotes(_ content: String) async throws -> AgentEvent {
        try await setNotes(content, noteID: "general")
    }

    func setNotes(_ content: String, noteID: String) async throws -> AgentEvent {
        let event = try await request(
            payload: Self.noteRequestData(type: "set_notes", content: content, noteID: noteID)
        )
        guard event.type != "project_failed" else {
            throw AgentClientError.command(event.message ?? "save Notes failed")
        }
        return event
    }

    func createNote(_ content: String) async throws -> AgentEvent {
        try await noteWorkspaceMutation(type: "create_note", content: content)
    }

    func openNote(_ noteID: String) async throws -> AgentEvent {
        try await noteWorkspaceMutation(type: "open_note", noteID: noteID)
    }

    func closeNote(_ noteID: String) async throws -> AgentEvent {
        try await noteWorkspaceMutation(type: "close_note", noteID: noteID)
    }

    func deleteNote(_ noteID: String) async throws -> AgentEvent {
        try await noteWorkspaceMutation(type: "delete_note", noteID: noteID)
    }

    func setNotePinned(_ noteID: String, pinned: Bool) async throws -> AgentEvent {
        try await noteWorkspaceMutation(type: "set_note_pinned", noteID: noteID, pinned: pinned)
    }

    func reorderPinnedNotes(_ noteIDs: [String]) async throws -> AgentEvent {
        try await noteWorkspaceMutation(type: "reorder_pinned_notes", noteIDs: noteIDs)
    }

    private func noteWorkspaceMutation(
        type: String,
        noteID: String? = nil,
        noteIDs: [String]? = nil,
        pinned: Bool? = nil,
        content: String? = nil
    ) async throws -> AgentEvent {
        let event = try await request(
            type: type, pinned: pinned, content: content, noteID: noteID, noteIDs: noteIDs
        )
        guard event.type != "project_failed", event.notesWorkspace != nil else {
            throw AgentClientError.command(event.message ?? "update Notes tabs failed")
        }
        return event
    }
}
