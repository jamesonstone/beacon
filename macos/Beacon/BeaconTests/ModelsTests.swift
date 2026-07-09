import XCTest
@testable import Beacon

final class ModelsTests: XCTestCase {
    func testDecodesSchemaVersionOne() throws {
        let data = Data(Self.snapshotJSON.utf8)
        let snapshot = try JSONDecoder().decode(BeaconSnapshot.self, from: data)
        XCTAssertEqual(snapshot.schemaVersion, 1)
        XCTAssertEqual(snapshot.summary.reviewReady, 1)
        XCTAssertEqual(snapshot.lanes.first?.pullRequest?.number, 42)
        XCTAssertEqual(snapshot.groups.ready, ["gh:owner/repo#42"])
    }

    func testCommandPathIncludesCommonHomebrewLocationsOnce() {
        let path = CLIClient.commandPath(existing: "/usr/bin:/opt/homebrew/bin")
        XCTAssertTrue(path.hasPrefix("/opt/homebrew/bin:/usr/local/bin"))
        XCTAssertEqual(path.components(separatedBy: ":").filter { $0 == "/opt/homebrew/bin" }.count, 1)
    }

    func testBundledHelperUsesDistinctExecutableName() {
        XCTAssertEqual(CLIClient.defaultExecutableURL().lastPathComponent, "beacon-cli")
    }

    private static let snapshotJSON = #"""
    {
      "schema_version": 1,
      "generated_at": "2026-07-09T16:00:00Z",
      "config_path": "/Users/test/.config/beacon/config.yaml",
      "refresh": [],
      "summary": {"total": 1, "review_ready": 1, "needs_action": 0, "waiting": 0, "idle": 0, "errors": 0},
      "groups": {"ready": ["gh:owner/repo#42"], "action": [], "waiting": [], "idle": []},
      "lanes": [{
        "id": "gh:owner/repo#42", "repository": "repo", "github": "owner/repo", "base": "main", "branch": "feature",
        "pull_request": {"number": 42, "title": "Feature", "url": "https://example.test/42", "head_ref_name": "feature", "head_ref_oid": "abc", "base_ref_name": "main", "is_draft": false, "updated_at": "2026-07-09T16:00:00Z", "ci_state": "success"},
        "signals": {"worktree": "not_local", "publication": "published", "pull_request": "open", "ci": "success", "review": "none", "merge": "clean", "freshness": "current"},
        "review_ready": true, "next_action": "review_pr", "reasons": [], "warnings": [], "blockers": [], "updated_at": "2026-07-09T16:00:00Z"
      }],
      "errors": []
    }
    """#
}
