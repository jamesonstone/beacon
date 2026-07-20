import SwiftUI

enum ProjectWatermarkPresentation {
    static let cycle: TimeInterval = 10
    static let minimumFrameInterval: TimeInterval = 1.0 / 12.0
    static let staticPhase = 0.5
    static let fontSize: CGFloat = 48
    static let minimumScaleFactor: CGFloat = 0.42

    static func phase(at date: Date) -> Double {
        let elapsed = date.timeIntervalSinceReferenceDate.truncatingRemainder(dividingBy: cycle)
        return (elapsed < 0 ? elapsed + cycle : elapsed) / cycle
    }

    static func displayedPhase(at date: Date, reduceMotion: Bool) -> Double {
        reduceMotion ? staticPhase : phase(at: date)
    }

    static func sweepStart(for phase: Double) -> UnitPoint {
        UnitPoint(x: -1 + 2 * min(max(phase, 0), 1), y: 0.2)
    }

    static func sweepEnd(for phase: Double) -> UnitPoint {
        let start = sweepStart(for: phase)
        return UnitPoint(x: start.x + 1, y: 0.8)
    }

    static func opacity(increasedContrast: Bool) -> Double {
        increasedContrast ? 1 : 0.86
    }

    static func saturation(differentiateWithoutColor: Bool) -> Double {
        differentiateWithoutColor ? 0 : 1
    }
}

struct ProjectWatermark: View {
    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.colorSchemeContrast) private var colorSchemeContrast

    let projectName: String
    let theme: BeaconTheme

    var body: some View {
        TimelineView(
            .animation(
                minimumInterval: ProjectWatermarkPresentation.minimumFrameInterval,
                paused: reduceMotion
            )
        ) { context in
            let phase = ProjectWatermarkPresentation.displayedPhase(
                at: context.date,
                reduceMotion: reduceMotion
            )
            Text(projectName.uppercased())
                .font(BeaconTypography.bold(ProjectWatermarkPresentation.fontSize))
                .tracking(-1.4)
                .lineLimit(1)
                .minimumScaleFactor(ProjectWatermarkPresentation.minimumScaleFactor)
                .allowsTightening(true)
                .foregroundStyle(sweepGradient(at: phase))
                .saturation(
                    ProjectWatermarkPresentation.saturation(
                        differentiateWithoutColor: differentiateWithoutColor
                    )
                )
                .opacity(
                    ProjectWatermarkPresentation.opacity(
                        increasedContrast: colorSchemeContrast == .increased
                    )
                )
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .padding(.horizontal, -8)
        }
        .accessibilityHidden(true)
        .allowsHitTesting(false)
    }

    private func sweepGradient(at phase: Double) -> LinearGradient {
        let palette = theme.projectWatermark
        return LinearGradient(
            stops: [
                .init(color: palette.base.color, location: 0),
                .init(color: palette.base.color, location: 0.28),
                .init(color: palette.highlightLeading.color, location: 0.42),
                .init(color: palette.highlightCenter.color, location: 0.50),
                .init(color: palette.highlightTrailing.color, location: 0.58),
                .init(color: palette.base.color, location: 0.72),
                .init(color: palette.base.color, location: 1),
            ],
            startPoint: ProjectWatermarkPresentation.sweepStart(for: phase),
            endPoint: ProjectWatermarkPresentation.sweepEnd(for: phase)
        )
    }
}
