import Foundation

@MainActor
extension AppState {
    var pinnedNoteIDs: [String] {
        let tabs = notesWorkspace?.tabs ?? []
        let known = Set(tabs.map(\.id) + ["general"])
        let explicit = notesWorkspace?.pinnedIDs ?? tabs.filter { $0.pinned == true }.map(\.id)
        var ordered = ["general"]
        for id in explicit where id != "general" && id != "new" && known.contains(id) && !ordered.contains(id) {
            ordered.append(id)
        }
        return ordered
    }

    func setNotePinned(_ noteID: String, pinned: Bool) async {
        guard noteID != "general", noteID != "new" else { return }
        guard pinnedNoteIDs.contains(noteID) != pinned else { return }
        await applyNotesMutation(
            { try await agent.setNotePinned(noteID, pinned: pinned) },
            fallback: { try await $0.setNotePinned(noteID, pinned: pinned) }
        )
    }

    func movePinnedNote(_ noteID: String, before targetID: String) async {
        var details = pinnedNoteIDs.filter { $0 != "general" }
        guard noteID != targetID, let sourceIndex = details.firstIndex(of: noteID) else { return }
        details.remove(at: sourceIndex)
        let targetIndex: Int
        if targetID == "general" {
            targetIndex = 0
        } else if let index = details.firstIndex(of: targetID) {
            targetIndex = index
        } else {
            return
        }
        details.insert(noteID, at: targetIndex)
        guard details != pinnedNoteIDs.filter({ $0 != "general" }) else { return }
        await applyNotesMutation(
            { try await agent.reorderPinnedNotes(details) },
            fallback: { try await $0.reorderPinnedNotes(details) }
        )
    }
}
