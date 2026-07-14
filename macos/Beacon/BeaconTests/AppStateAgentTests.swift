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

    func testRepositorySyncCheckAndApplyShareAgentAuthority() async {
        let behind = RepositorySyncItem(
            projectID: "owner/repo", name: "repo", path: "/repo", base: "main", remote: "origin",
            currentBranch: "GH-12", baseWorktree: nil, currentAhead: 0, currentBehind: 2,
            defaultAhead: 0, defaultBehind: 2, dirty: false, detached: false,
            needsUpdate: true, canUpdate: true, fetched: false, updated: false,
            state: "behind", action: "switch_and_fast_forward", reason: "safe", error: nil
        )
        let current = RepositorySyncItem(
            projectID: "owner/repo", name: "repo", path: "/repo", base: "main", remote: "origin",
            currentBranch: "main", baseWorktree: "/repo", currentAhead: 0, currentBehind: 0,
            defaultAhead: 0, defaultBehind: 0, dirty: false, detached: false,
            needsUpdate: false, canUpdate: false, fetched: false, updated: true,
            state: "current", action: "none", reason: "updated", error: nil
        )
        let agent = ScriptedAgent(
            events: [TestSnapshots.snapshotEvent(TestSnapshots.empty)],
            repositorySyncReports: [
                RepositorySyncReport(checkedAt: "2026-07-14T12:00:00Z", fetchAttempted: false, repositories: [behind]),
                RepositorySyncReport(checkedAt: "2026-07-14T12:01:00Z", fetchAttempted: true, repositories: [current]),
            ]
        )
        let state = AppState(agent: agent, installer: nil)

        await state.checkRepositorySync(refresh: false)
        XCTAssertEqual(state.repositoriesNeedingSync.map(\.projectID), ["owner/repo"])
        XCTAssertEqual(state.safeRepositoryUpdates.map(\.projectID), ["owner/repo"])

        await state.syncRepositories(["owner/repo"])
        let checkRefreshes = await agent.repositoryCheckRefreshes
        let syncedProjectIDs = await agent.syncedProjectIDs
        XCTAssertTrue(state.repositoriesNeedingSync.isEmpty)
        XCTAssertEqual(state.repositorySyncReport?.repositories.first?.currentBranch, "main")
        XCTAssertEqual(checkRefreshes, [false])
        XCTAssertEqual(syncedProjectIDs, [["owner/repo"]])
        XCTAssertNil(state.repositorySyncError)
    }

    func testRepositorySyncFallsBackToBundledHelperForOlderAgent() async {
        let behind = RepositorySyncItem(
            projectID: "owner/repo", name: "repo", path: "/repo", base: "main", remote: "origin",
            currentBranch: "main", baseWorktree: "/repo", currentAhead: 0, currentBehind: 1,
            defaultAhead: 0, defaultBehind: 1, dirty: false, detached: false,
            needsUpdate: true, canUpdate: true, fetched: false, updated: false,
            state: "behind", action: "fast_forward", reason: "safe", error: nil
        )
        let current = RepositorySyncItem(
            projectID: "owner/repo", name: "repo", path: "/repo", base: "main", remote: "origin",
            currentBranch: "main", baseWorktree: "/repo", currentAhead: 0, currentBehind: 0,
            defaultAhead: 0, defaultBehind: 0, dirty: false, detached: false,
            needsUpdate: false, canUpdate: false, fetched: false, updated: true,
            state: "current", action: "none", reason: "updated", error: nil
        )
        let fallback = RepositorySyncFallbackClient(reports: [
            RepositorySyncReport(checkedAt: "2026-07-14T12:00:00Z", fetchAttempted: false, repositories: [behind]),
            RepositorySyncReport(checkedAt: "2026-07-14T12:01:00Z", fetchAttempted: true, repositories: [current]),
        ])
        let agent = ScriptedAgent(
            events: [TestSnapshots.snapshotEvent(TestSnapshots.empty)],
            repositorySyncError: AgentClientError.command("unknown agent request: get_repository_sync")
        )
        let state = AppState(agent: agent, installer: nil, repositorySyncFallback: fallback)

        await state.checkRepositorySync(refresh: false)
        await state.syncRepositories(["owner/repo"])

        let checkRefreshes = await fallback.checkRefreshes
        let syncedProjectIDs = await fallback.syncedProjectIDs
        XCTAssertEqual(checkRefreshes, [false])
        XCTAssertEqual(syncedProjectIDs, [["owner/repo"]])
        XCTAssertTrue(state.repositoriesNeedingSync.isEmpty)
        XCTAssertNil(state.repositorySyncError)
    }

    func testDependencyLimitsStayIdleUntilExplicitCheck() async {
        let report = DependencyLimitReport(
            checkedAt: "2026-07-14T12:30:00Z",
            dependencies: [DependencyLimit(name: "gh", buckets: [
                DependencyLimitBucket(
                    id: "graphql", name: "GraphQL", limit: 5_000,
                    used: 2_500, remaining: 2_500, resetAt: "2026-07-14T13:00:00Z"
                ),
            ])]
        )
        let client = DependencyLimitsClient(report: report)
        let agent = ScriptedAgent(events: [TestSnapshots.snapshotEvent(TestSnapshots.empty)])
        let state = AppState(
            agent: agent,
            installer: nil,
            repositorySyncFallback: nil,
            dependencyLimitsClient: client
        )

        state.start()
        defer { state.stop() }
        for _ in 0..<20 where state.snapshot == nil {
            try? await Task.sleep(for: .milliseconds(10))
        }
        let callsBeforeCheck = await client.calls
        XCTAssertEqual(callsBeforeCheck, 0)

        await state.checkDependencyLimits()

        let callsAfterCheck = await client.calls
        XCTAssertEqual(callsAfterCheck, 1)
        XCTAssertEqual(state.dependencyUsagePercent, 50)
        XCTAssertEqual(state.dependencyUsageLevel, .warning)
        XCTAssertNil(state.dependencyLimitsError)
        XCTAssertFalse(state.isCheckingDependencyLimits)
    }

    func testAgentEventsRejectOlderProjectRevision() async {
        let agent = ScriptedAgent(events: [
            TestSnapshots.agentEvent(snapshot: TestSnapshots.withLane, projectID: "owner/repo", revision: 2),
            TestSnapshots.agentEvent(snapshot: TestSnapshots.empty, projectID: "owner/repo", revision: 1),
        ])
        let state = AppState(agent: agent, installer: nil)
        state.start()
        defer { state.stop() }
        for _ in 0..<30 where state.snapshot == nil {
            try? await Task.sleep(for: .milliseconds(10))
        }
        XCTAssertEqual(state.snapshot, TestSnapshots.withLane)
    }

    func testAgentEventsRejectDifferentScanWhileRefreshIsActive() async {
        let agent = ScriptedAgent(
            events: [
                TestSnapshots.snapshotEvent(TestSnapshots.withLane),
                TestSnapshots.agentEvent(snapshot: TestSnapshots.empty, projectID: "owner/repo", revision: 2, scanID: "older-scan"),
            ],
            statusDetails: AgentStatusDetails(
                running: true, pid: 1, startedAt: nil, refreshing: true,
                scanID: "scan", projectCount: 1, socket: "/socket"
            )
        )
        let state = AppState(agent: agent, installer: nil)
        state.start()
        defer { state.stop() }
        for _ in 0..<30 where state.snapshot == nil {
            try? await Task.sleep(for: .milliseconds(10))
        }
        XCTAssertEqual(state.snapshot, TestSnapshots.withLane)
    }

    func testAgentDisconnectPreservesLastGoodSnapshot() async {
        let agent = ScriptedAgent(
            events: [TestSnapshots.agentEvent(snapshot: TestSnapshots.withLane, projectID: "owner/repo", revision: 1)],
            terminalError: TestError.failed
        )
        let state = AppState(agent: agent, installer: nil)
        state.start()
        defer { state.stop() }
        for _ in 0..<30 where state.lastError == nil {
            try? await Task.sleep(for: .milliseconds(10))
        }
        XCTAssertEqual(state.snapshot, TestSnapshots.withLane)
        XCTAssertNotNil(state.lastError)
        XCTAssertFalse(state.agentAvailable)
    }

    func testUncachedDiscoveredProjectAppearsAsLoadingPlaceholder() async {
        let placeholder = AgentProjectStatus(
            projectID: "owner/new-repo",
            name: "new-repo",
            path: "/Users/test/new-repo",
            trackingState: "tracked",
            stage: "queued",
            revision: 1,
            updatedAt: "2026-07-11T16:00:00Z",
            mutedAt: nil,
            lastProbeAt: nil
        )
        let discovery = AgentEvent(
            protocolVersion: 1,
            requestID: nil,
            type: "project_discovered",
            scanID: "scan",
            projectID: placeholder.projectID,
            revision: 1,
            stage: "queued",
            generatedAt: placeholder.updatedAt,
            message: nil,
            snapshot: nil,
            projects: [placeholder],
            status: nil,
            notes: nil
        )
        let agent = ScriptedAgent(events: [
            TestSnapshots.agentEvent(snapshot: TestSnapshots.empty, projectID: "", revision: 0),
            discovery,
        ])
        let state = AppState(agent: agent, installer: nil)
        state.start()
        defer { state.stop() }
        for _ in 0..<30 where state.loadingProjects.isEmpty {
            try? await Task.sleep(for: .milliseconds(10))
        }
        XCTAssertEqual(state.loadingProjects, [placeholder])
    }
}
