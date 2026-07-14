import SwiftUI

enum UpToDatePresentation {
    static let title = "All caught up"
    static let detail = "No work is in progress. Enjoy the quiet—or catch the next spark."

    static func shouldShow(inProgressCount: Int, loadingProjectCount: Int) -> Bool {
        inProgressCount == 0 && loadingProjectCount == 0
    }
}

struct UpToDateBacksplash: View {
    let surface: DashboardSurface
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        Group {
            switch surface {
            case .menu:
                compactState
            case .window:
                ViewThatFits(in: .vertical) {
                    fullState
                    compactState
                }
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background {
            RoundedRectangle(cornerRadius: 14)
                .fill(
                    RadialGradient(
                        colors: [
                            BeaconPalette.cyan.opacity(0.10),
                            BeaconPalette.lavender.opacity(0.055),
                            BeaconPalette.pink.opacity(0.025),
                            Color.clear,
                        ],
                        center: .center,
                        startRadius: 8,
                        endRadius: 300
                    )
                )
        }
        .overlay {
            RoundedRectangle(cornerRadius: 14)
                .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.cyan), lineWidth: 0.6)
                .opacity(0.42)
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("\(UpToDatePresentation.title). No work is in progress.")
    }

    private var fullState: some View {
        VStack(spacing: 14) {
            orbit(size: 118)
            VStack(spacing: 5) {
                Text(UpToDatePresentation.title)
                    .font(BeaconTypography.bold(22))
                    .foregroundStyle(BeaconPalette.neonGradient)
                    .shadow(color: BeaconPalette.pink.opacity(0.24), radius: 2)
                Text(UpToDatePresentation.detail)
                    .font(BeaconTypography.regular(11))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.86))
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 430)
            }
            caughtUpBadge
        }
        .padding(28)
        .frame(minHeight: 280)
    }

    private var compactState: some View {
        HStack(spacing: 14) {
            orbit(size: 68)
            VStack(alignment: .leading, spacing: 4) {
                Text(UpToDatePresentation.title)
                    .font(BeaconTypography.bold(15))
                    .foregroundStyle(BeaconPalette.neonGradient)
                    .shadow(color: BeaconPalette.pink.opacity(0.20), radius: 2)
                Text("No work in progress—clear skies ahead.")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.82))
                    .lineLimit(2)
                caughtUpBadge
            }
        }
        .padding(16)
        .frame(minHeight: 112)
    }

    private var caughtUpBadge: some View {
        Label("Lane radar clear", systemImage: "sparkles")
            .font(BeaconTypography.semibold(9))
            .foregroundStyle(BeaconPalette.mint)
            .padding(.horizontal, 9)
            .padding(.vertical, 4)
            .background(BeaconPalette.softGradient(BeaconPalette.mint), in: Capsule())
            .overlay {
                Capsule()
                    .strokeBorder(BeaconPalette.mint.opacity(0.40), lineWidth: 0.7)
            }
    }

    private func orbit(size: CGFloat) -> some View {
        TimelineView(.animation(minimumInterval: 1.0 / 12.0, paused: reduceMotion)) { context in
            let angle = reduceMotion
                ? Angle.degrees(18)
                : Angle.degrees(context.date.timeIntervalSinceReferenceDate * 24)

            ZStack {
                Circle()
                    .fill(
                        RadialGradient(
                            colors: [
                                BeaconPalette.mint.opacity(0.28),
                                BeaconPalette.cyan.opacity(0.12),
                                BeaconPalette.lavender.opacity(0.04),
                            ],
                            center: .topLeading,
                            startRadius: 2,
                            endRadius: size * 0.6
                        )
                    )
                    .frame(width: size * 0.70, height: size * 0.70)
                    .shadow(color: BeaconPalette.cyan.opacity(0.28), radius: 14)

                Circle()
                    .stroke(BeaconPalette.neonGradient, lineWidth: 1.4)
                    .frame(width: size * 0.66, height: size * 0.66)

                Image(systemName: "checkmark.seal.fill")
                    .font(.system(size: size * 0.32, weight: .semibold))
                    .foregroundStyle(BeaconPalette.mint, BeaconPalette.cyan.opacity(0.42))
                    .shadow(color: BeaconPalette.mint.opacity(0.50), radius: 4)

                ZStack {
                    Ellipse()
                        .stroke(BeaconPalette.borderGradient(BeaconPalette.lavender), lineWidth: 1.2)
                        .frame(width: size, height: size * 0.40)
                    Image(systemName: "sparkle")
                        .font(.system(size: size * 0.13, weight: .bold))
                        .foregroundStyle(BeaconPalette.gold)
                        .shadow(color: BeaconPalette.gold.opacity(0.65), radius: 3)
                        .offset(x: size * 0.47)
                }
                .rotationEffect(angle)

                Image(systemName: "sparkles")
                    .font(.system(size: size * 0.12, weight: .bold))
                    .foregroundStyle(BeaconPalette.pink)
                    .offset(x: -size * 0.42, y: -size * 0.34)
                Image(systemName: "sparkle")
                    .font(.system(size: size * 0.09, weight: .bold))
                    .foregroundStyle(BeaconPalette.cyan)
                    .offset(x: size * 0.34, y: -size * 0.44)
            }
            .frame(width: size, height: size)
        }
    }
}
