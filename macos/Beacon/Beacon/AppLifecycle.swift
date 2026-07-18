import AppKit
import SwiftUI

@MainActor
final class BeaconApplicationModel {
    static let shared = BeaconApplicationModel()

    let state: AppState
    let loginItem: LoginItemController
    let terminal: DropDownTerminalController
    private var dashboardWindowController: DashboardWindowController?

    convenience init() {
        self.init(
            state: AppState(externalActivityClient: CLIClient()),
            loginItem: LoginItemController(),
            terminal: DropDownTerminalController()
        )
    }

    init(
        state: AppState,
        loginItem: LoginItemController,
        terminal: DropDownTerminalController
    ) {
        self.state = state
        self.loginItem = loginItem
        self.terminal = terminal
        terminal.setContainerFrameProvider { [weak self] in
            self?.dashboardFrameForTerminal()
        }
    }

    var isDashboardVisible: Bool {
        dashboardWindowController?.window?.isVisible == true
    }

    var dashboardWindow: NSWindow? {
        dashboardWindowController?.window
    }

    var isTerminalVisible: Bool {
        terminal.isVisible
    }

    func start() {
        state.start()
        terminal.start()
    }

    func stop() {
        terminal.stop()
        state.stop()
    }

    @discardableResult
    func terminate() -> Error? {
        stop()
        return state.stopAgentSynchronously()
    }

    func handleLaunch(isLoginLaunch: Bool) {
        start()
        if !isLoginLaunch {
            showDashboard()
        }
    }

    func showDashboard(activate: Bool = true) {
        dashboardController().show(activate: activate)
    }

    private func dashboardFrameForTerminal() -> NSRect? {
        let controller = dashboardController()
        controller.positionWindowIfNeeded()
        return controller.window?.frame
    }

    private func dashboardController() -> DashboardWindowController {
        if let dashboardWindowController {
            return dashboardWindowController
        }
        let controller = DashboardWindowController(
            state: state,
            loginItem: loginItem,
            terminal: terminal
        )
        dashboardWindowController = controller
        return controller
    }
}

@MainActor
final class DashboardWindowController: NSWindowController, NSWindowDelegate {
    static let preferredWidth: CGFloat = 580
    static let defaultFrameAutosaveName: NSWindow.FrameAutosaveName = "BeaconDashboardWindow"
    private let frameAutosaveName: NSWindow.FrameAutosaveName
    private let terminal: DropDownTerminalController
    private var hasPositionedWindow = false

    init(
        state: AppState,
        loginItem: LoginItemController,
        terminal: DropDownTerminalController,
        frameAutosaveName: NSWindow.FrameAutosaveName? = nil
    ) {
        self.frameAutosaveName = frameAutosaveName ?? Self.defaultFrameAutosaveName
        self.terminal = terminal
        let dashboard = MenuView(
            state: state,
            loginItem: loginItem,
            terminal: terminal,
            surface: .window,
            openDashboard: {}
        )
        let hostingController = NSHostingController(rootView: dashboard)
        let window = NSWindow(contentViewController: hostingController)
        window.title = "Beacon"
        window.setContentSize(NSSize(width: Self.preferredWidth, height: 620))
        window.minSize = NSSize(width: 430, height: 540)
        window.styleMask = [.titled, .closable, .miniaturizable, .resizable]
        window.titlebarAppearsTransparent = true
        window.isReleasedWhenClosed = false
        window.collectionBehavior = [.participatesInCycle]
        super.init(window: window)
        window.delegate = self
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        nil
    }

    func show(activate: Bool) {
        positionWindowIfNeeded()
        showWindow(nil)
        window?.makeKeyAndOrderFront(nil)
        if activate {
            NSApplication.shared.activate(ignoringOtherApps: true)
        }
    }

    func positionWindowIfNeeded() {
        guard let window else { return }
        if !hasPositionedWindow {
            if !window.setFrameUsingName(frameAutosaveName) {
                if let visibleFrame = (window.screen ?? NSScreen.main)?.visibleFrame {
                    window.setFrame(Self.initialFrame(in: visibleFrame), display: false)
                } else {
                    window.center()
                }
            }
            _ = window.setFrameAutosaveName(frameAutosaveName)
            hasPositionedWindow = true
        }
    }

    static func initialFrame(in visibleFrame: NSRect) -> NSRect {
        let width = min(preferredWidth, visibleFrame.width)
        return NSRect(
            x: visibleFrame.midX - (width / 2),
            y: visibleFrame.minY,
            width: width,
            height: visibleFrame.height
        )
    }

    func windowDidMove(_ notification: Notification) {
        terminal.refreshFrame()
    }

    func windowDidResize(_ notification: Notification) {
        terminal.refreshFrame()
    }
}

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    static var isRunningUnitTests: Bool {
        NSClassFromString("XCTestCase") != nil
            || ProcessInfo.processInfo.environment["XCTestConfigurationFilePath"] != nil
    }

    private let model: BeaconApplicationModel
    private let isLoginLaunch: Bool
    private var finishedLaunching = false

    override init() {
        model = .shared
        isLoginLaunch = ProcessInfo.processInfo.arguments.contains("--login")
        super.init()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        guard !Self.isRunningUnitTests else { return }
        NSApplication.shared.setActivationPolicy(.regular)
        finishedLaunching = true
        model.handleLaunch(isLoginLaunch: isLoginLaunch)
    }

    func applicationDidBecomeActive(_ notification: Notification) {
        guard finishedLaunching, !model.isDashboardVisible, !model.isTerminalVisible else { return }
        model.showDashboard()
    }

    func applicationShouldHandleReopen(
        _ sender: NSApplication,
        hasVisibleWindows flag: Bool
    ) -> Bool {
        model.showDashboard()
        return true
    }

    func applicationWillTerminate(_ notification: Notification) {
        guard !Self.isRunningUnitTests else { return }
        if let error = model.terminate() {
            NSLog("Beacon could not stop its background agent: %@", error.localizedDescription)
        }
    }
}
