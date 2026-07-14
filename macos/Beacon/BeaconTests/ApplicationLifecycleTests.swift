import XCTest
@testable import Beacon

@MainActor
final class ApplicationLifecycleTests: XCTestCase {
    func testXCTestHostDoesNotOwnRealAgentLifecycle() {
        XCTAssertTrue(AppDelegate.isRunningUnitTests)
    }

    func testDashboardWindowIsSingletonAndReopensAfterClose() {
        let model = testApplicationModel()

        model.showDashboard(activate: false)
        let firstWindow = model.dashboardWindow
        XCTAssertNotNil(firstWindow)
        XCTAssertTrue(model.isDashboardVisible)

        model.showDashboard(activate: false)
        XCTAssertTrue(firstWindow === model.dashboardWindow)

        firstWindow?.close()
        XCTAssertFalse(model.isDashboardVisible)

        model.showDashboard(activate: false)
        XCTAssertTrue(firstWindow === model.dashboardWindow)
        XCTAssertTrue(model.isDashboardVisible)
        model.dashboardWindow?.close()
    }

    func testDashboardInitialFrameUsesPreferredWidthAndFullVisibleHeight() {
        let visibleFrame = NSRect(x: 10, y: 40, width: 1_400, height: 860)

        let frame = DashboardWindowController.initialFrame(in: visibleFrame)

        XCTAssertEqual(frame.width, 580)
        XCTAssertEqual(frame.height, visibleFrame.height)
        XCTAssertEqual(frame.midX, visibleFrame.midX)
        XCTAssertEqual(frame.minY, visibleFrame.minY)
    }

    func testNormalLaunchOpensDashboardAndLoginLaunchStaysQuiet() {
        let normal = testApplicationModel()
        normal.handleLaunch(isLoginLaunch: false)
        XCTAssertTrue(normal.isDashboardVisible)
        normal.dashboardWindow?.close()
        normal.stop()

        let login = testApplicationModel()
        login.handleLaunch(isLoginLaunch: true)
        XCTAssertNil(login.dashboardWindow)
        login.stop()
    }

    func testApplicationStartAndTerminationOwnAgentLifecycle() async {
        let lifecycle = StubAgentLifecycleController()
        let state = AppState(
            agent: ScriptedAgent(events: [TestSnapshots.snapshotEvent(TestSnapshots.empty)]),
            installer: lifecycle,
            notesFallback: nil,
            repositorySyncFallback: nil,
            dependencyLimitsClient: nil
        )
        let model = BeaconApplicationModel(
            state: state,
            loginItem: LoginItemController(service: StubLoginItemService(status: .disabled))
        )

        model.start()
        model.start()
        XCTAssertEqual(lifecycle.startCount, 1)
        XCTAssertNil(model.terminate())
        XCTAssertEqual(lifecycle.stopCount, 1)
        XCTAssertFalse(state.agentAvailable)
    }

    func testLoginItemControllerRegistersAndUnregisters() {
        let service = StubLoginItemService(status: .disabled)
        let controller = LoginItemController(service: service)

        controller.setEnabled(true)
        XCTAssertEqual(controller.status, .enabled)
        XCTAssertEqual(service.registerCount, 1)

        controller.setEnabled(false)
        XCTAssertEqual(controller.status, .disabled)
        XCTAssertEqual(service.unregisterCount, 1)
    }

    func testLoginItemControllerPreservesApprovalAndErrors() {
        let service = StubLoginItemService(status: .requiresApproval)
        service.registerError = TestLifecycleError.failed
        let controller = LoginItemController(service: service)

        XCTAssertTrue(controller.requiresApproval)
        controller.setEnabled(true)

        XCTAssertEqual(controller.status, .requiresApproval)
        XCTAssertNotNil(controller.errorMessage)
        controller.openSystemSettings()
        XCTAssertEqual(service.settingsCount, 1)
    }

    private func testApplicationModel() -> BeaconApplicationModel {
        BeaconApplicationModel(
            state: AppState(client: LifecycleSnapshotClient()),
            loginItem: LoginItemController(service: StubLoginItemService(status: .disabled))
        )
    }
}

private actor LifecycleSnapshotClient: CLIClientProtocol {
    func scan() async throws -> BeaconSnapshot {
        BeaconSnapshot(
            schemaVersion: 3,
            generatedAt: "2026-07-12T12:00:00Z",
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

    func setProjectTracked(_ github: String, tracked: Bool) async throws {}
}

private final class StubLoginItemService: LoginItemServiceProtocol {
    var status: LoginItemStatus
    var registerError: Error?
    var unregisterError: Error?
    private(set) var registerCount = 0
    private(set) var unregisterCount = 0
    private(set) var settingsCount = 0

    init(status: LoginItemStatus) {
        self.status = status
    }

    func register() throws {
        registerCount += 1
        if let registerError { throw registerError }
        status = .enabled
    }

    func unregister() throws {
        unregisterCount += 1
        if let unregisterError { throw unregisterError }
        status = .disabled
    }

    func openSystemSettings() {
        settingsCount += 1
    }
}

private enum TestLifecycleError: Error {
    case failed
}

private final class StubAgentLifecycleController: AgentLifecycleControllerProtocol {
    private(set) var startCount = 0
    private(set) var installCount = 0
    private(set) var stopCount = 0

    func startAgent() throws {
        startCount += 1
    }

    func installAgent() async throws {
        installCount += 1
    }

    func stopAgent() throws {
        stopCount += 1
    }
}
