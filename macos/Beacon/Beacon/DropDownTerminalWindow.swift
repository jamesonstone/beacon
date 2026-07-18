import AppKit
import SwiftTerm

@MainActor
final class DropDownTerminalWindowController: NSWindowController, DropDownTerminalWindowControlling {
    private let terminalView: LocalProcessTerminalView
    private let containerFrameProvider: () -> NSRect?
    private var previousApplication: NSRunningApplication?
    private var transitionID = 0

    init(containerFrameProvider: @escaping () -> NSRect? = { nil }) {
        self.containerFrameProvider = containerFrameProvider
        terminalView = LocalProcessTerminalView(frame: .zero)
        let panel = NSPanel(
            contentRect: .zero,
            styleMask: [.titled, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        panel.title = "Beacon Terminal"
        panel.titleVisibility = .hidden
        panel.titlebarAppearsTransparent = true
        panel.titlebarSeparatorStyle = .none
        panel.standardWindowButton(.closeButton)?.isHidden = true
        panel.standardWindowButton(.miniaturizeButton)?.isHidden = true
        panel.standardWindowButton(.zoomButton)?.isHidden = true
        panel.isFloatingPanel = true
        panel.isReleasedWhenClosed = false
        panel.hidesOnDeactivate = false
        panel.level = .floating
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .transient]
        panel.animationBehavior = .none
        panel.contentView = terminalView
        super.init(window: panel)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        nil
    }

    var isVisible: Bool {
        window?.isVisible == true
    }

    func toggle(edge: TerminalEdge, height: TerminalHeight) {
        if isVisible {
            hide(edge: edge, height: height)
        } else {
            show(edge: edge, height: height)
        }
    }

    func update(edge: TerminalEdge, height: TerminalHeight) {
        guard let window, window.isVisible, let containerFrame = resolvedContainerFrame() else { return }
        transitionID += 1
        window.setFrame(
            DropDownTerminalPresentation.visibleFrame(in: containerFrame, edge: edge, height: height),
            display: true,
            animate: false
        )
    }

    func terminate() {
        transitionID += 1
        terminalView.terminate()
        window?.orderOut(nil)
        previousApplication = nil
    }

    private func show(edge: TerminalEdge, height: TerminalHeight) {
        guard let window, let containerFrame = resolvedContainerFrame() else { return }
        transitionID += 1
        let currentTransition = transitionID
        let currentApplication = NSRunningApplication.current
        let frontmostApplication = NSWorkspace.shared.frontmostApplication
        previousApplication = frontmostApplication?.processIdentifier == currentApplication.processIdentifier
            ? nil
            : frontmostApplication
        configureAppearance()
        startShellIfNeeded()

        let hiddenFrame = DropDownTerminalPresentation.hiddenFrame(
            in: containerFrame,
            edge: edge
        )
        let visiblePanelFrame = DropDownTerminalPresentation.visibleFrame(
            in: containerFrame,
            edge: edge,
            height: height
        )
        window.setFrame(hiddenFrame, display: false)
        window.makeKeyAndOrderFront(nil)
        NSApplication.shared.activate(ignoringOtherApps: true)
        window.makeFirstResponder(terminalView)

        animate(window: window, to: visiblePanelFrame) { [weak self] in
            guard self?.transitionID == currentTransition else { return }
            window.makeFirstResponder(self?.terminalView)
        }
    }

    private func hide(edge: TerminalEdge, height: TerminalHeight) {
        guard let window, let containerFrame = resolvedContainerFrame() else {
            window?.orderOut(nil)
            return
        }
        transitionID += 1
        let currentTransition = transitionID
        let hiddenFrame = DropDownTerminalPresentation.hiddenFrame(
            in: containerFrame,
            edge: edge
        )
        animate(window: window, to: hiddenFrame) { [weak self] in
            guard self?.transitionID == currentTransition else { return }
            window.orderOut(nil)
            self?.previousApplication?.activate(options: [])
            self?.previousApplication = nil
        }
    }

    private func animate(window: NSWindow, to frame: NSRect, completion: @escaping () -> Void) {
        let duration = NSWorkspace.shared.accessibilityDisplayShouldReduceMotion ? 0 : 0.18
        guard duration > 0 else {
            window.setFrame(frame, display: true)
            completion()
            return
        }
        NSAnimationContext.runAnimationGroup { context in
            context.duration = duration
            context.timingFunction = CAMediaTimingFunction(name: .easeInEaseOut)
            window.animator().setFrame(frame, display: true)
        } completionHandler: {
            completion()
        }
    }

    private func configureAppearance() {
        let theme = BeaconThemePreference.current()
        terminalView.font = BeaconTypography.appKitCodeFont(10)
        terminalView.nativeBackgroundColor = theme.tokens.canvas.nsColor
        terminalView.nativeForegroundColor = theme.tokens.textPrimary.nsColor
        terminalView.needsDisplay = true
    }

    private func startShellIfNeeded() {
        guard !terminalView.process.running else { return }
        let configuration = TerminalShellConfiguration.resolve()
        terminalView.startProcess(
            executable: configuration.executable,
            args: configuration.arguments,
            environment: configuration.environment,
            currentDirectory: configuration.currentDirectory
        )
    }

    private func resolvedContainerFrame() -> NSRect? {
        if let preferredFrame = containerFrameProvider(),
           preferredFrame.width > 0,
           preferredFrame.height > 0,
           let screen = Self.screen(containing: preferredFrame) {
            if let clippedFrame = DropDownTerminalPresentation.clippedContainerFrame(
                preferredFrame,
                to: screen.visibleFrame
            ) {
                return clippedFrame
            }
        }
        guard let visibleFrame = Self.activeScreen()?.visibleFrame else { return nil }
        return DashboardWindowController.initialFrame(in: visibleFrame)
    }

    private static func screen(containing frame: NSRect) -> NSScreen? {
        NSScreen.screens.max { left, right in
            intersectionArea(of: frame, with: left.frame)
                < intersectionArea(of: frame, with: right.frame)
        }
    }

    private static func intersectionArea(of left: NSRect, with right: NSRect) -> CGFloat {
        let intersection = left.intersection(right)
        guard !intersection.isNull else { return 0 }
        return intersection.width * intersection.height
    }

    private static func activeScreen() -> NSScreen? {
        let pointer = NSEvent.mouseLocation
        return NSScreen.screens.first { NSMouseInRect(pointer, $0.frame, false) }
            ?? NSScreen.main
    }
}
