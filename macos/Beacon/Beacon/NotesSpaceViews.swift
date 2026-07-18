import SwiftUI

enum BeaconSpaceMotion {
    static func phase(at date: Date, duration: TimeInterval) -> Double {
        guard duration > 0 else { return 0 }
        return date.timeIntervalSinceReferenceDate.truncatingRemainder(dividingBy: duration) / duration
    }

    static func angle(at date: Date, duration: TimeInterval) -> Angle {
        .degrees(phase(at: date, duration: duration) * 360)
    }
}

struct BeaconRocketMark: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        TimelineView(.animation(minimumInterval: 1.0 / 20.0, paused: reduceMotion)) { context in
            let phase = reduceMotion ? 0.12 : BeaconSpaceMotion.phase(at: context.date, duration: 9)
            ZStack {
                Circle()
                    .fill(BeaconThemePreference.current().tokens.textSecondary.color.opacity(0.72))
                    .frame(width: 2, height: 2)
                    .offset(x: -7, y: 6)
                Circle()
                    .fill(BeaconThemePreference.current().tokens.info.color.opacity(0.86))
                    .frame(width: 2, height: 2)
                    .offset(x: 7, y: -7)
                Capsule()
                    .fill(BeaconThemePreference.current().tokens.identityIssue.color.opacity(0.38))
                    .frame(width: 7, height: 1.5)
                    .rotationEffect(.degrees(-35))
                    .offset(x: -4, y: 5)
                Text("🚀")
                    .font(.system(size: 13))
                    .rotationEffect(.degrees(42 + sin(phase * .pi * 2) * 4))
                    .offset(
                        x: cos(phase * .pi * 2) * 2.3,
                        y: sin(phase * .pi * 2) * 2.3
                    )
                    .shadow(color: BeaconThemePreference.current().tokens.info.color.opacity(0.55), radius: 2)
            }
        }
        .frame(width: 22, height: 22)
        .accessibilityHidden(true)
    }
}

struct NotesSolarSystemMark: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        TimelineView(.animation(minimumInterval: 1.0 / 20.0, paused: reduceMotion)) { context in
            let angle = reduceMotion ? Angle.degrees(35) : BeaconSpaceMotion.angle(at: context.date, duration: 7)
            ZStack {
                Ellipse()
                    .stroke(BeaconThemePreference.current().tokens.textSecondary.color.opacity(0.42), lineWidth: 0.7)
                    .frame(width: 22, height: 11)
                    .rotationEffect(.degrees(-18))
                Circle()
                    .fill(BeaconThemePreference.current().tokens.warning.color)
                    .frame(width: 6, height: 6)
                    .shadow(color: BeaconThemePreference.current().tokens.warning.color.opacity(0.75), radius: 3)
                Circle()
                    .fill(BeaconThemePreference.current().tokens.info.color)
                    .frame(width: 4, height: 4)
                    .offset(x: 10)
                    .rotationEffect(angle)
                Circle()
                    .fill(BeaconThemePreference.current().tokens.identityIssue.color)
                    .frame(width: 2.5, height: 2.5)
                    .offset(x: -6)
                    .rotationEffect(.degrees(-angle.degrees * 1.6))
            }
        }
        .frame(width: 26, height: 20)
        .accessibilityHidden(true)
    }
}

struct EmptyNotesSpaceView: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        VStack(spacing: 8) {
            TimelineView(.animation(minimumInterval: 1.0 / 16.0, paused: reduceMotion)) { context in
                let angle = reduceMotion ? Angle.degrees(-24) : BeaconSpaceMotion.angle(at: context.date, duration: 12)
                ZStack {
                    star(x: -54, y: -22, color: BeaconThemePreference.current().tokens.textSecondary.color)
                    star(x: 47, y: -30, color: BeaconThemePreference.current().tokens.info.color)
                    star(x: 60, y: 19, color: BeaconThemePreference.current().tokens.identityIssue.color)
                    star(x: -43, y: 29, color: BeaconThemePreference.current().tokens.success.color)
                    Ellipse()
                        .stroke(
                            BeaconThemePreference.current().tokens.borderStrong.color.opacity(0.58),
                            style: StrokeStyle(lineWidth: 1, dash: [3, 3])
                        )
                        .frame(width: 112, height: 52)
                        .rotationEffect(.degrees(-12))
                    Circle()
                        .fill(BeaconThemePreference.current().tokens.surfaceRaised.color)
                        .frame(width: 31, height: 31)
                        .overlay {
                            Ellipse()
                                .stroke(BeaconThemePreference.current().tokens.warning.color.opacity(0.65), lineWidth: 2)
                                .frame(width: 43, height: 12)
                                .rotationEffect(.degrees(-14))
                        }
                        .shadow(color: BeaconThemePreference.current().tokens.identityIssue.color.opacity(0.38), radius: 8)
                    Text("🚀")
                        .font(.system(size: 15))
                        .rotationEffect(angle + .degrees(90))
                        .offset(x: 54)
                        .rotationEffect(angle)
                        .shadow(color: BeaconThemePreference.current().tokens.info.color.opacity(0.55), radius: 3)
                }
                .frame(width: 148, height: 88)
            }
            .accessibilityHidden(true)

            Text("No detail notes yet")
                .font(BeaconTypography.semibold(17))
                .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
            Text("Create one above or from a line in General.")
                .font(BeaconTypography.regular(9))
                .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
        }
        .multilineTextAlignment(.center)
        .accessibilityElement(children: .combine)
    }

    private func star(x: CGFloat, y: CGFloat, color: Color) -> some View {
        Image(systemName: "sparkle")
            .font(.system(size: 7, weight: .bold))
            .foregroundStyle(color.opacity(0.82))
            .offset(x: x, y: y)
    }
}
