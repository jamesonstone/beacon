import XCTest
@testable import Beacon

extension ModelsTests {
    @MainActor
    func testQuickSwitcherFilteringSearchesTitlesDetailsAndKeywords() {
        let command = BeaconCommandItem(
            id: "note-refactor", title: "Generate endpoints refactor",
            detail: "Closed · detail-1", symbol: "doc.text",
            keywords: "signal note labcore", action: {}
        )

        XCTAssertTrue(command.matches("endpoints"))
        XCTAssertTrue(command.matches("closed"))
        XCTAssertTrue(command.matches("labcore"))
        XCTAssertFalse(command.matches("swagger"))
    }

    func testSignalNoteCreationLabelMatchesSharedPresentationContract() {
        XCTAssertEqual(
            SignalNotesPresentation.createFromGeneralLabel,
            "Create New Note from Highlighted Text in General"
        )
        XCTAssertEqual(SignalNotesPresentation.createFromGeneralSymbol, "doc.badge.plus")
    }

    func testDefaultAgentSocketUsesCacheDirectory() {
        XCTAssertTrue(AgentClient.defaultSocketPath().hasSuffix("/.cache/beacon/agent.sock"))
    }

    func testGeneralNoteRequestsPreserveLegacyAgentPayloadShape() throws {
        let getRequest = try Self.requestObject(
            AgentClient.noteRequestData(type: "get_notes", noteID: "general")
        )
        XCTAssertEqual(getRequest["type"] as? String, "get_notes")
        XCTAssertNil(getRequest["note_id"])

        let setRequest = try Self.requestObject(
            AgentClient.noteRequestData(type: "set_notes", content: "saved", noteID: "general")
        )
        XCTAssertEqual(setRequest["content"] as? String, "saved")
        XCTAssertNil(setRequest["note_id"])

        let detailRequest = try Self.requestObject(
            AgentClient.noteRequestData(type: "get_notes", noteID: "detail-1")
        )
        XCTAssertEqual(detailRequest["note_id"] as? String, "detail-1")
    }

    func testNotePinRequestsCarryExplicitPinAndOrderFields() throws {
        let pinRequest = try Self.requestObject(
            AgentClient.notePinRequestData(noteID: "detail-1", pinned: false)
        )
        XCTAssertEqual(pinRequest["type"] as? String, "set_note_pinned")
        XCTAssertEqual(pinRequest["note_id"] as? String, "detail-1")
        XCTAssertEqual(pinRequest["pinned"] as? Bool, false)

        let orderRequest = try Self.requestObject(
            AgentClient.pinnedOrderRequestData(noteIDs: ["detail-2", "detail-1"])
        )
        XCTAssertEqual(orderRequest["type"] as? String, "reorder_pinned_notes")
        XCTAssertEqual(orderRequest["note_ids"] as? [String], ["detail-2", "detail-1"])
    }

    func testLaneOrderRequestUsesCompleteLaneIDField() throws {
        let request = try Self.requestObject(
            AgentClient.laneOrderRequestData(laneIDs: ["lane-b", "lane-a"])
        )
        XCTAssertEqual(request["type"] as? String, "reorder_lanes")
        XCTAssertEqual(request["lane_ids"] as? [String], ["lane-b", "lane-a"])
        XCTAssertNil(request["note_ids"])
    }

    func testDecodesRichIssueAndReviewFeedbackDetailsAdditively() throws {
        var object = try Self.snapshotObject()
        var workingSet = try XCTUnwrap(object["working_set"] as? [String: Any])
        workingSet["order"] = ["gh:owner/repo#42"]
        object["working_set"] = workingSet
        var lanes = try XCTUnwrap(object["lanes"] as? [[String: Any]])
        var pullRequest = try XCTUnwrap(lanes[0]["pull_request"] as? [String: Any])
        pullRequest["body"] = "## Summary\nDetails"
        pullRequest["body_truncated"] = false
        var feedback = try XCTUnwrap(pullRequest["feedback"] as? [String: Any])
        feedback["threads_truncated"] = false
        feedback["threads"] = [[
            "id": "thread-1",
            "path": "internal/work.go",
            "line": 42,
            "outdated": false,
            "comments_truncated": false,
            "comments": [[
                "id": "comment-1",
                "author": "reviewer",
                "body": "Please handle retries.",
                "body_truncated": false,
                "url": "https://example.test/comment-1",
                "created_at": "2026-07-18T12:00:00Z",
                "updated_at": "2026-07-18T12:00:00Z",
            ]],
        ]]
        pullRequest["feedback"] = feedback
        lanes[0]["pull_request"] = pullRequest
        var issue = try XCTUnwrap(lanes[0]["issue"] as? [String: Any])
        issue["body"] = "Issue details"
        issue["body_truncated"] = false
        lanes[0]["issue"] = issue
        object["lanes"] = lanes

        let data = try JSONSerialization.data(withJSONObject: object)
        let snapshot = try JSONDecoder().decode(BeaconSnapshot.self, from: data)

        XCTAssertEqual(snapshot.workingSet?.order, ["gh:owner/repo#42"])
        XCTAssertEqual(snapshot.lanes[0].pullRequest?.body, "## Summary\nDetails")
        XCTAssertEqual(snapshot.lanes[0].pullRequest?.feedback.threads?.first?.displayLine, 42)
        XCTAssertEqual(snapshot.lanes[0].pullRequest?.feedback.threads?.first?.comments.first?.author, "reviewer")
        XCTAssertEqual(snapshot.lanes[0].issue?.body, "Issue details")
    }

    static func requestObject(_ data: Data) throws -> [String: Any] {
        let object = try JSONSerialization.jsonObject(with: data)
        return try XCTUnwrap(object as? [String: Any])
    }

    static func snapshotObject() throws -> [String: Any] {
        let object = try JSONSerialization.jsonObject(with: Data(snapshotJSON.utf8))
        return try XCTUnwrap(object as? [String: Any])
    }
}
