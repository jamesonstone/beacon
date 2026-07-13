import XCTest
@testable import Beacon

extension AppStateTests {
    func testProjectTrackingInventorySeparatesTrackedAndUntrackedProjects() async {
        let state = AppState(client: StubClient(result: .success(TestSnapshots.withTrackingInventory)))
        await state.scan()

        XCTAssertEqual(state.trackedProjects.map(\.github), ["owner/tracked"])
        XCTAssertEqual(state.untrackedProjects.map(\.github), ["owner/untracked"])
        XCTAssertEqual(state.untrackedProjectCount, 1)
        XCTAssertEqual(state.inProgressCount, 0)
    }

    func testProjectTrackingMutationRunsCLIAndRefreshesSnapshot() async {
        let client = RecordingClient(results: [
            .success(TestSnapshots.withTrackingInventory),
            .success(TestSnapshots.withTrackedInventory),
        ])
        let state = AppState(client: client)
        await state.scan()
        let project = try! XCTUnwrap(state.untrackedProjects.first)

        state.setProjectTracked(project, tracked: true)

        XCTAssertTrue(state.untrackedProjects.isEmpty)
        XCTAssertEqual(state.untrackedProjectCount, 0)
        XCTAssertEqual(state.queuedTrackingCount, 1)
        for _ in 0..<50 where state.queuedTrackingCount > 0 {
            try? await Task.sleep(for: .milliseconds(10))
        }

        let calls = await client.trackingCalls
        XCTAssertEqual(calls, [TrackingCall(github: "owner/untracked", tracked: true)])
        XCTAssertTrue(state.untrackedProjects.isEmpty)
        XCTAssertEqual(state.queuedTrackingCount, 0)
        XCTAssertNil(state.lastError)
    }

    func testProjectTrackingMutationsQueueImmediatelyAndRunSerially() async {
        let client = RecordingClient(
            results: [
                .success(TestSnapshots.withTrackingInventory),
                .success(TestSnapshots.withTrackingInventory),
                .success(TestSnapshots.withTrackingInventory),
            ],
            trackingDelay: .milliseconds(50)
        )
        let state = AppState(client: client)
        await state.scan()
        let tracked = try! XCTUnwrap(state.trackedProjects.first)
        let untracked = try! XCTUnwrap(state.untrackedProjects.first)

        state.setProjectTracked(untracked, tracked: true)
        state.setProjectTracked(tracked, tracked: false)

        XCTAssertEqual(state.queuedTrackingCount, 2)
        XCTAssertEqual(state.untrackedProjectCount, 1)
        XCTAssertEqual(state.trackedProjects.map(\.github), ["owner/untracked"])
        XCTAssertEqual(state.untrackedProjects.map(\.github), ["owner/tracked"])

        for _ in 0..<100 where state.queuedTrackingCount > 0 {
            try? await Task.sleep(for: .milliseconds(10))
        }
        let trackingCalls = await client.trackingCalls
        let maximumConcurrentCalls = await client.maximumConcurrentTrackingCalls
        XCTAssertEqual(trackingCalls, [
            TrackingCall(github: "owner/untracked", tracked: true),
            TrackingCall(github: "owner/tracked", tracked: false),
        ])
        XCTAssertEqual(maximumConcurrentCalls, 1)
        XCTAssertEqual(state.queuedTrackingCount, 0)
    }

    func testTwentyProjectsCanBeQueuedWithoutWaiting() async {
        let initial = TestSnapshots.trackedInventory(count: 20)
        let client = RecordingClient(
            results: Array(repeating: .success(initial), count: 21),
            trackingDelay: .milliseconds(2)
        )
        let state = AppState(client: client)
        await state.scan()
        let projects = state.trackedProjects

        for project in projects {
            state.setProjectTracked(project, tracked: false)
        }

        XCTAssertEqual(state.queuedTrackingCount, 20)
        XCTAssertTrue(state.trackedProjects.isEmpty)
        XCTAssertEqual(state.untrackedProjects.count, 20)

        for _ in 0..<200 where state.queuedTrackingCount > 0 {
            try? await Task.sleep(for: .milliseconds(10))
        }
        let calls = await client.trackingCalls
        let maximumConcurrentCalls = await client.maximumConcurrentTrackingCalls
        XCTAssertEqual(calls.map(\.github), projects.map(\.github))
        XCTAssertEqual(maximumConcurrentCalls, 1)
        XCTAssertEqual(state.queuedTrackingCount, 0)
    }

    func testProjectTrackingQueueContinuesAfterFailure() async {
        let client = RecordingClient(
            results: [
                .success(TestSnapshots.withTrackingInventory),
                .success(TestSnapshots.withTrackingInventory),
            ],
            trackingResults: [.failure(TestError.failed), .success(())]
        )
        let state = AppState(client: client)
        await state.scan()
        let tracked = try! XCTUnwrap(state.trackedProjects.first)
        let untracked = try! XCTUnwrap(state.untrackedProjects.first)

        state.setProjectTracked(tracked, tracked: false)
        state.setProjectTracked(untracked, tracked: true)

        for _ in 0..<100 where state.queuedTrackingCount > 0 {
            try? await Task.sleep(for: .milliseconds(10))
        }
        let trackingCallCount = await client.trackingCalls.count
        XCTAssertEqual(trackingCallCount, 2)
        XCTAssertEqual(state.queuedTrackingCount, 0)
        XCTAssertTrue(state.lastError?.contains("owner/tracked") == true)
    }

}
