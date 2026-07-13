@testable import Beacon

enum TestSnapshots {
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
                followState: "following",
                progress: nil,
                laneIDs: [],
                errors: []
            )
        }
        return BeaconSnapshot(
            schemaVersion: 3,
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

    static func empty(schemaVersion: Int = 3) -> BeaconSnapshot {
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

    static let withTrackingInventory: BeaconSnapshot = trackingInventory(untracked: true)
    static let withRecentInventory: BeaconSnapshot = trackingInventory(untracked: true, outsideState: "recent")
    static let withTrackedInventory: BeaconSnapshot = trackingInventory(untracked: false)

    private static func trackingInventory(untracked: Bool, outsideState: String = "quiet") -> BeaconSnapshot {
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
            schemaVersion: 3,
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
                    followState: "following",
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
                    followState: untracked ? outsideState : "following",
                    lastActivityAt: outsideState == "recent" ? "2026-07-12T12:30:00Z" : nil,
                    activityReason: outsideState == "recent" ? "new GitHub activity" : nil,
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
            schemaVersion: 3,
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
