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
                        .foregroundStyle(BeaconPalette.borderGradient(accent))
                    Text("Explicit check · no background polling")
                        .font(BeaconTypography.regular(8))
                        .foregroundStyle(BeaconPalette.lavender.opacity(0.78))
                        .lineLimit(1)
                }
                .layoutPriority(1)
                Spacer()
                refreshButton
            }

            if let error = state.dependencyLimitsError {
                Label(error, systemImage: "exclamationmark.triangle.fill")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconPalette.coral)
                    .lineLimit(3)
            }

            if let report = state.dependencyLimitsReport {
                HStack {
                    Label("Highest usage", systemImage: "gauge.with.dots.needle.50percent")
                        .font(BeaconTypography.regular(8))
                        .foregroundStyle(BeaconPalette.lavender)
                    Spacer()
                    Text(report.hasUsage ? "\(report.highestUsagePercent)%" : "No usage yet")
                        .font(BeaconTypography.semibold(10))
                        .foregroundStyle(accent)
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
                    .font(BeaconTypography.regular(8))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.7))
                    .frame(maxWidth: .infinity, alignment: .trailing)
            } else if state.isCheckingDependencyLimits {
                ProgressView("Asking gh for current limits…")
                    .tint(BeaconPalette.cyan)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                ContentUnavailableView(
                    "No limit check yet",
                    systemImage: "light.beacon.max.fill",
                    description: Text("Select Check Now to inspect Beacon's dependency allowances once.")
                )
                .symbolRenderingMode(.palette)
                .foregroundStyle(BeaconPalette.gold, BeaconPalette.cyan)
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
                    .foregroundStyle(BeaconPalette.cyan)
                Spacer()
                Text("\(dependency.buckets.count) buckets")
                    .font(BeaconTypography.regular(8))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.7))
            }
            ForEach(dependency.buckets) { bucket in
                bucketRow(bucket)
            }
        }
        .padding(10)
        .background(BeaconPalette.softGradient(accent), in: RoundedRectangle(cornerRadius: 10))
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
                Text(bucket.used > 0 ? "\(bucket.usagePercent)%" : "0%")
                    .font(BeaconTypography.semibold(9))
                    .monospacedDigit()
                    .foregroundStyle(rowAccent)
            }
            GeometryReader { geometry in
                ZStack(alignment: .leading) {
                    Capsule().fill(BeaconPalette.lavender.opacity(0.12))
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
            .font(BeaconTypography.regular(7))
            .foregroundStyle(BeaconPalette.lavender.opacity(0.72))
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
    var accentColor: Color {
        switch self {
        case .unmeasured: BeaconPalette.cyan
        case .healthy: BeaconPalette.mint
        case .warning: BeaconPalette.gold
        case .critical: BeaconPalette.coral
        }
    }
}
