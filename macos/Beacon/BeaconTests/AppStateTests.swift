import XCTest
@testable import Beacon

@MainActor
final class AppStateTests: XCTestCase {
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
