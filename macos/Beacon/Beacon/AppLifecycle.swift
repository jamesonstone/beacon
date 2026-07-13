import AppKit
import SwiftUI

@MainActor
final class BeaconApplicationModel {
    static let shared = BeaconApplicationModel()

    let state: AppState
    let loginItem: LoginItemController
    private var dashboardWindowController: DashboardWindowController?

    convenience init() {
        self.init(state: AppState(), loginItem: LoginItemController())
    }

    init(state: AppState, loginItem: LoginItemController) {
        self.state = state
        self.loginItem = loginItem
    }

    var isDashboardVisible: Bool {
        dashboardWindowController?.window?.isVisible == true
    }

    var dashboardWindow: NSWindow? {
        dashboardWindowController?.window
    }

    func start() {
        state.start()
    }

    func stop() {
        state.stop()
    }

    func handleLaunch(isLoginLaunch: Bool) {
        start()
        if !isLoginLaunch {
            showDashboard()
        }
    }

    func showDashboard(activate: Bool = true) {
        if dashboardWindowController == nil {
            dashboardWindowController = DashboardWindowController(
                state: state,
                loginItem: loginItem
            )
        }
        dashboardWindowController?.show(activate: activate)
    }
}

@MainActor
final class DashboardWindowController: NSWindowController {
    static let preferredWidth: CGFloat = 580
    private var hasPositionedWindow = false

    init(state: AppState, loginItem: LoginItemController) {
        let dashboard = MenuView(
            state: state,
            loginItem: loginItem,
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
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        nil
    }

    func show(activate: Bool) {
        guard let window else { return }
        if !hasPositionedWindow {
            if let visibleFrame = (window.screen ?? NSScreen.main)?.visibleFrame {
                window.setFrame(Self.initialFrame(in: visibleFrame), display: false)
            } else {
                window.center()
            }
            hasPositionedWindow = true
        }
        showWindow(nil)
        window.makeKeyAndOrderFront(nil)
        if activate {
            NSApplication.shared.activate(ignoringOtherApps: true)
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
}

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    private let model: BeaconApplicationModel
    private let isLoginLaunch: Bool
    private var finishedLaunching = false

    override init() {
        model = .shared
        isLoginLaunch = ProcessInfo.processInfo.arguments.contains("--login")
        super.init()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApplication.shared.setActivationPolicy(.regular)
        finishedLaunching = true
        model.handleLaunch(isLoginLaunch: isLoginLaunch)
    }

    func applicationDidBecomeActive(_ notification: Notification) {
        guard finishedLaunching, !model.isDashboardVisible else { return }
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
        model.stop()
    }
}
