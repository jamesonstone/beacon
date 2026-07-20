import XCTest
@testable import Beacon

@MainActor
final class AppStateOrderingTests: XCTestCase {
    func testLaneReorderRejectsCrossTypeAndCrossStatusMove() async {
        var snapshot = TestSnapshots.withIdleInventory
        snapshot.workingSet = WorkingSetGroups(
            path: "/Users/test/.local/state/beacon/lanes.json",
            active: ["active-base", "active-work"],
            waiting: [],
            recent: [],
            parked: ["quiet-worktree"],
            order: ["active-base", "active-work", "quiet-worktree"]
        )
        let agent = RecordingLaneAttentionAgent(mutationEvent: TestSnapshots.snapshotEvent(snapshot))
        let state = AppState(agent: agent, installer: nil)
        state.apply(TestSnapshots.snapshotEvent(snapshot))

        await state.reorderLane("active-work", before: "active-base")
        await state.reorderLane("active-work", before: "quiet-worktree")
        await state.moveLane("active-work", by: 1)

        let calls = await agent.laneOrderCalls
        XCTAssertEqual(calls, [])
    }

    func testLaneReorderSendsCompleteOrderWithinSameProjectAndType() async {
        let first = TestSnapshots.lane(
            id: "issue-one", repository: "active", github: "owner/active",
            branch: "GH-1", issue: TestSnapshots.issue
        )
        let second = TestSnapshots.lane(
            id: "issue-two", repository: "active", github: "owner/active",
            branch: "GH-2", issue: TestSnapshots.issue
        )
        let snapshot = TestSnapshots.workingSetSnapshot(
            lanes: [first, second],
            active: [first.id, second.id]
        )
        let agent = RecordingLaneAttentionAgent(mutationEvent: TestSnapshots.snapshotEvent(snapshot))
        let state = AppState(agent: agent, installer: nil)
        state.apply(TestSnapshots.snapshotEvent(snapshot))

        await state.reorderLane(second.id, before: first.id)

        let calls = await agent.laneOrderCalls
        XCTAssertEqual(calls, [[second.id, first.id]])
    }

    func testLaneReorderRejectsSameTypeAcrossProjects() async {
        let first = TestSnapshots.lane(
            id: "alpha-issue", repository: "alpha", github: "owner/alpha",
            branch: "GH-1", issue: TestSnapshots.issue
        )
        let second = TestSnapshots.lane(
            id: "beta-issue", repository: "beta", github: "owner/beta",
            branch: "GH-2", issue: TestSnapshots.issue
        )
        let snapshot = TestSnapshots.workingSetSnapshot(
            lanes: [first, second],
            active: [first.id, second.id]
        )
        let agent = RecordingLaneAttentionAgent(mutationEvent: TestSnapshots.snapshotEvent(snapshot))
        let state = AppState(agent: agent, installer: nil)
        state.apply(TestSnapshots.snapshotEvent(snapshot))

        await state.reorderLane(second.id, before: first.id)

        let calls = await agent.laneOrderCalls
        XCTAssertEqual(calls, [])
    }

    func testMoveLaneSendsCompleteOrderWithinSameProjectAndType() async {
        let first = TestSnapshots.lane(
            id: "issue-one", repository: "active", github: "owner/active",
            branch: "GH-1", issue: TestSnapshots.issue
        )
        let second = TestSnapshots.lane(
            id: "issue-two", repository: "active", github: "owner/active",
            branch: "GH-2", issue: TestSnapshots.issue
        )
        let snapshot = TestSnapshots.workingSetSnapshot(
            lanes: [first, second],
            active: [first.id, second.id]
        )
        let agent = RecordingLaneAttentionAgent(mutationEvent: TestSnapshots.snapshotEvent(snapshot))
        let state = AppState(agent: agent, installer: nil)
        state.apply(TestSnapshots.snapshotEvent(snapshot))

        await state.moveLane(first.id, by: 1)

        let calls = await agent.laneOrderCalls
        XCTAssertEqual(calls, [[second.id, first.id]])
    }
}
