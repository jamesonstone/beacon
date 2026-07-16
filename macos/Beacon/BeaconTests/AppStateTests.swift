import XCTest
@testable import Beacon

@MainActor
final class AppStateTests: XCTestCase {
    func testExternalActivityTargetsLaneAndProjectWithoutChangingCounts() {
        let state = AppState(client: StubClient(result: .success(TestSnapshots.empty)))
        let records = [
            ExternalActivityRecord(
                provider: "codex", state: "working", sessionKey: "one",
                projectID: "owner/repo", laneID: "lane-31",
                observedAt: "2026-07-16T12:00:00Z", expiresAt: "2026-07-16T14:00:00Z"
            ),
            ExternalActivityRecord(
                provider: "claude-code", state: "needs_attention", sessionKey: "two",
                projectID: "owner/repo", laneID: nil,
                observedAt: "2026-07-16T12:00:00Z", expiresAt: "2026-07-17T12:00:00Z"
            ),
        ]
        state.applyExternalActivity(ExternalActivitySnapshot(version: 1, records: records, nextExpiry: nil))

        XCTAssertEqual(state.activityChip(projectID: "owner/repo")?.label, "Claude Code · Needs attention")
        XCTAssertEqual(state.activityChip(projectID: "owner/repo", laneID: "lane-31")?.label, "Codex · Working")
        XCTAssertEqual(state.inProgressCount, 0)
        XCTAssertEqual(state.readyCount, 0)
    }

    func testExternalActivityExpiryInvokesGoPruneInsteadOfFilteringInSwift() async {
        let record = ExternalActivityRecord(
            provider: "codex", state: "working", sessionKey: "hashed",
            projectID: "owner/repo", laneID: nil,
            observedAt: "2026-07-16T12:00:00Z", expiresAt: "2026-07-16T14:00:00Z"
        )
        let client = ExternalActivityClientStub(
            initial: ExternalActivitySnapshot(
                version: 1,
                records: [record],
                nextExpiry: "2000-01-01T00:00:00Z"
            )
        )
        let state = AppState(
            agent: ScriptedAgent(events: []), installer: nil,
            notesFallback: nil, repositorySyncFallback: nil,
            dependencyLimitsClient: nil, externalActivityClient: client
        )

        await state.refreshExternalContext()
        for _ in 0..<20 {
            if await client.pruneCalls > 0 { break }
            try? await Task.sleep(for: .milliseconds(10))
        }

        let pruneCalls = await client.pruneCalls
        XCTAssertEqual(pruneCalls, 1)
        XCTAssertEqual(state.externalActivity, .empty)
    }

    func testActivityCacheWatcherObservesAtomicDirectoryChanges() async throws {
        let directory = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent(UUID().uuidString, isDirectory: true)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let observed = expectation(description: "cache directory change")
        observed.assertForOverFulfill = false
        let watcher = try XCTUnwrap(ActivityCacheWatcher(directory: directory) {
            observed.fulfill()
        })

        try Data("{}".utf8).write(to: directory.appendingPathComponent("activity.json"), options: .atomic)

        await fulfillment(of: [observed], timeout: 1)
        watcher.cancel()
    }

    func testSuccessfulScanStoresSnapshot() async {
        let expected = TestSnapshots.empty
        let state = AppState(client: StubClient(result: .success(expected)))
        await state.scan()
        XCTAssertEqual(state.snapshot, expected)
        XCTAssertNil(state.lastError)
        XCTAssertFalse(state.isScanning)
    }

    func testInProgressCountUsesEveryNonIdleGroup() async {
        let state = AppState(client: StubClient(result: .success(TestSnapshots.withProgressGroups)))
        await state.scan()

        XCTAssertEqual(state.inProgressCount, 6)
    }

    func testInProgressCountIsZeroBeforeAndAfterEmptyScan() async {
        let state = AppState(client: StubClient(result: .success(TestSnapshots.empty)))
        XCTAssertEqual(state.inProgressCount, 0)

        await state.scan()

        XCTAssertEqual(state.inProgressCount, 0)
    }

    func testFailedScanPreservesError() async {
        let state = AppState(client: StubClient(result: .failure(TestError.failed)))
        await state.scan()
        XCTAssertNil(state.snapshot)
        XCTAssertNotNil(state.lastError)
    }

    func testFailedScanPreservesLastSuccessfulSnapshot() async {
        let client = SequenceClient(results: [
            .success(TestSnapshots.withLane),
            .failure(TestError.failed),
        ])
        let state = AppState(client: client)

        await state.scan()
        await state.scan()

        XCTAssertEqual(state.snapshot, TestSnapshots.withLane)
        XCTAssertNotNil(state.lastError)
    }

    func testRejectsUnsupportedSchemaWithoutReplacingSnapshot() async {
        let client = SequenceClient(results: [
            .success(TestSnapshots.empty),
            .success(TestSnapshots.empty(schemaVersion: 1)),
        ])
        let state = AppState(client: client)

        await state.scan()
        await state.scan()

        XCTAssertEqual(state.snapshot, TestSnapshots.empty)
        XCTAssertEqual(state.lastError, "Beacon CLI returned invalid JSON: unsupported schema version 1")
    }

    func testProjectGroupingUsesSnapshotProjectOrderAndProgress() async {
        let state = AppState(client: StubClient(result: .success(TestSnapshots.withLane)))
        await state.scan()

        let groups = state.projectGroups(for: TestSnapshots.withLane.lanes)

        XCTAssertEqual(groups.map(\.name), ["repo"])
        XCTAssertEqual(groups.first?.progress?.phase, "implement")
        XCTAssertEqual(groups.first?.lanes.map(\.id), ["issue:owner/repo#41"])
    }

    func testProjectInventoryUsesExplicitFollowStateAndSupportsSearch() async {
        let state = AppState(client: StubClient(result: .success(TestSnapshots.withIdleInventory)))
        await state.scan()

        XCTAssertEqual(state.followedProjects.map(\.github), ["owner/active"])
        XCTAssertEqual(state.quietProjectCount, 1)
        XCTAssertEqual(state.projects(in: "quiet").map(\.name), ["quiet"])
        XCTAssertEqual(state.projects(in: "quiet", matching: "OWNER/QUIET").map(\.name), ["quiet"])
        XCTAssertTrue(state.projects(in: "quiet", matching: "missing").isEmpty)
        XCTAssertEqual(state.topLane()?.id, "active-work")
    }

    func testWorkingSetCountAndTopItemSkipLanesWithoutOpenTargets() async {
        var snapshot = TestSnapshots.withIdleInventory
        snapshot.workingSet = WorkingSetGroups(
            path: "/Users/test/.local/state/beacon/lanes.json",
            active: ["active-base", "active-work"],
            waiting: [],
            recent: [],
            parked: ["quiet-worktree"]
        )
        let state = AppState(client: StubClient(result: .success(snapshot)))

        await state.scan()

        XCTAssertEqual(state.inProgressCount, 2)
        XCTAssertEqual(state.topLane()?.id, "active-work")
        XCTAssertEqual(state.lanes(for: snapshot.workingSet?.parked ?? []).map(\.id), ["quiet-worktree"])
    }

    func testProjectFollowingIsIndependentOfLaneAttentionState() async {
        var snapshot = TestSnapshots.withIdleInventory
        snapshot.workingSet = WorkingSetGroups(
            path: "/Users/test/.local/state/beacon/lanes.json",
            active: ["quiet-base"],
            waiting: [],
            recent: [],
            parked: ["quiet-worktree"]
        )
        let state = AppState(client: StubClient(result: .success(snapshot)))

        await state.scan()

        XCTAssertEqual(state.followedProjects.map(\.github), ["owner/active"])
        XCTAssertEqual(state.quietProjects.map(\.github), ["owner/quiet"])
    }

    func testIgnoreLaneParksThroughSharedAgentAuthority() async throws {
        var active = TestSnapshots.withIdleInventory
        active.workingSet = WorkingSetGroups(
            path: "/Users/test/.local/state/beacon/lanes.json",
            active: ["active-work"],
            waiting: [],
            recent: [],
            parked: []
        )
        var parked = active
        parked.workingSet = WorkingSetGroups(
            path: "/Users/test/.local/state/beacon/lanes.json",
            active: [],
            waiting: [],
            recent: [],
            parked: ["active-work"]
        )
        let agent = RecordingLaneAttentionAgent(
            mutationEvent: TestSnapshots.snapshotEvent(parked)
        )
        let state = AppState(agent: agent, installer: nil)
        state.apply(TestSnapshots.snapshotEvent(active))

        let lane = try XCTUnwrap(active.lanes.first { $0.id == "active-work" })
        await state.ignoreLane(lane)

        let calls = await agent.calls
        XCTAssertEqual(calls, [LaneAttentionCall(id: "active-work", state: "parked")])
        XCTAssertEqual(state.snapshot?.workingSet?.active, [])
        XCTAssertEqual(state.snapshot?.workingSet?.parked, ["active-work"])
        XCTAssertNil(state.lastError)
    }

    func testOpenTargetPrefersPullRequestThenIssueThenWorktree() throws {
        let lane = TestSnapshots.lane(
            pullRequest: TestSnapshots.pullRequest,
            issue: TestSnapshots.issue,
            worktree: TestSnapshots.worktree
        )
        XCTAssertEqual(AppState.openTarget(for: lane)?.absoluteString, "https://example.test/pull/42")

        let issueLane = TestSnapshots.lane(issue: TestSnapshots.issue, worktree: TestSnapshots.worktree)
        XCTAssertEqual(AppState.openTarget(for: issueLane)?.absoluteString, "https://example.test/issues/41")

        let localLane = TestSnapshots.lane(worktree: TestSnapshots.worktree)
        XCTAssertEqual(AppState.openTarget(for: localLane)?.path, "/Users/test/repo")
    }
}
