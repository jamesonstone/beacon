@testable import Beacon

extension TestSnapshots {
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
            schemaVersion: 3,
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
            schemaVersion: 3,
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
}
