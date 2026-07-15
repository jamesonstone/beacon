import AppKit
import SwiftUI

@main
struct BeaconApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @StateObject private var state = BeaconApplicationModel.shared.state
    @StateObject private var loginItem = BeaconApplicationModel.shared.loginItem

    var body: some Scene {
        MenuBarExtra {
            MenuView(
                state: state,
                loginItem: loginItem,
                surface: .menu,
                openDashboard: { BeaconApplicationModel.shared.showDashboard() }
            )
        } label: {
            BeaconMenuBarLabel(inProgressCount: state.inProgressCount)
        }
        .menuBarExtraStyle(.window)
    }
}

struct BeaconMenuBarLabel: View {
    let inProgressCount: Int

    var body: some View {
        BeaconMenuBarDome(count: inProgressCount)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(BeaconMenuBarPresentation.accessibilityText(inProgressCount))
    }
}

struct BeaconMenuBarDome: View {
    let count: Int

    private var domeWidth: CGFloat {
        BeaconMenuBarPresentation.domeWidth(count)
    }

    var body: some View {
        ZStack {
            Capsule()
                .fill(BeaconPalette.gold)
                .frame(width: 1.5, height: 4)
                .offset(y: -7)

            Capsule()
                .fill(BeaconPalette.cyan)
                .frame(width: 1.5, height: 4)
                .rotationEffect(.degrees(-48))
                .offset(x: -(domeWidth * 0.38), y: -5)

            Capsule()
                .fill(BeaconPalette.cyan)
                .frame(width: 1.5, height: 4)
                .rotationEffect(.degrees(48))
                .offset(x: domeWidth * 0.38, y: -5)

            BeaconDomeShape()
                .fill(
                    LinearGradient(
                        colors: [BeaconPalette.gold, BeaconPalette.coral],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                )
                .frame(width: domeWidth, height: 11)
                .offset(y: 3)

            Capsule()
                .fill(BeaconPalette.gold)
                .frame(width: domeWidth + 2, height: 1.5)
                .offset(y: 7)

            Text(BeaconMenuBarPresentation.displayCount(count))
                .font(.system(
                    size: BeaconMenuBarPresentation.countFontSize(count),
                    weight: .heavy,
                    design: .rounded
                ))
                .monospacedDigit()
                .foregroundStyle(Color(red: 0.04, green: 0.03, blue: 0.12))
                .lineLimit(1)
                .offset(y: 3)
        }
        .frame(width: domeWidth + 6, height: 18)
        .shadow(color: BeaconPalette.cyan.opacity(0.80), radius: 1.5)
        .shadow(color: BeaconPalette.gold.opacity(0.70), radius: 2.5)
    }
}

struct BeaconDomeShape: Shape {
    func path(in rect: CGRect) -> Path {
        let shoulderY = rect.minY + rect.height * 0.58
        let bottomY = rect.maxY - 0.5
        var path = Path()
        path.move(to: CGPoint(x: rect.minX + 1, y: bottomY))
        path.addLine(to: CGPoint(x: rect.minX + 2, y: shoulderY))
        path.addCurve(
            to: CGPoint(x: rect.midX, y: rect.minY),
            control1: CGPoint(x: rect.minX + 2, y: rect.minY + rect.height * 0.16),
            control2: CGPoint(x: rect.midX - rect.width * 0.22, y: rect.minY)
        )
        path.addCurve(
            to: CGPoint(x: rect.maxX - 2, y: shoulderY),
            control1: CGPoint(x: rect.midX + rect.width * 0.22, y: rect.minY),
            control2: CGPoint(x: rect.maxX - 2, y: rect.minY + rect.height * 0.16)
        )
        path.addLine(to: CGPoint(x: rect.maxX - 1, y: bottomY))
        path.closeSubpath()
        return path
    }
}

enum BeaconMenuBarPresentation {
    static func displayCount(_ count: Int) -> String {
        count > 99 ? "99+" : String(max(0, count))
    }

    static func domeWidth(_ count: Int) -> CGFloat {
        switch displayCount(count).count {
        case 1: 14
        case 2: 18
        default: 24
        }
    }

    static func countFontSize(_ count: Int) -> CGFloat {
        switch displayCount(count).count {
        case 1: 9
        case 2: 8
        default: 6.5
        }
    }

    static func accessibilityText(_ count: Int) -> String {
        if count <= 0 {
            return "Beacon, no items in progress"
        }
        return "Beacon, \(count) items in progress"
    }
}

enum BeaconPalette {
    static let cyan = Color(red: 0.20, green: 0.91, blue: 1.0)
    static let mint = Color(red: 0.42, green: 1.0, blue: 0.76)
    static let lavender = Color(red: 0.70, green: 0.58, blue: 1.0)
    static let pink = Color(red: 1.0, green: 0.36, blue: 0.76)
    static let coral = Color(red: 1.0, green: 0.47, blue: 0.56)
    static let gold = Color(red: 1.0, green: 0.82, blue: 0.46)

    static let neonGradient = LinearGradient(
        colors: [cyan, lavender, pink],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static let panelBackground = LinearGradient(
        colors: [
            cyan.opacity(0.055),
            lavender.opacity(0.045),
            pink.opacity(0.035),
        ],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static let switcherBackground = LinearGradient(
        colors: [
            Color(red: 0.035, green: 0.028, blue: 0.055),
            Color(red: 0.075, green: 0.045, blue: 0.090),
        ],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static func softGradient(_ accent: Color) -> LinearGradient {
        LinearGradient(
            colors: [accent.opacity(0.18), lavender.opacity(0.09), cyan.opacity(0.035)],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
    }

    static func borderGradient(_ accent: Color) -> LinearGradient {
        LinearGradient(
            colors: [accent.opacity(0.8), lavender.opacity(0.38), cyan.opacity(0.18)],
            startPoint: .leading,
            endPoint: .trailing
        )
    }
}
