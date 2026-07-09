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

    func testFailedScanPreservesError() async {
        let state = AppState(client: StubClient(result: .failure(TestError.failed)))
        await state.scan()
        XCTAssertNil(state.snapshot)
        XCTAssertNotNil(state.lastError)
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

private enum TestError: Error { case failed }

private enum TestSnapshots {
    static let empty = BeaconSnapshot(
        schemaVersion: 1,
        generatedAt: "2026-07-09T16:00:00Z",
        configPath: "/Users/test/.config/beacon/config.yaml",
        refresh: [],
        summary: SnapshotSummary(total: 0, reviewReady: 0, needsAction: 0, waiting: 0, idle: 0, errors: 0),
        groups: LaneGroups(ready: [], action: [], waiting: [], idle: []),
        lanes: [],
        errors: []
    )
}
