import SwiftUI

@main
struct BeaconApp: App {
    @StateObject private var state = AppState()

    var body: some Scene {
        MenuBarExtra {
            MenuView(state: state)
        } label: {
            BeaconMenuBarLabel(inProgressCount: state.inProgressCount)
                .task { state.start() }
        }
        .menuBarExtraStyle(.window)
    }
}

struct BeaconMenuBarLabel: View {
    let inProgressCount: Int

    var body: some View {
        Group {
            if inProgressCount > 0 {
                Text(displayCount)
                    .font(.system(size: 12, weight: .heavy, design: .rounded))
                    .monospacedDigit()
                    .foregroundStyle(.white)
                    .padding(.horizontal, displayCount.count > 2 ? 5 : 7)
                    .frame(minWidth: 22, minHeight: 18)
                    .background(
                        Capsule()
                            .fill(BeaconPalette.menuBarBadgeFill)
                    )
                    .overlay {
                        Capsule()
                            .strokeBorder(BeaconPalette.neonGradient, lineWidth: 1.5)
                    }
                    .overlay {
                        Capsule()
                            .strokeBorder(.white.opacity(0.42), lineWidth: 0.5)
                            .padding(1.5)
                    }
                    .shadow(color: BeaconPalette.cyan.opacity(0.9), radius: 2)
                    .shadow(color: BeaconPalette.pink.opacity(0.65), radius: 3)
                    .fixedSize()
            } else {
                ZStack {
                    Circle()
                        .fill(
                            RadialGradient(
                                colors: [
                                    Color(red: 0.20, green: 0.08, blue: 0.42),
                                    Color(red: 0.025, green: 0.02, blue: 0.10),
                                ],
                                center: .topLeading,
                                startRadius: 1,
                                endRadius: 13
                            )
                        )

                    Circle()
                        .strokeBorder(BeaconPalette.neonGradient, lineWidth: 1.2)

                    Ellipse()
                        .stroke(BeaconPalette.neonGradient, lineWidth: 1.4)
                        .frame(width: 20, height: 8)
                        .rotationEffect(.degrees(-24))

                    Image(systemName: "sparkle")
                        .font(.system(size: 7, weight: .bold))
                        .foregroundStyle(.white)
                        .offset(x: 2, y: -2)
                }
                .frame(width: 18, height: 18)
                .shadow(color: BeaconPalette.cyan.opacity(0.9), radius: 2)
                .shadow(color: BeaconPalette.pink.opacity(0.55), radius: 3)
            }
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(accessibilityText)
    }

    private var displayCount: String {
        inProgressCount > 99 ? "99+" : String(inProgressCount)
    }

    private var accessibilityText: String {
        if inProgressCount == 0 {
            return "Beacon, no items in progress"
        }
        return "Beacon, \(inProgressCount) items in progress"
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

    static let menuBarBadgeFill = LinearGradient(
        colors: [
            Color(red: 0.015, green: 0.025, blue: 0.09).opacity(0.96),
            Color(red: 0.15, green: 0.06, blue: 0.28).opacity(0.94),
        ],
        startPoint: .top,
        endPoint: .bottom
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
