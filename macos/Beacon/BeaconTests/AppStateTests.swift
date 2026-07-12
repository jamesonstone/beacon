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

    func testQuietProjectsExcludeProjectsWithActiveWorkAndSupportSearch() async {
        let state = AppState(client: StubClient(result: .success(TestSnapshots.withIdleInventory)))
        await state.scan()

        XCTAssertEqual(state.quietProjectCount, 1)
        XCTAssertEqual(state.quietProjectGroups().map(\.name), ["quiet"])
        XCTAssertEqual(state.quietProjectGroups().first?.lanes.map(\.id), ["quiet-base", "quiet-worktree"])
        XCTAssertEqual(state.quietProjectGroups(matching: "WORKTREE").first?.lanes.count, 2)
        XCTAssertTrue(state.quietProjectGroups(matching: "missing").isEmpty)
        XCTAssertEqual(state.topLane()?.id, "active-work")
    }

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

    func testStartScansBeforeMenuOpens() async {
        let state = AppState(client: StubClient(result: .success(TestSnapshots.empty)))
        state.start()
        defer { state.stop() }
        for _ in 0..<20 where state.snapshot == nil {
            try? await Task.sleep(for: .milliseconds(10))
        }
        XCTAssertEqual(state.snapshot, TestSnapshots.empty)
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
        let agent = ScriptedAgent(events: [
            TestSnapshots.snapshotEvent(TestSnapshots.withLane),
            TestSnapshots.agentEvent(snapshot: TestSnapshots.empty, projectID: "owner/repo", revision: 2, scanID: "older-scan"),
        ])
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

private struct StubClient: CLIClientProtocol {
    let result: Result<BeaconSnapshot, Error>
    func scan() async throws -> BeaconSnapshot { try result.get() }
    func setProjectTracked(_ github: String, tracked: Bool) async throws {}
}

private actor SequenceClient: CLIClientProtocol {
    var results: [Result<BeaconSnapshot, Error>]

    init(results: [Result<BeaconSnapshot, Error>]) {
        self.results = results
    }

    func scan() async throws -> BeaconSnapshot {
        try results.removeFirst().get()
    }

    func setProjectTracked(_ github: String, tracked: Bool) async throws {}
}

private struct TrackingCall: Equatable {
    let github: String
    let tracked: Bool
}

private actor RecordingClient: CLIClientProtocol {
    var results: [Result<BeaconSnapshot, Error>]
    private(set) var trackingCalls: [TrackingCall] = []
    private(set) var maximumConcurrentTrackingCalls = 0
    private var activeTrackingCalls = 0
    private var trackingResults: [Result<Void, Error>]
    private let trackingDelay: Duration

    init(
        results: [Result<BeaconSnapshot, Error>],
        trackingResults: [Result<Void, Error>] = [],
        trackingDelay: Duration = .zero
    ) {
        self.results = results
        self.trackingResults = trackingResults
        self.trackingDelay = trackingDelay
    }

    func scan() async throws -> BeaconSnapshot {
        try results.removeFirst().get()
    }

    func setProjectTracked(_ github: String, tracked: Bool) async throws {
        trackingCalls.append(TrackingCall(github: github, tracked: tracked))
        activeTrackingCalls += 1
        maximumConcurrentTrackingCalls = max(maximumConcurrentTrackingCalls, activeTrackingCalls)
        defer { activeTrackingCalls -= 1 }
        if trackingDelay > .zero {
            try await Task.sleep(for: trackingDelay)
        }
        if !trackingResults.isEmpty {
            try trackingResults.removeFirst().get()
        }
    }
}

private enum TestError: Error { case failed }

private actor ScriptedAgent: AgentClientProtocol {
    let events: [AgentEvent]
    let terminalError: Error?

    init(events: [AgentEvent], terminalError: Error? = nil) {
        self.events = events
        self.terminalError = terminalError
    }

    func snapshot() async throws -> AgentEvent {
        try XCTUnwrap(events.first)
    }

    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error> {
        let values = events
        let failure = terminalError
        return AsyncThrowingStream { continuation in
            for event in values { continuation.yield(event) }
            if let failure {
                continuation.finish(throwing: failure)
            } else {
                continuation.finish()
            }
        }
    }

    func refresh(project: String?) async throws -> String { "scan" }
    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent {
        try XCTUnwrap(events.first)
    }
    func status() async throws -> AgentStatusDetails {
        AgentStatusDetails(running: true, pid: 1, startedAt: nil, refreshing: false, scanID: nil, projectCount: 1, socket: "/socket")
    }
}

private enum TestSnapshots {
    static let empty = empty()

    static func trackedInventory(count: Int) -> BeaconSnapshot {
        let projects = (0..<count).map { index in
            BeaconProject(
                name: "project-\(index)",
                path: "/Users/test/project-\(index)",
                github: "owner/project-\(index)",
                base: "main",
                remote: "origin",
                trackingState: "tracked",
                progress: nil,
                laneIDs: [],
                errors: []
            )
        }
        return BeaconSnapshot(
            schemaVersion: 2,
            generatedAt: "2026-07-12T13:00:00Z",
            configPath: "/Users/test/.config/beacon/config.yaml",
            tracking: nil,
            refresh: [],
            summary: SnapshotSummary(
                projects: count,
                trackedProjects: count,
                untrackedProjects: 0,
                total: 0,
                reviewReady: 0,
                needsAction: 0,
                waiting: 0,
                idle: 0,
                errors: 0,
                openIssues: 0,
                unresolvedFeedback: 0
            ),
            groups: LaneGroups(ready: [], action: [], waiting: [], idle: [], untracked: []),
            projects: projects,
            lanes: [],
            errors: []
        )
    }

    static func empty(schemaVersion: Int = 2) -> BeaconSnapshot {
        BeaconSnapshot(
            schemaVersion: schemaVersion,
            generatedAt: "2026-07-09T16:00:00Z",
            configPath: "/Users/test/.config/beacon/config.yaml",
            tracking: nil,
            refresh: [],
            summary: SnapshotSummary(
                projects: 0,
                trackedProjects: 0,
                untrackedProjects: 0,
                total: 0,
                reviewReady: 0,
                needsAction: 0,
                waiting: 0,
                idle: 0,
                errors: 0,
                openIssues: 0,
                unresolvedFeedback: 0
            ),
            groups: LaneGroups(ready: [], action: [], waiting: [], idle: [], untracked: []),
            projects: [],
            lanes: [],
            errors: []
        )
    }

    static func agentEvent(snapshot: BeaconSnapshot, projectID: String, revision: UInt64, scanID: String = "scan") -> AgentEvent {
        AgentEvent(
            protocolVersion: 1,
            requestID: nil,
            type: "project_updated",
            scanID: scanID,
            projectID: projectID,
            revision: revision,
            stage: "ready",
            generatedAt: snapshot.generatedAt,
            message: nil,
            snapshot: snapshot,
            projects: nil,
            status: nil
        )
    }

    static func snapshotEvent(_ snapshot: BeaconSnapshot) -> AgentEvent {
        AgentEvent(
            protocolVersion: 1,
            requestID: nil,
            type: "snapshot",
            scanID: nil,
            projectID: nil,
            revision: nil,
            stage: "cached",
            generatedAt: snapshot.generatedAt,
            message: nil,
            snapshot: snapshot,
            projects: nil,
            status: nil
        )
    }

    static let issue = IssueDetails(
        number: 41,
        title: "Feature issue",
        url: "https://example.test/issues/41",
        labels: [],
        assignees: ["me"],
        updatedAt: "2026-07-09T15:00:00Z"
    )

    static let worktree = WorktreeDetails(
        path: "/Users/test/repo",
        headOID: "abc",
        upstream: "origin/feature",
        staged: 0,
        unstaged: 0,
        untracked: 0,
        conflicted: 0,
        ahead: 0,
        behind: 0,
        aheadBase: 1,
        behindBase: 0,
        detached: false,
        locked: false,
        prunable: false,
        updatedAt: "2026-07-09T16:00:00Z"
    )

    static let pullRequest = PullRequestDetails(
        number: 42,
        title: "Feature",
        url: "https://example.test/pull/42",
        headRefName: "feature",
        headRefOID: "abc",
        baseRefName: "main",
        isDraft: false,
        updatedAt: "2026-07-09T16:00:00Z",
        reviewDecision: nil,
        mergeStateStatus: "CLEAN",
        mergeable: "MERGEABLE",
        ciState: "success",
        checks: CheckSummary(total: 1, success: 1, pending: 0, failure: 0, skipped: 0, unknown: 0),
        feedback: FeedbackSummary(comments: 0, reviews: 0, approvals: 0, changesRequested: 0, unresolvedThreads: 0),
        closingIssues: [issue]
    )

    static let progress = ProgressDetails(
        source: "kit",
        featureID: "0002",
        feature: "Dashboard",
        phase: "implement",
        summary: "In progress",
        path: "docs/specs/0002/SPEC.md"
    )

    static func lane(
        id: String = "issue:owner/repo#41",
        repository: String = "repo",
        github: String = "owner/repo",
        branch: String = "feature",
        nextAction: String? = nil,
        pullRequest: PullRequestDetails? = nil,
        issue: IssueDetails? = nil,
        worktree: WorktreeDetails? = nil
    ) -> WorkLane {
        WorkLane(
            id: id,
            repository: repository,
            github: github,
            base: "main",
            branch: branch,
            worktree: worktree,
            pullRequest: pullRequest,
            issue: issue,
            progress: progress,
            signals: LaneSignals(
                worktree: worktree == nil ? "not_local" : "clean",
                publication: "published",
                pullRequest: pullRequest == nil ? "none" : "open",
                ci: pullRequest?.ciState ?? "none",
                review: "none",
                merge: "clean",
                freshness: "current",
                issue: issue == nil ? "none" : "open"
            ),
            reviewReady: pullRequest != nil,
            nextAction: nextAction ?? (pullRequest == nil ? "start_issue" : "manual_test_then_merge"),
            reasons: [],
            warnings: [],
            blockers: [],
            updatedAt: "2026-07-09T16:00:00Z"
        )
    }

    static let withLane: BeaconSnapshot = {
        let issueLane = lane(issue: issue)
        return BeaconSnapshot(
            schemaVersion: 2,
            generatedAt: "2026-07-09T16:00:00Z",
            configPath: "/Users/test/.config/beacon/config.yaml",
            tracking: nil,
            refresh: [],
            summary: SnapshotSummary(
                projects: 1,
                trackedProjects: 1,
                untrackedProjects: 0,
                total: 1,
                reviewReady: 0,
                needsAction: 1,
                waiting: 0,
                idle: 0,
                errors: 0,
                openIssues: 1,
                unresolvedFeedback: 0
            ),
            groups: LaneGroups(ready: [], action: [issueLane.id], waiting: [], idle: [], untracked: []),
            projects: [BeaconProject(
                name: "repo",
                path: "/Users/test/repo",
                github: "owner/repo",
                base: "main",
                remote: "origin",
                trackingState: "tracked",
                progress: progress,
                laneIDs: [issueLane.id],
                errors: []
            )],
            lanes: [issueLane],
            errors: []
        )
    }()

    static let withIdleInventory: BeaconSnapshot = {
        let activeWork = lane(
            id: "active-work",
            repository: "active",
            github: "owner/active",
            branch: "feature",
            nextAction: "fix_ci",
            issue: issue
        )
        let activeBase = lane(
            id: "active-base",
            repository: "active",
            github: "owner/active",
            branch: "main",
            nextAction: "none"
        )
        let quietBase = lane(
            id: "quiet-base",
            repository: "quiet",
            github: "owner/quiet",
            branch: "main",
            nextAction: "none"
        )
        let quietWorktree = lane(
            id: "quiet-worktree",
            repository: "quiet",
            github: "owner/quiet",
            branch: "old-worktree",
            nextAction: "none"
        )
        return BeaconSnapshot(
            schemaVersion: 2,
            generatedAt: "2026-07-09T16:00:00Z",
            configPath: "/Users/test/.config/beacon/config.yaml",
            tracking: nil,
            refresh: [],
            summary: SnapshotSummary(
                projects: 2,
                trackedProjects: 2,
                untrackedProjects: 0,
                total: 4,
                reviewReady: 0,
                needsAction: 1,
                waiting: 0,
                idle: 3,
                errors: 0,
                openIssues: 1,
                unresolvedFeedback: 0
            ),
            groups: LaneGroups(
                ready: [],
                action: [activeWork.id],
                waiting: [],
                idle: [activeBase.id, quietBase.id, quietWorktree.id],
                untracked: []
            ),
            projects: [
                BeaconProject(
                    name: "active",
                    path: "/Users/test/active",
                    github: "owner/active",
                    base: "main",
                    remote: "origin",
                    trackingState: "tracked",
                    progress: nil,
                    laneIDs: [activeWork.id, activeBase.id],
                    errors: []
                ),
                BeaconProject(
                    name: "quiet",
                    path: "/Users/test/quiet",
                    github: "owner/quiet",
                    base: "main",
                    remote: "origin",
                    trackingState: "tracked",
                    progress: nil,
                    laneIDs: [quietBase.id, quietWorktree.id],
                    errors: []
                ),
            ],
            lanes: [activeWork, activeBase, quietBase, quietWorktree],
            errors: []
        )
    }()

    static let withTrackingInventory: BeaconSnapshot = trackingInventory(untracked: true)
    static let withTrackedInventory: BeaconSnapshot = trackingInventory(untracked: false)

    private static func trackingInventory(untracked: Bool) -> BeaconSnapshot {
        let trackedLane = lane(
            id: "tracked-base",
            repository: "tracked",
            github: "owner/tracked",
            branch: "main",
            nextAction: "none"
        )
        let untrackedLane = lane(
            id: "untracked-base",
            repository: "untracked",
            github: "owner/untracked",
            branch: "main",
            nextAction: "none"
        )
        return BeaconSnapshot(
            schemaVersion: 2,
            generatedAt: "2026-07-09T16:00:00Z",
            configPath: "/Users/test/.config/beacon/config.yaml",
            tracking: TrackingDetails(
                path: "/Users/test/.config/beacon/tracking.yaml",
                autoReactivated: []
            ),
            refresh: [],
            summary: SnapshotSummary(
                projects: 2,
                trackedProjects: untracked ? 1 : 2,
                untrackedProjects: untracked ? 1 : 0,
                total: untracked ? 1 : 2,
                reviewReady: 0,
                needsAction: 0,
                waiting: 0,
                idle: untracked ? 1 : 2,
                errors: 0,
                openIssues: 0,
                unresolvedFeedback: 0
            ),
            groups: LaneGroups(
                ready: [],
                action: [],
                waiting: [],
                idle: untracked ? [trackedLane.id] : [trackedLane.id, untrackedLane.id],
                untracked: untracked ? [untrackedLane.id] : []
            ),
            projects: [
                BeaconProject(
                    name: "tracked",
                    path: "/Users/test/tracked",
                    github: "owner/tracked",
                    base: "main",
                    remote: "origin",
                    trackingState: "tracked",
                    progress: nil,
                    laneIDs: [trackedLane.id],
                    errors: []
                ),
                BeaconProject(
                    name: "untracked",
                    path: "/Users/test/untracked",
                    github: "owner/untracked",
                    base: "main",
                    remote: "origin",
                    trackingState: untracked ? "untracked" : "tracked",
                    progress: nil,
                    laneIDs: [untrackedLane.id],
                    errors: []
                ),
            ],
            lanes: [trackedLane, untrackedLane],
            errors: []
        )
    }

    static let withProgressGroups: BeaconSnapshot = {
        let progressLane = lane(issue: issue)
        return BeaconSnapshot(
            schemaVersion: 2,
            generatedAt: "2026-07-09T16:00:00Z",
            configPath: "/Users/test/.config/beacon/config.yaml",
            tracking: nil,
            refresh: [],
            summary: SnapshotSummary(
                projects: 1,
                trackedProjects: 1,
                untrackedProjects: 0,
                total: 10,
                reviewReady: 2,
                needsAction: 3,
                waiting: 1,
                idle: 4,
                errors: 0,
                openIssues: 1,
                unresolvedFeedback: 0
            ),
            groups: LaneGroups(
                ready: ["ready-1", "ready-2"],
                action: ["action-1", "action-2", "action-3"],
                waiting: ["waiting-1"],
                idle: ["idle-1", "idle-2", "idle-3", "idle-4"],
                untracked: []
            ),
            projects: [BeaconProject(
                name: "repo",
                path: "/Users/test/repo",
                github: "owner/repo",
                base: "main",
                remote: "origin",
                trackingState: "tracked",
                progress: progress,
                laneIDs: [progressLane.id],
                errors: []
            )],
            lanes: [progressLane],
            errors: []
        )
    }()
}
