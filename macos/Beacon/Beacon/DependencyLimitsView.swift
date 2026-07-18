import Foundation
import SwiftUI

struct DependencyLimitsView: View {
    @ObservedObject var state: AppState
    let onClose: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 8) {
                Button(action: onClose) {
                    Image(systemName: "chevron.left")
                        .frame(width: 24, height: 24)
                }
                .buttonStyle(.plain)
                .font(BeaconTypography.medium(9))
                .help("Back to Dashboard")
                .accessibilityLabel("Dashboard")

                VStack(alignment: .leading, spacing: 1) {
                    Text("Dependency Limits")
                        .font(BeaconTypography.semibold(12))
                        .foregroundStyle(BeaconThemePreference.current().tokens.textPrimary.color)
                    Text("Explicit check · no background polling")
                        .font(BeaconTypography.regular(8))
                        .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                        .lineLimit(1)
                }
                .layoutPriority(1)
                Spacer()
                refreshButton
            }

            if let error = state.dependencyLimitsError {
                Label(error, systemImage: "exclamationmark.triangle.fill")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconThemePreference.current().tokens.danger.color)
                    .lineLimit(3)
            }

            if let report = state.dependencyLimitsReport {
                HStack {
                    Label("Highest usage", systemImage: "gauge.with.dots.needle.50percent")
                        .font(BeaconTypography.regular(8))
                        .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
                    Spacer()
                    if report.hasUsage {
                        Label(
                            "\(report.highestUsagePercent)% · \(state.dependencyUsageLevel.title)",
                            systemImage: state.dependencyUsageLevel.symbol
                        )
                        .font(BeaconTypography.counter(10, weight: .semibold))
                        .foregroundStyle(accent)
                    } else {
                        Label("No usage yet", systemImage: DependencyUsageLevel.unmeasured.symbol)
                            .font(BeaconTypography.semibold(10))
                            .foregroundStyle(accent)
                    }
                }

                ScrollView {
                    LazyVStack(spacing: 9) {
                        ForEach(report.dependencies) { dependency in
                            dependencyCard(dependency)
                        }
                    }
                    .padding(.vertical, 2)
                }

                Text("Checked \(checkedLabel(report.checkedAt)) with one gh api rate_limit request.")
                    .font(BeaconTypography.identifier(8))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                    .frame(maxWidth: .infinity, alignment: .trailing)
            } else if state.isCheckingDependencyLimits {
                ProgressView("Asking gh for current limits…")
                    .tint(BeaconThemePreference.current().tokens.info.color)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                ContentUnavailableView(
                    "No limit check yet",
                    systemImage: "light.beacon.max.fill",
                    description: Text("Select Check Now to inspect Beacon's dependency allowances once.")
                )
                .symbolRenderingMode(.palette)
                .foregroundStyle(BeaconThemePreference.current().tokens.warning.color, BeaconThemePreference.current().tokens.info.color)
            }
        }
    }

    private var refreshButton: some View {
        Button {
            Task { await state.checkDependencyLimits() }
        } label: {
            if state.isCheckingDependencyLimits {
                ProgressView().controlSize(.small)
            } else {
                Label("Check Now", systemImage: "gauge.with.dots.needle.50percent")
            }
        }
        .buttonStyle(.bordered)
        .font(BeaconTypography.medium(9))
        .disabled(state.isCheckingDependencyLimits)
        .help("Run one bounded gh api rate_limit request")
    }

    private func dependencyCard(_ dependency: DependencyLimit) -> some View {
        VStack(alignment: .leading, spacing: 7) {
            HStack {
                Label(dependency.name, systemImage: "terminal.fill")
                    .font(BeaconTypography.semibold(10))
                    .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                Spacer()
                Text("\(dependency.buckets.count) buckets")
                    .font(BeaconTypography.counter(8))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
            }
            ForEach(dependency.buckets) { bucket in
                bucketRow(bucket)
            }
        }
        .padding(10)
        .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 10))
        .overlay {
            RoundedRectangle(cornerRadius: 10)
                .strokeBorder(accent.opacity(0.34), lineWidth: 0.7)
        }
    }

    private func bucketRow(_ bucket: DependencyLimitBucket) -> some View {
        let level = DependencyLimitPresentation.level(percent: bucket.usagePercent, hasUsage: bucket.used > 0)
        let rowAccent = level.accentColor
        return VStack(alignment: .leading, spacing: 3) {
            HStack {
                Text(bucket.name)
                    .font(BeaconTypography.medium(9))
                Spacer()
                Label(
                    "\(bucket.used > 0 ? bucket.usagePercent : 0)% · \(level.title)",
                    systemImage: level.symbol
                )
                .font(BeaconTypography.counter(9, weight: .semibold))
                .foregroundStyle(rowAccent)
            }
            GeometryReader { geometry in
                ZStack(alignment: .leading) {
                    Capsule().fill(BeaconThemePreference.current().tokens.textSecondary.color.opacity(0.12))
                    Capsule()
                        .fill(rowAccent)
                        .frame(width: geometry.size.width * CGFloat(bucket.usagePercent) / 100)
                }
            }
            .frame(height: 5)
            HStack {
                Text("\(bucket.used) used · \(bucket.remaining) remaining · \(bucket.limit) total")
                Spacer()
                Text("Resets \(resetLabel(bucket.resetAt))")
            }
            .font(BeaconTypography.identifier(7))
            .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
        }
    }

    private var accent: Color {
        state.dependencyUsageLevel.accentColor
    }

    private func checkedLabel(_ value: String) -> String {
        guard let date = parsedDate(value) else { return value }
        return date.formatted(date: .omitted, time: .standard)
    }

    private func resetLabel(_ value: String) -> String {
        guard let date = parsedDate(value) else { return value }
        return RelativeDateTimeFormatter().localizedString(for: date, relativeTo: Date())
    }

    private func parsedDate(_ value: String) -> Date? {
        let fractional = ISO8601DateFormatter()
        fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return fractional.date(from: value) ?? ISO8601DateFormatter().date(from: value)
    }
}

extension DependencyUsageLevel {
    var title: String {
        switch self {
        case .unmeasured: "Unmeasured"
        case .healthy: "Healthy"
        case .warning: "Warning"
        case .critical: "Critical"
        }
    }

    var symbol: String {
        switch self {
        case .unmeasured: "questionmark.circle"
        case .healthy: "checkmark.circle.fill"
        case .warning: "exclamationmark.circle.fill"
        case .critical: "exclamationmark.triangle.fill"
        }
    }

    var accentColor: Color {
        switch self {
        case .unmeasured: BeaconThemePreference.current().tokens.info.color
        case .healthy: BeaconThemePreference.current().tokens.success.color
        case .warning: BeaconThemePreference.current().tokens.warning.color
        case .critical: BeaconThemePreference.current().tokens.danger.color
        }
    }
}
