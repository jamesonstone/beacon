import XCTest
@testable import Beacon

final class ModelsTests: XCTestCase {
    func testDecodesCompleteSchemaVersionThree() throws {
        let data = Data(Self.snapshotJSON.utf8)
        let snapshot = try JSONDecoder().decode(BeaconSnapshot.self, from: data)
        XCTAssertEqual(snapshot.schemaVersion, 3)
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
        XCTAssertEqual(snapshot.workingSet?.active, ["gh:owner/repo#42"])
        XCTAssertEqual(snapshot.lanes.first?.attention?.delta, "CI changed from pending to success")
        XCTAssertEqual(snapshot.lanes.first?.attention?.tags, ["manual test", "release"])
    }

    func testDashboardViewModesHaveStablePresentationContracts() {
        XCTAssertEqual(DashboardViewMode.allCases.map(\.rawValue), ["stacked", "tiles", "kanban"])
        XCTAssertEqual(DashboardViewMode.stacked.title, "Stacked")
        XCTAssertEqual(DashboardViewMode.tiles.symbol, "rectangle.grid.1x2")
        XCTAssertTrue(DashboardViewMode.kanban.title.contains("Experimental"))
    }

    func testDashboardTabsKeepActiveAsTheStableDefault() {
        XCTAssertEqual(DashboardTab.defaultTab, .active)
        XCTAssertEqual(DashboardTab.allCases.map(\.rawValue), ["active", "parking_lot", "quiet", "untracked"])
        XCTAssertEqual(DashboardTab.parkingLot.title, "Parking Lot")
        XCTAssertEqual(DashboardTab.untracked.symbol, "eye.slash.fill")
    }

    func testEvidenceBadgeDismissalsAreExactValueScopedAndDeterministic() {
        let none = EvidenceBadgeDismissals.key(laneID: "lane-1", dimension: "CI", value: "none")
        let success = EvidenceBadgeDismissals.key(laneID: "lane-1", dimension: "ci", value: "success")
        let anotherLane = EvidenceBadgeDismissals.key(laneID: "lane-2", dimension: "ci", value: "none")

        XCTAssertNotEqual(none, success)
        XCTAssertNotEqual(none, anotherLane)

        let encoded = EvidenceBadgeDismissals.encode([success, none])
        XCTAssertEqual(encoded, EvidenceBadgeDismissals.encode([none, success]))
        XCTAssertEqual(EvidenceBadgeDismissals.decode(encoded), [none, success])
        XCTAssertTrue(EvidenceBadgeDismissals.decode("not-json").isEmpty)
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

    func testDecodesAgentProtocolEventAndProjectStage() throws {
        let object: [String: Any] = [
            "protocol_version": 1,
            "type": "project_updated",
            "scan_id": "scan-1",
            "project_id": "owner/repo",
            "revision": 7,
            "stage": "github",
            "generated_at": "2026-07-11T16:00:00Z",
            "projects": [[
                "project_id": "owner/repo",
                "name": "repo",
                "path": "/repo",
                "tracking_state": "muted",
                "stage": "github",
                "revision": 7,
                "updated_at": "2026-07-11T16:00:00Z",
                "muted_at": "2026-07-11T15:00:00Z",
                "last_probe_at": "2026-07-11T15:50:00Z",
            ]],
        ]
        let data = try JSONSerialization.data(withJSONObject: object)
        let event = try JSONDecoder().decode(AgentEvent.self, from: data)
        XCTAssertEqual(event.protocolVersion, 1)
        XCTAssertEqual(event.projects?.first?.stage, "github")
        XCTAssertEqual(event.projects?.first?.trackingState, "muted")
        XCTAssertEqual(event.projects?.first?.lastProbeAt, "2026-07-11T15:50:00Z")
    }

    func testDefaultAgentSocketUsesCacheDirectory() {
        XCTAssertTrue(AgentClient.defaultSocketPath().hasSuffix("/.cache/beacon/agent.sock"))
    }

    private static func snapshotObject() throws -> [String: Any] {
        let object = try JSONSerialization.jsonObject(with: Data(snapshotJSON.utf8))
        return try XCTUnwrap(object as? [String: Any])
    }

    private static let snapshotJSON = #"""
    {
      "schema_version": 3,
      "generated_at": "2026-07-09T16:00:00Z",
      "config_path": "/Users/test/.config/beacon/config.yaml",
      "tracking": {"path": "/Users/test/.config/beacon/tracking.yaml", "auto_reactivated": []},
      "refresh": [],
      "summary": {"projects": 1, "tracked_projects": 1, "untracked_projects": 0, "total": 1, "review_ready": 1, "needs_action": 0, "waiting": 0, "idle": 0, "errors": 0, "open_issues": 1, "unresolved_feedback": 1, "active_lanes": 1, "recent_lanes": 0, "parked_lanes": 0},
      "groups": {"ready": ["gh:owner/repo#42"], "action": [], "waiting": [], "idle": [], "untracked": []},
      "working_set": {"path": "/Users/test/.local/state/beacon/lanes.json", "active": ["gh:owner/repo#42"], "waiting": [], "recent": [], "parked": []},
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
        "review_ready": true, "next_action": "address_review", "reasons": [], "warnings": [], "blockers": [], "updated_at": "2026-07-09T16:00:00Z",
        "attention": {"state": "active", "pinned": true, "manual": false, "tags": ["manual test", "release"], "note": "test manually", "note_stale": true, "delta": "CI changed from pending to success", "previous": {}, "current": {}}
      }],
      "errors": []
    }
    """#
}
