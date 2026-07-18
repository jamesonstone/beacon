import XCTest
@testable import Beacon

@MainActor
final class TerminalFeatureTests: XCTestCase {
    func testFramesCoverBothEdgesAndOffsetScreens() {
        let screen = NSRect(x: 1_920, y: 24, width: 1_600, height: 1_000)

        let top = DropDownTerminalPresentation.visibleFrame(
            in: screen,
            edge: .top,
            height: .balanced
        )
        XCTAssertEqual(top, NSRect(x: 1_920, y: 574, width: 1_600, height: 450))
        XCTAssertEqual(
            DropDownTerminalPresentation.hiddenFrame(in: screen, edge: .top, height: .balanced),
            NSRect(x: 1_920, y: 1_024, width: 1_600, height: 450)
        )

        let bottom = DropDownTerminalPresentation.visibleFrame(
            in: screen,
            edge: .bottom,
            height: .compact
        )
        XCTAssertEqual(bottom, NSRect(x: 1_920, y: 24, width: 1_600, height: 300))
        XCTAssertEqual(
            DropDownTerminalPresentation.hiddenFrame(in: screen, edge: .bottom, height: .compact),
            NSRect(x: 1_920, y: -276, width: 1_600, height: 300)
        )
    }

    func testEveryHeightUsesItsDeclaredFraction() {
        let screen = NSRect(x: 0, y: 0, width: 1_200, height: 800)

        for height in TerminalHeight.allCases {
            let frame = DropDownTerminalPresentation.visibleFrame(
                in: screen,
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
