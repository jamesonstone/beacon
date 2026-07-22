import SwiftUI

enum ProjectWatermarkPresentation {
    static let highlightPosition = 0.5
    static let fontSize: CGFloat = 48
    static let minimumScaleFactor: CGFloat = 0.42

    static func highlightStart(for position: Double) -> UnitPoint {
        UnitPoint(x: -1 + 2 * min(max(position, 0), 1), y: 0.2)
    }

    static func highlightEnd(for position: Double) -> UnitPoint {
        let start = highlightStart(for: position)
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
    @Environment(\.colorSchemeContrast) private var colorSchemeContrast

    let projectName: String
    let theme: BeaconTheme

    var body: some View {
        Text(projectName.uppercased())
            .font(BeaconTypography.bold(ProjectWatermarkPresentation.fontSize))
            .tracking(-1.4)
            .lineLimit(1)
            .minimumScaleFactor(ProjectWatermarkPresentation.minimumScaleFactor)
            .allowsTightening(true)
            .foregroundStyle(
                highlightGradient(at: ProjectWatermarkPresentation.highlightPosition)
            )
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
            .accessibilityHidden(true)
            .allowsHitTesting(false)
    }

    private func highlightGradient(at position: Double) -> LinearGradient {
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
            startPoint: ProjectWatermarkPresentation.highlightStart(for: position),
            endPoint: ProjectWatermarkPresentation.highlightEnd(for: position)
        )
    }
}
