import XCTest
@testable import Beacon

extension AppStateTests {
    func testStartLoadsCachedSnapshotWithoutRefreshing() async {
        let agent = ScriptedAgent(events: [TestSnapshots.snapshotEvent(TestSnapshots.empty)])
        let state = AppState(agent: agent, installer: nil)
        state.start()
        defer { state.stop() }
        for _ in 0..<20 where state.snapshot == nil {
            try? await Task.sleep(for: .milliseconds(10))
        }
        let refreshCalls = await agent.refreshCalls
        XCTAssertEqual(state.snapshot, TestSnapshots.empty)
        XCTAssertEqual(refreshCalls, 0)
        XCTAssertFalse(state.isScanning)
    }

    func testFastCompletedScanReconcilesSpinnerWithAgentStatus() async {
        let agent = ScriptedAgent(events: [TestSnapshots.snapshotEvent(TestSnapshots.empty)])
        let state = AppState(agent: agent, installer: nil)

        await state.scan()

        let refreshCalls = await agent.refreshCalls
        XCTAssertEqual(refreshCalls, 1)
        XCTAssertFalse(state.isScanning)
    }

    func testSignalNotesLoadAndSaveThroughSharedAgentAuthority() async {
        let agent = ScriptedAgent(
            events: [TestSnapshots.snapshotEvent(TestSnapshots.empty)],
            signalNotes: AgentNotes(
                content: "# Signal Log\n\nInitial spark",
                path: "/tmp/beacon/notes.md",
                updatedAt: "2026-07-13T13:00:00Z"
            )
        )
        let state = AppState(agent: agent, installer: nil)

        await state.loadNotes()
        XCTAssertEqual(state.notesContent, "# Signal Log\n\nInitial spark")
        XCTAssertEqual(state.notesPath, "/tmp/beacon/notes.md")

        await state.saveNotes("# Signal Log\n\nNew orbit")
        let setCalls = await agent.setNotesCalls
        XCTAssertEqual(setCalls, 1)
        XCTAssertEqual(state.notesContent, "# Signal Log\n\nNew orbit")
        XCTAssertEqual(state.notesUpdatedAt, "2026-07-13T14:00:00Z")
        XCTAssertNil(state.notesError)
        XCTAssertFalse(state.isSavingNotes)
    }

    func testSignalNotesAutosaveDefaultsExpandedAndDebouncesToLatestEdit() async {
        XCTAssertTrue(SignalNotesPresentation.expandedByDefault)
        XCTAssertEqual(SignalNotesPresentation.autosaveDelay, .seconds(3))
        let autosave = SignalNotesAutosave(delay: .milliseconds(20))
        var saved: [String] = []

        autosave.schedule(content: "first") { saved.append($0) }
        autosave.schedule(content: "second") { saved.append($0) }
        try? await Task.sleep(for: .milliseconds(60))

        XCTAssertEqual(saved, ["second"])
    }

    func testSignalNoteWorkspaceSharesDraftAndFlushesBeforeSwitching() async {
        let agent = NotesWorkspaceAgent()
        let state = AppState(agent: agent, installer: nil)

        await state.loadNotes()
        XCTAssertEqual(state.activeNoteID, "general")
        XCTAssertEqual(state.openNoteTabs.map(\.id), ["general", "detail-1", "detail-2"])
        XCTAssertEqual(state.noteHistory.map(\.id), ["detail-2", "detail-1"])

        state.updateNotesDraft("Renamed General\nexpanded")
        XCTAssertEqual(state.activeNoteTitle, "General")
        XCTAssertTrue(state.notesAreDirty)

        await state.activateNote("detail-1")

        let saved = await agent.savedNoteIDs
        XCTAssertEqual(saved, ["general"])
        XCTAssertEqual(state.activeNoteID, "detail-1")
        XCTAssertEqual(state.notesDraft, "First detail\nbody")
        XCTAssertFalse(state.notesAreDirty)
    }

    func testNotesPinningKeepsPinnedTabsLeftAndReordersThem() async {
        let agent = NotesWorkspaceAgent()
        let state = AppState(agent: agent, installer: nil)
        await state.loadNotes()

        await state.setNotePinned("detail-2", pinned: true)
        XCTAssertEqual(state.pinnedNoteIDs, ["general", "detail-2"])
        XCTAssertEqual(state.openNoteTabs.map(\.id), ["general", "detail-2", "detail-1"])

        await state.setNotePinned("detail-1", pinned: true)
        await state.movePinnedNote("detail-1", before: "detail-2")
        XCTAssertEqual(state.pinnedNoteIDs, ["general", "detail-1", "detail-2"])
        XCTAssertEqual(state.openNoteTabs.map(\.id), ["general", "detail-1", "detail-2"])

        await state.closeNote("detail-2")
        let closedNoteIDs = await agent.closedNoteIDs
        XCTAssertTrue(closedNoteIDs.isEmpty)
        await state.setNotePinned("detail-1", pinned: false)
        XCTAssertEqual(state.pinnedNoteIDs, ["general", "detail-2"])
        XCTAssertEqual(state.openNoteTabs.map(\.id), ["general", "detail-2", "detail-1"])
        let pinnedReorders = await agent.pinnedReorders
        XCTAssertEqual(pinnedReorders, [["detail-1", "detail-2"]])
    }

    func testSignalNoteLiveTitleDuplicateActivationAndCycling() async {
        let agent = NotesWorkspaceAgent()
        let state = AppState(agent: agent, installer: nil)
        await state.loadNotes()

        await state.activateNote("detail-1")
        state.updateNotesDraft("Live renamed title\nbody")
        XCTAssertEqual(state.activeNoteTitle, "Live renamed title")
        await state.activateNote("detail-1")
        XCTAssertEqual(state.openNoteTabs.map(\.id), ["general", "detail-1", "detail-2"])

        await state.cycleNotes(direction: 1)
        XCTAssertEqual(state.activeNoteID, "detail-2")
        await state.cycleNotes(direction: -1)
        XCTAssertEqual(state.activeNoteID, "detail-1")
    }

    func testSignalNoteCloseFlushesAndSelectsLeftNeighbor() async {
        let agent = NotesWorkspaceAgent(activeID: "detail-2")
        let state = AppState(agent: agent, installer: nil)
        await state.loadNotes()
        state.updateNotesDraft("Second detail revised\nbody")

        await state.closeNote("detail-2")

        let saved = await agent.savedNoteIDs
        XCTAssertEqual(saved, ["detail-2"])
        XCTAssertEqual(state.activeNoteID, "detail-1")
        XCTAssertEqual(state.openNoteTabs.map(\.id), ["general", "detail-1"])
        XCTAssertTrue(state.noteHistory.contains { $0.id == "detail-2" && !$0.isOpen })
    }

    func testSignalNoteDeleteDiscardsActiveDraftAndRemovesHistory() async {
        let agent = NotesWorkspaceAgent(activeID: "detail-2")
        let state = AppState(agent: agent, installer: nil)
        await state.loadNotes()
        state.updateNotesDraft("Unsaved content that will be deleted")

        await state.deleteNote("detail-2")

        let saved = await agent.savedNoteIDs
        let deleted = await agent.deletedNoteIDs
        XCTAssertTrue(saved.isEmpty)
        XCTAssertEqual(deleted, ["detail-2"])
        XCTAssertEqual(state.activeNoteID, "detail-1")
        XCTAssertEqual(state.openNoteTabs.map(\.id), ["general", "detail-1"])
        XCTAssertFalse(state.noteHistory.contains { $0.id == "detail-2" })
        XCTAssertNil(state.notesError)
    }

    func testSignalNoteSaveFailureKeepsCurrentTabOpen() async {
        let agent = NotesWorkspaceAgent(failSaves: true)
        let state = AppState(agent: agent, installer: nil)
        await state.loadNotes()
        state.updateNotesDraft("Unsaved edit")

        await state.activateNote("detail-1")

        XCTAssertEqual(state.activeNoteID, "general")
        XCTAssertEqual(state.notesDraft, "Unsaved edit")
        XCTAssertNotNil(state.notesError)
        let opened = await agent.openedNoteIDs
        XCTAssertTrue(opened.isEmpty)
    }

    func testSignalNoteCreationUsesRememberedGeneralCaretLine() async {
        let agent = NotesWorkspaceAgent()
        let state = AppState(agent: agent, installer: nil)
        await state.loadNotes()
        state.updateNotesCurrentLine("  [labcore] generate endpoints refactor  ")

        await state.showNewNotePicker()
        XCTAssertEqual(state.activeNoteID, "new")
        XCTAssertEqual(state.notesCurrentLine, "[labcore] generate endpoints refactor")
        await state.createNoteFromCurrentLine()

        XCTAssertEqual(state.activeNoteID, "detail-3")
        XCTAssertEqual(state.notesDraft, "[labcore] generate endpoints refactor\n\n")
        XCTAssertEqual(state.openNoteTabs.last?.id, "detail-3")
    }

    func testSignalNoteWorkspaceUsesBundledFallbackForOlderAgent() async {
        let expanded = "[labcore] generate endpoints refactor\nexpanded detail"
        let agent = LegacyNotesAgent()
        let fallback = ScriptedNotesFallbackClient(workspaces: [
            legacyFallbackWorkspace(activeID: "general", openIDs: ["general"]),
            legacyFallbackWorkspace(activeID: "detail-1", openIDs: ["general", "detail-1"]),
            legacyFallbackWorkspace(
                activeID: "detail-1", openIDs: ["general", "detail-1"], detailContent: expanded
            ),
            legacyFallbackWorkspace(
                activeID: "general", openIDs: ["general", "detail-1"], detailContent: expanded
            ),
            legacyFallbackWorkspace(
                activeID: "detail-1", openIDs: ["general", "detail-1"], detailContent: expanded
            ),
            legacyFallbackWorkspace(activeID: "general", openIDs: ["general"], detailContent: expanded),
            legacyFallbackWorkspace(activeID: "general", openIDs: ["general"], detailContent: expanded, includeDetail: false),
        ])
        let state = AppState(agent: agent, installer: nil, notesFallback: fallback)

        await state.loadNotes()
        state.updateNotesCurrentLine("[labcore] generate endpoints refactor")
        await state.createNoteFromCurrentLine()
        state.updateNotesDraft(expanded)
        await state.activateNote("general")
        await state.activateNote("detail-1")
        await state.closeNote("detail-1")
        await state.deleteNote("detail-1")

        let agentCalls = await agent.calls
        let fallbackCalls = await fallback.calls
        XCTAssertEqual(agentCalls, ["workspace"])
        XCTAssertEqual(fallbackCalls, [
            "workspace", "create", "set:detail-1", "open:general", "open:detail-1", "close:detail-1", "delete:detail-1",
        ])
        XCTAssertEqual(state.activeNoteID, "general")
        XCTAssertEqual(state.openNoteTabs.map(\.id), ["general"])
        XCTAssertFalse(state.noteHistory.contains { $0.id == "detail-1" })
        XCTAssertNil(state.notesError)
    }
}
