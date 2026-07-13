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
            status: nil
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
