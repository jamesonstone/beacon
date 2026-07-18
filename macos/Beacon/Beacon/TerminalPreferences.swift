import AppKit

enum TerminalEdge: String, CaseIterable, Identifiable {
    case top
    case bottom

    static let storageKey = "beacon.terminal.edge"
    static let defaultEdge = TerminalEdge.top

    var id: String { rawValue }
    var title: String { rawValue.capitalized }
    var symbol: String { self == .top ? "rectangle.tophalf.inset.filled" : "rectangle.bottomhalf.inset.filled" }
}

enum TerminalHeight: String, CaseIterable, Identifiable {
    case compact
    case balanced
    case spacious

    static let storageKey = "beacon.terminal.height"
    static let defaultHeight = TerminalHeight.balanced

    var id: String { rawValue }

    var title: String {
        switch self {
        case .compact: "Compact"
        case .balanced: "Balanced"
        case .spacious: "Spacious"
        }
    }

    var fraction: CGFloat {
        switch self {
        case .compact: 0.30
        case .balanced: 0.45
        case .spacious: 0.60
        }
    }
}

enum DropDownTerminalPresentation {
    private static let collapsedHeight: CGFloat = 1

    static func visibleFrame(
        in containerFrame: NSRect,
        edge: TerminalEdge,
        height: TerminalHeight
    ) -> NSRect {
        let panelHeight = containerFrame.height * height.fraction
        let originY = edge == .top
            ? containerFrame.maxY - panelHeight
            : containerFrame.minY
        return NSRect(
            x: containerFrame.minX,
            y: originY,
            width: containerFrame.width,
            height: panelHeight
        )
    }

    static func hiddenFrame(
        in containerFrame: NSRect,
        edge: TerminalEdge
    ) -> NSRect {
        NSRect(
            x: containerFrame.minX,
            y: edge == .top ? containerFrame.maxY - collapsedHeight : containerFrame.minY,
            width: containerFrame.width,
            height: collapsedHeight
        )
    }

    static func clippedContainerFrame(_ preferredFrame: NSRect, to visibleScreenFrame: NSRect) -> NSRect? {
        let clippedFrame = preferredFrame.intersection(visibleScreenFrame)
        guard !clippedFrame.isNull, clippedFrame.width > 0, clippedFrame.height > 0 else {
            return nil
        }
        return clippedFrame
    }
}

struct TerminalShellConfiguration: Equatable {
    let executable: String
    let arguments: [String]
    let environment: [String]
    let currentDirectory: String

    static func resolve(
        environment source: [String: String] = ProcessInfo.processInfo.environment,
        homeDirectory: String = FileManager.default.homeDirectoryForCurrentUser.path,
        isExecutable: (String) -> Bool = { FileManager.default.isExecutableFile(atPath: $0) }
    ) -> TerminalShellConfiguration {
        let requestedShell = source["SHELL"] ?? ""
        let shell = requestedShell.hasPrefix("/") && isExecutable(requestedShell)
            ? requestedShell
            : "/bin/zsh"
        var environment = source
        environment["TERM"] = "xterm-256color"
        environment["COLORTERM"] = "truecolor"
        return TerminalShellConfiguration(
            executable: shell,
            arguments: ["-l"],
            environment: environment.map { "\($0.key)=\($0.value)" }.sorted(),
            currentDirectory: homeDirectory
        )
    }
}

enum WarpTerminalIntegration {
    static let bundleIdentifier = "dev.warp.Warp-Stable"
    static let guideURL = URL(string: "https://docs.warp.dev/terminal/windows/global-hotkey")!

    static var applicationURL: URL? {
        NSWorkspace.shared.urlForApplication(withBundleIdentifier: bundleIdentifier)
    }

    static var isInstalled: Bool {
        isInstalled { NSWorkspace.shared.urlForApplication(withBundleIdentifier: $0) }
    }

    static func isInstalled(using lookup: (String) -> URL?) -> Bool {
        lookup(bundleIdentifier) != nil
    }

    static func openApplication() {
        guard let applicationURL else { return }
        NSWorkspace.shared.open(applicationURL)
    }

    static func openGuide() {
        NSWorkspace.shared.open(guideURL)
    }
}
