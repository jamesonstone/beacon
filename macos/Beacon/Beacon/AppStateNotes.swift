import Foundation

@MainActor
extension AppState {
    func loadNotes() async {
        do {
            do {
                apply(try await agent.notesWorkspace())
            } catch {
                apply(try await agent.notes())
            }
            notesError = nil
        } catch {
            notesError = error.localizedDescription
        }
    }

    func saveNotes(_ content: String) async {
        _ = await saveNotes(content, noteID: activeNoteID)
    }

    var activeNoteID: String {
        notesWorkspace?.activeID ?? notesDraftID
    }

    var openNoteTabs: [AgentNoteTab] {
        let tabs = notesWorkspace?.tabs ?? []
        let byID = Dictionary(uniqueKeysWithValues: tabs.map { ($0.id, $0) })
        return (notesWorkspace?.openIDs ?? ["general"]).compactMap { id in
            byID[id] ?? (id == "general" ? AgentNoteTab(
                id: "general", title: "General", path: notesPath.isEmpty ? nil : notesPath,
                createdAt: nil, updatedAt: notesUpdatedAt, openedAt: nil, isOpen: true, pinned: true
            ) : nil)
        }
    }

    var noteHistory: [AgentNoteTab] {
        (notesWorkspace?.tabs ?? []).filter { $0.id != "general" && $0.id != "new" }
    }

    var activeNoteTitle: String {
        if activeNoteID == "new" { return "New Tab" }
        if activeNoteID == "general" { return "General" }
        let firstLine = notesDraft.components(separatedBy: .newlines).first?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return firstLine.isEmpty ? "Untitled" : firstLine
    }

    var notesAreDirty: Bool {
        activeNoteID != "new" && notesDraft != notesContent
    }

    func updateNotesDraft(_ content: String) {
        notesDraft = content
        scheduleNotesAutosave(content)
    }

    func updateNotesCurrentLine(_ line: String) {
        guard activeNoteID == "general" else { return }
        notesCurrentLine = line.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    func revertNotes() {
        notesAutosave.cancel()
        notesDraft = notesContent
    }

    func showNewNotePicker() async {
        guard await flushNotes() else { return }
        await applyNotesMutation { try await agent.openNote("new") }
    }

    func createNote(title: String) async {
        let normalized = title.trimmingCharacters(in: .whitespacesAndNewlines)
        let content = normalized.isEmpty ? "\n" : normalized + "\n\n"
        await createNote(content: content)
    }

    func createNoteFromCurrentLine() async {
        guard !notesCurrentLine.isEmpty else { return }
        await createNote(content: notesCurrentLine + "\n\n")
    }

    func createNote(content: String) async {
        guard await flushNotes() else { return }
        await applyNotesMutation { try await agent.createNote(content) }
    }

    func activateNote(_ noteID: String) async {
        guard noteID != activeNoteID else { return }
        guard await flushNotes() else { return }
        await applyNotesMutation { try await agent.openNote(noteID) }
    }

    func closeNote(_ noteID: String) async {
        guard noteID != "general" else { return }
        guard await flushNotes() else { return }
        await applyNotesMutation { try await agent.closeNote(noteID) }
    }

    func cycleNotes(direction: Int) async {
        let identifiers = openNoteTabs.map(\.id)
        guard identifiers.count > 1, let index = identifiers.firstIndex(of: activeNoteID) else { return }
        let target = (index + direction + identifiers.count) % identifiers.count
        await activateNote(identifiers[target])
    }

    func activateNote(at index: Int) async {
        let tabs = openNoteTabs
        guard tabs.indices.contains(index) else { return }
        await activateNote(tabs[index].id)
    }

    func applyNotesWorkspace(_ workspace: AgentNotesWorkspace) {
        let keepDraft = notesDraftID == workspace.activeID && notesDraft != notesContent
        notesWorkspace = workspace
        guard let active = workspace.active else {
            notesDraftID = workspace.activeID
            notesContent = ""
            notesPath = ""
            notesUpdatedAt = nil
            notesDraft = ""
            notesError = nil
            return
        }
        applyNotesDocument(active, noteID: workspace.activeID, preserveDirtyDraft: keepDraft)
    }

    func applyNotesDocument(
        _ notes: AgentNotes,
        noteID: String,
        preserveDirtyDraft: Bool? = nil
    ) {
        let keepDraft = preserveDirtyDraft ?? (notesDraftID == noteID && notesDraft != notesContent)
        notesDraftID = noteID
        notesContent = notes.content
        notesPath = notes.path
        notesUpdatedAt = notes.updatedAt
        if !keepDraft {
            notesDraft = notes.content
        }
        notesError = nil
    }

    private func scheduleNotesAutosave(_ content: String) {
        guard activeNoteID != "new", content != notesContent else {
            notesAutosave.cancel()
            return
        }
        let noteID = activeNoteID
        notesAutosave.schedule(content: content) { [weak self] candidate in
            guard let self, self.activeNoteID == noteID, candidate != self.notesContent else { return }
            _ = await self.saveNotes(candidate, noteID: noteID)
        }
    }

    private func flushNotes() async -> Bool {
        notesAutosave.cancel()
        while isSavingNotes {
            try? await Task.sleep(for: .milliseconds(50))
        }
        guard notesAreDirty else { return true }
        return await saveNotes(notesDraft, noteID: activeNoteID)
    }

    @discardableResult
    private func saveNotes(_ content: String, noteID: String) async -> Bool {
        guard !isSavingNotes else { return false }
        isSavingNotes = true
        defer { isSavingNotes = false }
        do {
            apply(try await agent.setNotes(content, noteID: noteID))
            notesError = nil
            return true
        } catch {
            notesError = error.localizedDescription
            return false
        }
    }

    private func applyNotesMutation(_ operation: () async throws -> AgentEvent) async {
        do {
            apply(try await operation())
            notesError = nil
        } catch {
            notesError = error.localizedDescription
        }
    }
}
