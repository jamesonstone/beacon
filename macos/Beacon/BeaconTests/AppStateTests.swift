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
}

private struct StubClient: CLIClientProtocol {
    let result: Result<BeaconSnapshot, Error>
    func scan() async throws -> BeaconSnapshot { try result.get() }
}

private actor SequenceClient: CLIClientProtocol {
    var results: [Result<BeaconSnapshot, Error>]

    init(results: [Result<BeaconSnapshot, Error>]) {
        self.results = results
    }

    func scan() async throws -> BeaconSnapshot {
        try results.removeFirst().get()
    }
}

private enum TestError: Error { case failed }

private enum TestSnapshots {
    static let empty = empty()

    static func empty(schemaVersion: Int = 2) -> BeaconSnapshot {
        BeaconSnapshot(
            schemaVersion: schemaVersion,
            generatedAt: "2026-07-09T16:00:00Z",
            configPath: "/Users/test/.config/beacon/config.yaml",
            refresh: [],
            summary: SnapshotSummary(
                projects: 0,
                total: 0,
                reviewReady: 0,
                needsAction: 0,
                waiting: 0,
                idle: 0,
                errors: 0,
                openIssues: 0,
                unresolvedFeedback: 0
            ),
            groups: LaneGroups(ready: [], action: [], waiting: [], idle: []),
            projects: [],
            lanes: [],
            errors: []
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
        pullRequest: PullRequestDetails? = nil,
        issue: IssueDetails? = nil,
        worktree: WorktreeDetails? = nil
    ) -> WorkLane {
        WorkLane(
            id: "issue:owner/repo#41",
            repository: "repo",
            github: "owner/repo",
            base: "main",
            branch: "feature",
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
            nextAction: pullRequest == nil ? "start_issue" : "manual_test_then_merge",
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
            refresh: [],
            summary: SnapshotSummary(
                projects: 1,
                total: 1,
                reviewReady: 0,
                needsAction: 1,
                waiting: 0,
                idle: 0,
                errors: 0,
                openIssues: 1,
                unresolvedFeedback: 0
            ),
            groups: LaneGroups(ready: [], action: [issueLane.id], waiting: [], idle: []),
            projects: [BeaconProject(
                name: "repo",
                path: "/Users/test/repo",
                github: "owner/repo",
                base: "main",
                remote: "origin",
                progress: progress,
                laneIDs: [issueLane.id],
                errors: []
            )],
            lanes: [issueLane],
            errors: []
        )
    }()

    static let withProgressGroups: BeaconSnapshot = {
        let progressLane = lane(issue: issue)
        return BeaconSnapshot(
            schemaVersion: 2,
            generatedAt: "2026-07-09T16:00:00Z",
            configPath: "/Users/test/.config/beacon/config.yaml",
            refresh: [],
            summary: SnapshotSummary(
                projects: 1,
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
                idle: ["idle-1", "idle-2", "idle-3", "idle-4"]
            ),
            projects: [BeaconProject(
                name: "repo",
                path: "/Users/test/repo",
                github: "owner/repo",
                base: "main",
                remote: "origin",
                progress: progress,
                laneIDs: [progressLane.id],
                errors: []
            )],
            lanes: [progressLane],
            errors: []
        )
    }()
}
