import AppKit
import SwiftUI

enum DashboardViewMode: String, CaseIterable, Identifiable {
    case stacked
    case tiles
    case kanban
    case overview

    var id: String { rawValue }

    var title: String {
        switch self {
        case .stacked: "Stacked"
        case .tiles: "Horizontal Tiles"
        case .kanban: "Kanban (Experimental)"
        case .overview: "Overview (Experimental)"
        }
    }

    var symbol: String {
        switch self {
        case .stacked: "rectangle.stack"
        case .tiles: "rectangle.grid.1x2"
        case .kanban: "rectangle.split.3x1"
        case .overview: "rectangle.grid.2x2"
        }
    }
}

enum DashboardDensity: String, CaseIterable, Identifiable {
    case comfortable
    case compact
    case dense

    static let storageKey = "beacon.dashboard.density"
    static let defaultDensity = DashboardDensity.comfortable

    var id: String { rawValue }

    var title: String {
        switch self {
        case .comfortable: "Comfortable"
        case .compact: "Compact"
        case .dense: "Dense"
        }
    }

    var symbol: String {
        switch self {
        case .comfortable: "rectangle.grid.1x2"
        case .compact: "rectangle.grid.2x2"
        case .dense: "rectangle.grid.3x2"
        }
    }

    var cardPadding: CGFloat {
        switch self {
        case .comfortable: 10
        case .compact: 8
        case .dense: 6
        }
    }

    var spacing: CGFloat {
        switch self {
        case .comfortable: 5
        case .compact: 4
        case .dense: 2
        }
    }

    var titleSize: CGFloat {
        switch self {
        case .comfortable: DashboardLanePresentation.laneTitleSize
        case .compact: 11
        case .dense: 10
        }
    }

    var titleLines: Int { self == .comfortable ? 1 : 2 }

    var tileWidth: CGFloat {
        switch self {
        case .comfortable: 248
        case .compact: 220
        case .dense: 184
        }
    }
}

enum BeaconFontSize: Int, CaseIterable, Identifiable {
    case compact = 11
    case standard = 12
    case comfortable = 13
    case large = 14
    case extraLarge = 16

    var id: Int { rawValue }
    var title: String { "\(rawValue) pt" }
}

enum BeaconTypography {
    static let baseSizeKey = "beacon.dashboard.font-size"
    static let defaultBaseSize = BeaconFontSize.standard.rawValue

    static func regular(_ size: CGFloat) -> Font {
        preferred(size: size, weight: .regular)
    }

    static func medium(_ size: CGFloat) -> Font {
        preferred(size: size, weight: .medium)
    }

    static func semibold(_ size: CGFloat) -> Font {
        preferred(size: size, weight: .semibold)
    }

    static func bold(_ size: CGFloat) -> Font {
        preferred(size: size, weight: .bold)
    }

    static func identifier(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        .system(size: resolvedSize(size), weight: weight, design: .monospaced)
    }

    static func counter(_ size: CGFloat, weight: Font.Weight = .medium) -> Font {
        .system(size: resolvedSize(size), weight: weight, design: .monospaced)
    }

    static func appKitFont(_ size: CGFloat, weight: NSFont.Weight = .regular) -> NSFont {
        NSFont.systemFont(ofSize: resolvedSize(size), weight: weight)
    }

    static func appKitCodeFont(_ size: CGFloat, weight: NSFont.Weight = .regular) -> NSFont {
        NSFont.monospacedSystemFont(ofSize: resolvedSize(size), weight: weight)
    }

    static var selectionSignature: String {
        "system:\(selectedBaseSize):\(BeaconThemePreference.current().id.rawValue)"
    }

    static func resolvedSize(_ size: CGFloat) -> CGFloat {
        resolvedSize(size, baseSize: selectedBaseSize)
    }

    static func resolvedSize(_ size: CGFloat, baseSize: Int) -> CGFloat {
        max(11, size + CGFloat(baseSize - 10))
    }

    private static var selectedBaseSize: Int {
        let value = UserDefaults.standard.integer(forKey: baseSizeKey)
        return BeaconFontSize(rawValue: value)?.rawValue ?? defaultBaseSize
    }

    private static func preferred(size: CGFloat, weight: Font.Weight) -> Font {
        .system(size: resolvedSize(size), weight: weight, design: .default)
    }
}

enum DashboardLaneAccent: String, CaseIterable {
    case local
    case pullRequest
    case issue

    var color: Color {
        switch self {
        case .local: BeaconThemePreference.current().tokens.identityLocal.color
        case .pullRequest: BeaconThemePreference.current().tokens.identityPullRequest.color
        case .issue: BeaconThemePreference.current().tokens.identityIssue.color
        }
    }
}

enum DashboardLaneIdentity: String, CaseIterable {
    case local
    case pullRequest
    case issue

    var accent: DashboardLaneAccent {
        switch self {
        case .local: .local
        case .pullRequest: .pullRequest
        case .issue: .issue
        }
    }

    var title: String {
        switch self {
        case .local: "Local"
        case .pullRequest: "Pull Request"
        case .issue: "Issue"
        }
    }

    var symbol: String {
        switch self {
        case .local: "laptopcomputer"
        case .pullRequest: "arrow.triangle.pull"
        case .issue: "smallcircle.filled.circle"
        }
    }
}

enum DashboardLanePresentation {
    static let projectNameSize: CGFloat = 15
    static let laneTitleSize: CGFloat = 13

    static func showsIgnoreAction(in tab: DashboardTab) -> Bool {
        tab == .following
    }

    static func showsCheckoutWarning(for lane: WorkLane) -> Bool {
        lane.checkoutWarning?.kind == "merged_remote_branch_deleted"
    }

    static func checkoutWarningIsCritical(for lane: WorkLane) -> Bool {
        lane.checkoutWarning?.severity == "critical"
    }

    static func identity(for lane: WorkLane) -> DashboardLaneIdentity {
        if lane.pullRequest != nil {
            return .pullRequest
        }
        if lane.issue != nil {
            return .issue
        }
        return .local
    }
}

enum NeonWave {
    static let cycle: TimeInterval = 6
    static var gradient: LinearGradient { BeaconThemePreference.current().brandGradient }

    static func phase(at date: Date) -> Double {
        let elapsed = date.timeIntervalSinceReferenceDate.truncatingRemainder(dividingBy: cycle)
        return (elapsed < 0 ? elapsed + cycle : elapsed) / cycle
    }

    static func rotation(at date: Date) -> Angle {
        .degrees(phase(at: date) * 360)
    }
}

struct NeonWaveWordmark: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    private let text: String

    init(_ text: String) {
        self.text = text
    }

    var body: some View {
        TimelineView(.animation(minimumInterval: 1.0 / 20.0, paused: reduceMotion)) { context in
            Text(text)
                .foregroundStyle(NeonWave.gradient)
                .hueRotation(reduceMotion ? .zero : NeonWave.rotation(at: context.date))
                .shadow(color: BeaconThemePreference.current().tokens.identityIssue.color.opacity(0.28), radius: 2)
                .accessibilityLabel(text)
        }
    }
}
