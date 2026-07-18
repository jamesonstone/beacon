import XCTest
@testable import Beacon

@MainActor
final class TerminalFeatureTests: XCTestCase {
    func testFramesCoverBothEdgesAndStayInsideOffsetDashboardBounds() {
        let dashboardFrame = NSRect(x: 2_090, y: 80, width: 580, height: 720)

        let top = DropDownTerminalPresentation.visibleFrame(
            in: dashboardFrame,
            edge: .top,
            height: .balanced
        )
        XCTAssertEqual(top, NSRect(x: 2_090, y: 476, width: 580, height: 324))
        XCTAssertTrue(dashboardFrame.contains(top))
        XCTAssertEqual(
            DropDownTerminalPresentation.hiddenFrame(
                in: dashboardFrame,
                edge: .top
            ),
            NSRect(x: 2_090, y: 799, width: 580, height: 1)
        )

        let bottom = DropDownTerminalPresentation.visibleFrame(
            in: dashboardFrame,
            edge: .bottom,
            height: .compact
        )
        XCTAssertEqual(bottom, NSRect(x: 2_090, y: 80, width: 580, height: 216))
        XCTAssertTrue(dashboardFrame.contains(bottom))
        XCTAssertEqual(
            DropDownTerminalPresentation.hiddenFrame(
                in: dashboardFrame,
                edge: .bottom
            ),
            NSRect(x: 2_090, y: 80, width: 580, height: 1)
        )
    }

    func testDashboardBoundsAreClippedToTheirVisibleScreen() {
        let screen = NSRect(x: 0, y: 24, width: 1_440, height: 876)
        let partlyOffscreenDashboard = NSRect(x: -80, y: 100, width: 580, height: 700)

        XCTAssertEqual(
            DropDownTerminalPresentation.clippedContainerFrame(
                partlyOffscreenDashboard,
                to: screen
            ),
            NSRect(x: 0, y: 100, width: 500, height: 700)
        )
        XCTAssertNil(
            DropDownTerminalPresentation.clippedContainerFrame(
                NSRect(x: 2_000, y: 100, width: 580, height: 700),
                to: screen
            )
        )
    }

    func testEveryHeightUsesItsDeclaredFraction() {
        let dashboardFrame = NSRect(x: 120, y: 40, width: 580, height: 800)

        for height in TerminalHeight.allCases {
            let frame = DropDownTerminalPresentation.visibleFrame(
                in: dashboardFrame,
                edge: .bottom,
                height: height
            )
            XCTAssertEqual(frame.height, 800 * height.fraction)
        }
    }

    func testControllerPersistsPreferencesAndRetainsOneWindow() {
        let defaults = terminalTestDefaults()
        let registrar = TestGlobalHotKeyRegistrar()
        let window = TestTerminalWindowController()
        var factoryCount = 0
        let controller = DropDownTerminalController(
            defaults: defaults,
            registrar: registrar,
            makeWindowController: {
                factoryCount += 1
                return window
            }
        )

        controller.edge = .bottom
        controller.height = .spacious
        controller.toggle()
        XCTAssertTrue(controller.isVisible)
        controller.refreshFrame()
        XCTAssertEqual(window.updateCount, 1)
        controller.refreshAppearance()
        XCTAssertEqual(window.appearanceUpdateCount, 1)
        controller.toggle()
        XCTAssertFalse(controller.isVisible)

        XCTAssertEqual(defaults.string(forKey: TerminalEdge.storageKey), TerminalEdge.bottom.rawValue)
        XCTAssertEqual(defaults.string(forKey: TerminalHeight.storageKey), TerminalHeight.spacious.rawValue)
        XCTAssertEqual(factoryCount, 1)
        XCTAssertEqual(window.toggleCount, 2)
        XCTAssertEqual(window.lastEdge, .bottom)
        XCTAssertEqual(window.lastHeight, .spacious)

        let restored = DropDownTerminalController(
            defaults: defaults,
            registrar: TestGlobalHotKeyRegistrar(),
            makeWindowController: { TestTerminalWindowController() }
        )
        XCTAssertEqual(restored.edge, .bottom)
        XCTAssertEqual(restored.height, .spacious)
    }

    func testControllerStartStopAndHotKeyCallbackAreIdempotent() async {
        let registrar = TestGlobalHotKeyRegistrar()
        let window = TestTerminalWindowController()
        let controller = DropDownTerminalController(
            defaults: terminalTestDefaults(),
            registrar: registrar,
            makeWindowController: { window }
        )

        controller.start()
        controller.start()
        XCTAssertEqual(registrar.registerCount, 1)
        XCTAssertEqual(controller.hotKeyStatus, .registered)

        registrar.handler?()
        await Task.yield()
        XCTAssertEqual(window.toggleCount, 1)

        controller.stop()
        controller.stop()
        XCTAssertEqual(registrar.unregisterCount, 1)
        XCTAssertEqual(window.terminateCount, 1)
        XCTAssertEqual(controller.hotKeyStatus, .inactive)
    }

    func testControllerSurfacesRegistrationFailure() {
        let registrar = TestGlobalHotKeyRegistrar()
        registrar.registrationError = TestTerminalError.registration
        let controller = DropDownTerminalController(
            defaults: terminalTestDefaults(),
            registrar: registrar,
            makeWindowController: { TestTerminalWindowController() }
        )

        controller.start()

        guard case .failed(let message) = controller.hotKeyStatus else {
            return XCTFail("Expected failed hotkey status")
        }
        XCTAssertTrue(message.contains("registration"))
    }

    func testShellConfigurationUsesExecutableLoginShellAndTrueColorEnvironment() {
        let configuration = TerminalShellConfiguration.resolve(
            environment: ["SHELL": "/opt/homebrew/bin/fish", "PATH": "/test/bin"],
            homeDirectory: "/Users/test",
            isExecutable: { $0 == "/opt/homebrew/bin/fish" }
        )

        XCTAssertEqual(configuration.executable, "/opt/homebrew/bin/fish")
        XCTAssertEqual(configuration.arguments, ["-l"])
        XCTAssertEqual(configuration.currentDirectory, "/Users/test")
        XCTAssertTrue(configuration.environment.contains("TERM=xterm-256color"))
        XCTAssertTrue(configuration.environment.contains("COLORTERM=truecolor"))
        XCTAssertTrue(configuration.environment.contains("PATH=/test/bin"))
    }

    func testShellConfigurationRejectsRelativeOrUnavailableShell() {
        for shell in ["zsh", "/missing/shell"] {
            let configuration = TerminalShellConfiguration.resolve(
                environment: ["SHELL": shell],
                homeDirectory: "/Users/test",
                isExecutable: { _ in false }
            )
            XCTAssertEqual(configuration.executable, "/bin/zsh")
        }
    }

    func testWarpDetectionUsesStableBundleIdentifier() {
        var requestedIdentifier: String?
        let installed = WarpTerminalIntegration.isInstalled { identifier in
            requestedIdentifier = identifier
            return URL(fileURLWithPath: "/Applications/Warp.app")
        }

        XCTAssertTrue(installed)
        XCTAssertEqual(requestedIdentifier, "dev.warp.Warp-Stable")
        XCTAssertFalse(WarpTerminalIntegration.isInstalled { _ in nil })
    }
}

final class TestGlobalHotKeyRegistrar: GlobalHotKeyRegistering {
    var registrationError: Error?
    private(set) var registerCount = 0
    private(set) var unregisterCount = 0
    private(set) var handler: (() -> Void)?

    func register(handler: @escaping () -> Void) throws {
        registerCount += 1
        if let registrationError { throw registrationError }
        self.handler = handler
    }

    func unregister() {
        unregisterCount += 1
        handler = nil
    }
}

@MainActor
final class TestTerminalWindowController: DropDownTerminalWindowControlling {
    var isVisible = false
    private(set) var toggleCount = 0
    private(set) var updateCount = 0
    private(set) var appearanceUpdateCount = 0
    private(set) var terminateCount = 0
    private(set) var lastEdge: TerminalEdge?
    private(set) var lastHeight: TerminalHeight?

    func toggle(edge: TerminalEdge, height: TerminalHeight) {
        toggleCount += 1
        isVisible.toggle()
        lastEdge = edge
        lastHeight = height
    }

    func update(edge: TerminalEdge, height: TerminalHeight) {
        updateCount += 1
        lastEdge = edge
        lastHeight = height
    }

    func updateAppearance() {
        appearanceUpdateCount += 1
    }

    func terminate() {
        terminateCount += 1
        isVisible = false
    }
}

@MainActor
func makeTestTerminalController(
    registrar: TestGlobalHotKeyRegistrar = TestGlobalHotKeyRegistrar()
) -> DropDownTerminalController {
    DropDownTerminalController(
        defaults: terminalTestDefaults(),
        registrar: registrar,
        makeWindowController: { TestTerminalWindowController() }
    )
}

func terminalTestDefaults() -> UserDefaults {
    let suiteName = "BeaconTerminalTests.\(UUID().uuidString)"
    let defaults = UserDefaults(suiteName: suiteName)!
    defaults.removePersistentDomain(forName: suiteName)
    return defaults
}

private enum TestTerminalError: LocalizedError {
    case registration

    var errorDescription: String? { "registration failed" }
}
