import SwiftUI

extension MenuView {
    var header: some View {
        HStack(alignment: .center, spacing: 8) {
            HStack(spacing: 6) {
                BeaconRocketMark()
                NeonWaveWordmark("Beacon")
                    .font(BeaconTypography.bold(17))
            }
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                Text("\(state.inProgressCount) lanes in focus")
                    .font(BeaconTypography.medium(10))
                    .foregroundStyle(BeaconPalette.mint)
                    .lineLimit(1)
                if let generatedAt = state.snapshot?.generatedAt {
                    Text("Updated \(timeSinceActivity(generatedAt))")
                        .font(BeaconTypography.regular(8))
                        .foregroundStyle(BeaconPalette.lavender.opacity(0.82))
                        .lineLimit(1)
                }
            }
            Spacer()
            refreshButton
            repositorySyncButton
            dependencyLimitsButton
            viewModeMenu
            settingsMenu
        }
    }

    var repositorySyncButton: some View {
        Button {
            let isOpening = toggleDashboardDestination(.repositorySync)
            if isOpening, state.repositorySyncReport == nil, !state.isCheckingRepositorySync {
                Task { await state.checkRepositorySync(refresh: false) }
            }
        } label: {
            Group {
                if state.isCheckingRepositorySync || state.isApplyingRepositorySync {
                    ProgressView()
                        .controlSize(.small)
                        .tint(BeaconPalette.gold)
                } else {
                    Image(systemName: "arrow.triangle.2.circlepath")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(BeaconPalette.gold)
                }
            }
            .frame(width: 28, height: 28)
            .background(BeaconPalette.softGradient(BeaconPalette.gold), in: RoundedRectangle(cornerRadius: 8))
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(BeaconPalette.gold.opacity(0.42), lineWidth: 0.7)
            }
            .overlay(alignment: .topTrailing) {
                if !state.repositoriesNeedingSync.isEmpty {
                    Text("\(min(state.repositoriesNeedingSync.count, 99))")
                        .font(.system(size: 7, weight: .bold, design: .rounded))
                        .foregroundStyle(Color.black)
                        .padding(3)
                        .background(BeaconPalette.gold, in: Circle())
                        .offset(x: 3, y: -3)
                }
            }
        }
        .buttonStyle(.plain)
        .help(dashboardDestination == .repositorySync
            ? "Return to Following"
            : "Repository Sync — check and fast-forward local default branches")
        .accessibilityLabel("Repository Sync, \(state.repositoriesNeedingSync.count) need attention")
    }

    var dependencyLimitsButton: some View {
        let accent = state.dependencyUsageLevel.accentColor
        return Button {
            let isOpening = toggleDashboardDestination(.dependencyLimits)
            if isOpening, state.dependencyLimitsReport == nil, !state.isCheckingDependencyLimits {
                Task { await state.checkDependencyLimits() }
            }
        } label: {
            Group {
                if state.isCheckingDependencyLimits {
                    ProgressView()
                        .controlSize(.small)
                        .tint(accent)
                } else if state.dependencyLimitsReport?.hasUsage == true {
                    Text("\(state.dependencyUsagePercent)%")
                        .font(.system(size: 8, weight: .heavy, design: .rounded))
                        .monospacedDigit()
                        .foregroundStyle(accent)
                        .minimumScaleFactor(0.75)
                } else {
                    Image(systemName: "gauge.with.dots.needle.50percent")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(accent)
                }
            }
            .frame(width: 28, height: 28)
            .background(BeaconPalette.softGradient(accent), in: RoundedRectangle(cornerRadius: 8))
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(accent.opacity(0.42), lineWidth: 0.7)
            }
        }
        .buttonStyle(.plain)
        .help(dashboardDestination == .dependencyLimits
            ? "Return to Following"
            : "Dependency Limits — check gh allowance explicitly")
        .accessibilityLabel(dependencyLimitsAccessibilityLabel)
    }

    var dependencyLimitsAccessibilityLabel: String {
        guard state.dependencyLimitsReport != nil else { return "Dependency Limits, not checked" }
        return "Dependency Limits, highest usage \(state.dependencyUsagePercent) percent"
    }

    var refreshButton: some View {
        Button {
            Task { await state.scan() }
        } label: {
            Group {
                if state.isScanning {
                    ProgressView()
                        .controlSize(.small)
                        .tint(BeaconPalette.mint)
                } else {
                    Image(systemName: "arrow.clockwise")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(BeaconPalette.mint)
                }
            }
            .frame(width: 28, height: 28)
            .background(BeaconPalette.softGradient(BeaconPalette.mint), in: RoundedRectangle(cornerRadius: 8))
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(BeaconPalette.mint.opacity(0.42), lineWidth: 0.7)
            }
        }
        .buttonStyle(.plain)
        .disabled(state.isScanning)
        .help(state.isScanning ? "Scanning Git and GitHub evidence" : "Scan Now — refresh Git and GitHub evidence")
        .accessibilityLabel(state.isScanning ? "Scan in progress" : "Scan Now")
    }
}
