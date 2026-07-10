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
                    .font(.system(size: 13, weight: .bold, design: .rounded))
                    .monospacedDigit()
                    .foregroundStyle(neonGradient)
                    .frame(minWidth: 16)
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

                    Ellipse()
                        .stroke(neonGradient, lineWidth: 1.4)
                        .frame(width: 20, height: 8)
                        .rotationEffect(.degrees(-24))

                    Image(systemName: "sparkle")
                        .font(.system(size: 7, weight: .bold))
                        .foregroundStyle(.white)
                        .offset(x: 2, y: -2)
                }
                .frame(width: 18, height: 18)
                .shadow(color: .cyan.opacity(0.75), radius: 2)
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

    private var neonGradient: LinearGradient {
        LinearGradient(
            colors: [
                Color(red: 0.10, green: 0.95, blue: 1.0),
                Color(red: 0.52, green: 0.28, blue: 1.0),
                Color(red: 1.0, green: 0.18, blue: 0.76),
            ],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
    }
}
