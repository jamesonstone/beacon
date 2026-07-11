import XCTest
@testable import Beacon

final class ModelsTests: XCTestCase {
    func testDecodesCompleteSchemaVersionTwo() throws {
        let data = Data(Self.snapshotJSON.utf8)
        let snapshot = try JSONDecoder().decode(BeaconSnapshot.self, from: data)
        XCTAssertEqual(snapshot.schemaVersion, 2)
        XCTAssertEqual(snapshot.projects.first?.progress?.phase, "deliver")
        XCTAssertEqual(snapshot.summary.reviewReady, 1)
        XCTAssertEqual(snapshot.summary.trackedProjects, 1)
        XCTAssertEqual(snapshot.summary.untrackedProjects, 0)
        XCTAssertEqual(snapshot.summary.openIssues, 1)
        XCTAssertEqual(snapshot.tracking?.path, "/Users/test/.config/beacon/tracking.yaml")
        XCTAssertEqual(snapshot.projects.first?.trackingState, "tracked")
        XCTAssertEqual(snapshot.lanes.first?.pullRequest?.number, 42)
        XCTAssertEqual(snapshot.lanes.first?.pullRequest?.checks.success, 2)
        XCTAssertEqual(snapshot.lanes.first?.pullRequest?.feedback.unresolvedThreads, 1)
        XCTAssertEqual(snapshot.lanes.first?.issue?.number, 41)
        XCTAssertEqual(snapshot.lanes.first?.signals.issue, "open")
        XCTAssertEqual(snapshot.groups.ready, ["gh:owner/repo#42"])
    }

    func testDecodesPartialRepositoryErrorWithoutDiscardingLanes() throws {
        var object = try Self.snapshotObject()
        object["errors"] = [["repository": "owner/repo", "stage": "github", "message": "timeout"]]
        var projects = try XCTUnwrap(object["projects"] as? [[String: Any]])
        projects[0]["errors"] = [["repository": "owner/repo", "stage": "github", "message": "timeout"]]
        object["projects"] = projects

        let data = try JSONSerialization.data(withJSONObject: object)
        let snapshot = try JSONDecoder().decode(BeaconSnapshot.self, from: data)

        XCTAssertEqual(snapshot.lanes.count, 1)
        XCTAssertEqual(snapshot.errors.first?.stage, "github")
        XCTAssertEqual(snapshot.projects.first?.errors.first?.message, "timeout")
    }

    func testDecodesFutureUnknownSignalsAndActions() throws {
        var object = try Self.snapshotObject()
        var lanes = try XCTUnwrap(object["lanes"] as? [[String: Any]])
        var signals = try XCTUnwrap(lanes[0]["signals"] as? [String: Any])
        signals["review"] = "awaiting_robot"
        lanes[0]["signals"] = signals
        lanes[0]["next_action"] = "inspect_with_agent"
        object["lanes"] = lanes

        let data = try JSONSerialization.data(withJSONObject: object)
        let snapshot = try JSONDecoder().decode(BeaconSnapshot.self, from: data)

        XCTAssertEqual(snapshot.lanes.first?.signals.review, "awaiting_robot")
        XCTAssertEqual(snapshot.lanes.first?.nextAction, "inspect_with_agent")
    }

    func testDecodesEarlierSchemaTwoSnapshotWithoutTrackingAdditions() throws {
        var object = try Self.snapshotObject()
        object.removeValue(forKey: "tracking")
        var summary = try XCTUnwrap(object["summary"] as? [String: Any])
        summary.removeValue(forKey: "tracked_projects")
        summary.removeValue(forKey: "untracked_projects")
        object["summary"] = summary
        var groups = try XCTUnwrap(object["groups"] as? [String: Any])
        groups.removeValue(forKey: "untracked")
        object["groups"] = groups
        var projects = try XCTUnwrap(object["projects"] as? [[String: Any]])
        projects[0].removeValue(forKey: "tracking_state")
        object["projects"] = projects

        let data = try JSONSerialization.data(withJSONObject: object)
        let snapshot = try JSONDecoder().decode(BeaconSnapshot.self, from: data)

        XCTAssertNil(snapshot.tracking)
        XCTAssertNil(snapshot.summary.trackedProjects)
        XCTAssertNil(snapshot.groups.untracked)
        XCTAssertTrue(try XCTUnwrap(snapshot.projects.first).isTracked)
    }

    func testCommandPathIncludesCommonHomebrewLocationsOnce() {
        let path = CLIClient.commandPath(existing: "/usr/bin:/opt/homebrew/bin")
        XCTAssertTrue(path.hasPrefix("/opt/homebrew/bin:/usr/local/bin"))
        XCTAssertEqual(path.components(separatedBy: ":").filter { $0 == "/opt/homebrew/bin" }.count, 1)
    }

    func testBundledHelperUsesDistinctExecutableName() {
        XCTAssertEqual(CLIClient.defaultExecutableURL().lastPathComponent, "beacon-cli")
    }

    private static func snapshotObject() throws -> [String: Any] {
        let object = try JSONSerialization.jsonObject(with: Data(snapshotJSON.utf8))
        return try XCTUnwrap(object as? [String: Any])
    }

    private static let snapshotJSON = #"""
    {
      "schema_version": 2,
      "generated_at": "2026-07-09T16:00:00Z",
      "config_path": "/Users/test/.config/beacon/config.yaml",
      "tracking": {"path": "/Users/test/.config/beacon/tracking.yaml", "auto_reactivated": []},
      "refresh": [],
      "summary": {"projects": 1, "tracked_projects": 1, "untracked_projects": 0, "total": 1, "review_ready": 1, "needs_action": 0, "waiting": 0, "idle": 0, "errors": 0, "open_issues": 1, "unresolved_feedback": 1},
      "groups": {"ready": ["gh:owner/repo#42"], "action": [], "waiting": [], "idle": [], "untracked": []},
      "projects": [{
        "name": "repo", "path": "/Users/test/repo", "github": "owner/repo", "base": "main", "remote": "origin",
        "tracking_state": "tracked",
        "progress": {"source": "kit", "feature_id": "0002", "feature": "Dashboard", "phase": "deliver", "summary": "Ready", "path": "docs/specs/0002/SPEC.md"},
        "lane_ids": ["gh:owner/repo#42"], "errors": []
      }],
      "lanes": [{
        "id": "gh:owner/repo#42", "repository": "repo", "github": "owner/repo", "base": "main", "branch": "feature",
        "pull_request": {
          "number": 42, "title": "Feature", "url": "https://example.test/pull/42", "head_ref_name": "feature", "head_ref_oid": "abc", "base_ref_name": "main", "is_draft": false, "updated_at": "2026-07-09T16:00:00Z", "ci_state": "success",
          "checks": {"total": 2, "success": 2, "pending": 0, "failure": 0, "skipped": 0, "unknown": 0},
          "feedback": {"comments": 2, "reviews": 1, "approvals": 0, "changes_requested": 0, "unresolved_threads": 1},
          "closing_issues": [{"number": 41, "title": "Feature issue", "url": "https://example.test/issues/41", "labels": ["enhancement"], "assignees": ["me"], "updated_at": "2026-07-09T15:00:00Z"}]
        },
        "issue": {"number": 41, "title": "Feature issue", "url": "https://example.test/issues/41", "labels": ["enhancement"], "assignees": ["me"], "updated_at": "2026-07-09T15:00:00Z"},
        "progress": {"source": "kit", "feature_id": "0002", "feature": "Dashboard", "phase": "deliver", "summary": "Ready", "path": "docs/specs/0002/SPEC.md"},
        "signals": {"worktree": "not_local", "publication": "published", "pull_request": "open", "ci": "success", "review": "feedback_pending", "merge": "clean", "freshness": "current", "issue": "open"},
        "review_ready": true, "next_action": "address_review", "reasons": [], "warnings": [], "blockers": [], "updated_at": "2026-07-09T16:00:00Z"
      }],
      "errors": []
    }
    """#
}
