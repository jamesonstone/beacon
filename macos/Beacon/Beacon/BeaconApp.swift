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
        HStack(spacing: 2) {
            ZStack {
                Circle()
                    .fill(
                        RadialGradient(
                            colors: [
                                BeaconPalette.pink.opacity(0.82),
                                Color(red: 0.05, green: 0.02, blue: 0.16),
                            ],
                            center: .center,
                            startRadius: 1,
                            endRadius: 10
                        )
                    )
                    .frame(width: 14, height: 14)

                Image(systemName: "light.beacon.max.fill")
                    .symbolRenderingMode(.palette)
                    .foregroundStyle(BeaconPalette.gold, BeaconPalette.cyan)
                    .font(.system(size: 16, weight: .bold))

                Circle()
                    .fill(.white)
                    .frame(width: 3, height: 3)
                    .shadow(color: BeaconPalette.gold, radius: 2)
            }
            .frame(width: 22, height: 18)
            .shadow(color: BeaconPalette.cyan.opacity(0.95), radius: 2)
            .shadow(color: BeaconPalette.pink.opacity(0.65), radius: 3)

            if inProgressCount > 0 {
                Text(BeaconMenuBarPresentation.displayCount(inProgressCount))
                    .font(.system(size: 9, weight: .heavy, design: .rounded))
                    .monospacedDigit()
                    .foregroundStyle(Color(red: 0.04, green: 0.03, blue: 0.12))
                    .padding(.horizontal, 4)
                    .frame(minWidth: 15, minHeight: 15)
                    .background(
                        LinearGradient(
                            colors: [BeaconPalette.gold, BeaconPalette.coral],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        ),
                        in: Capsule()
                    )
                    .overlay {
                        Capsule()
                            .strokeBorder(.white.opacity(0.72), lineWidth: 0.7)
                    }
                    .shadow(color: BeaconPalette.gold.opacity(0.85), radius: 2)
                    .fixedSize()
            }
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(BeaconMenuBarPresentation.accessibilityText(inProgressCount))
    }
}

enum BeaconMenuBarPresentation {
    static func displayCount(_ count: Int) -> String {
        count > 99 ? "99+" : String(max(0, count))
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
