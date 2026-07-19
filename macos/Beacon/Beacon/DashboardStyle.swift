import AppKit
import SwiftUI

enum DashboardViewMode: String, CaseIterable, Identifiable {
    case stacked
    case tiles
    case kanban
    case overview
    case fitted

    var id: String { rawValue }

    var title: String {
        switch self {
        case .stacked: "Stacked"
        case .tiles: "Horizontal Tiles"
        case .kanban: "Kanban (Experimental)"
        case .overview: "Overview (Experimental)"
        case .fitted: "Fit Following"
        }
    }

    var symbol: String {
        switch self {
        case .stacked: "rectangle.stack"
        case .tiles: "rectangle.grid.1x2"
        case .kanban: "rectangle.split.3x1"
        case .overview: "rectangle.grid.2x2"
        case .fitted: "rectangle.split.1x2"
        }
    }

    var locksNotesAtHalfHeight: Bool { self == .fitted }
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

struct DashboardViewModeMenu: View, Equatable {
    let mode: DashboardViewMode
    let themeID: String
    let increasedContrast: Bool
    let select: (DashboardViewMode) -> Void

    static func == (lhs: Self, rhs: Self) -> Bool {
        lhs.mode == rhs.mode
            && lhs.themeID == rhs.themeID
            && lhs.increasedContrast == rhs.increasedContrast
    }

    var body: some View {
        let theme = BeaconThemeCatalog.theme(forStoredID: themeID)
        let borderColor = increasedContrast
            ? theme.tokens.borderStrong.color
            : theme.tokens.border.color

        Menu {
            ForEach(DashboardViewMode.allCases) { option in
                Button {
                    select(option)
                } label: {
                    Label(
                        option.title,
                        systemImage: option == mode ? "checkmark" : option.symbol
                    )
                }
                .disabled(option == mode)
            }
        } label: {
            Image(systemName: mode.symbol)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(theme.tokens.info.color)
                .frame(width: 28, height: 28)
                .background(theme.tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(borderColor, lineWidth: increasedContrast ? 1.1 : 0.7)
                }
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .fixedSize()
        .help("View mode: \(mode.title)")
        .accessibilityLabel("View mode, \(mode.title)")
    }
}
