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
                    .font(BeaconTypography.counter(10, weight: .medium))
                    .foregroundStyle(BeaconThemePreference.current().tokens.success.color)
                    .lineLimit(1)
                if let generatedAt = state.snapshot?.generatedAt {
                    Text("Updated \(timeSinceActivity(generatedAt))")
                        .font(BeaconTypography.identifier(8))
                        .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                        .lineLimit(1)
                }
            }
            Spacer()
            refreshButton
            repositorySyncButton
            dependencyLimitsButton
            TaxonomyInfoButton()
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
                        .tint(BeaconThemePreference.current().tokens.warning.color)
                } else {
                    Image(systemName: "arrow.triangle.2.circlepath")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(BeaconThemePreference.current().tokens.warning.color)
                }
            }
            .frame(width: 28, height: 28)
            .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 8))
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(BeaconThemePreference.current().tokens.warning.color.opacity(0.42), lineWidth: 0.7)
            }
            .overlay(alignment: .topTrailing) {
                if !state.repositoriesNeedingSync.isEmpty {
                    Text("\(min(state.repositoriesNeedingSync.count, 99))")
                        .font(BeaconTypography.counter(11, weight: .bold))
                        .foregroundStyle(Color.black)
                        .padding(3)
                        .background(BeaconThemePreference.current().tokens.warning.color, in: Circle())
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
                    VStack(spacing: 0) {
                        Image(systemName: state.dependencyUsageLevel.symbol)
                            .font(.system(size: 7, weight: .bold))
                            .foregroundStyle(accent)
                        Text("\(state.dependencyUsagePercent)%")
                            .font(BeaconTypography.counter(11, weight: .heavy))
                            .foregroundStyle(BeaconThemePreference.current().tokens.textPrimary.color)
                    }
                } else {
                    Image(systemName: "gauge.with.dots.needle.50percent")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(accent)
                }
            }
            .frame(width: 28, height: 28)
            .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 8))
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
        return "Dependency Limits, highest usage \(state.dependencyUsagePercent) percent, \(state.dependencyUsageLevel.title)"
    }

    var refreshButton: some View {
        Button {
            Task { await state.scan() }
        } label: {
            Group {
                if state.isScanning {
                    ProgressView()
                        .controlSize(.small)
                        .tint(BeaconThemePreference.current().tokens.success.color)
                } else {
                    Image(systemName: "arrow.clockwise")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(BeaconThemePreference.current().tokens.success.color)
                }
            }
            .frame(width: 28, height: 28)
            .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 8))
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(BeaconThemePreference.current().tokens.success.color.opacity(0.42), lineWidth: 0.7)
            }
        }
        .buttonStyle(.plain)
        .disabled(state.isScanning)
        .help(state.isScanning ? "Scanning Git and GitHub evidence" : "Scan Now — refresh Git and GitHub evidence")
        .accessibilityLabel(state.isScanning ? "Scan in progress" : "Scan Now")
    }
}

private struct TaxonomyInfoButton: View {
    @State private var isPresented = false

    var body: some View {
        Button {
            isPresented.toggle()
        } label: {
            Image(systemName: "info.circle.fill")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
                .frame(width: 28, height: 28)
                .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(BeaconThemePreference.current().tokens.border.color, lineWidth: 0.7)
                }
        }
        .buttonStyle(.plain)
        .help("How Beacon organizes identity, status, actions, and evidence")
        .accessibilityLabel("Beacon taxonomy guide")
        .popover(isPresented: $isPresented, arrowEdge: .bottom) {
            ScrollView {
                VStack(alignment: .leading, spacing: 13) {
                    Label("Beacon's universal workflow", systemImage: "point.3.connected.trianglepath.dotted")
                        .font(BeaconTypography.bold(14))
                        .foregroundStyle(BeaconThemePreference.current().tokens.success.color)
                    Text("Every project and work item uses the same hierarchy. Color reinforces these labels and symbols, but never replaces them.")
                        .font(BeaconTypography.regular(10))
                    taxonomyRow("Membership", symbol: "star.fill", text: "Following is explicit. Activity can place a non-followed project in Recently Updated; otherwise it remains Quiet.", accent: BeaconThemePreference.current().tokens.success.color)
                    taxonomyRow("Identity", symbol: "number", text: "Every card explicitly says Local, PR, Issue, or Manual and pairs that label with a stable SF Symbol. Each theme supplies distinct semantic identity accents; identity color is never status.", accent: BeaconThemePreference.current().tokens.info.color)
                    taxonomyRow("Attention", symbol: "scope", text: "Active, Waiting, Recently Active, or Parked. Ignore moves only that lane to Parking Lot; Resume returns it to Go-derived attention without changing project membership.", accent: BeaconThemePreference.current().tokens.success.color)
                    taxonomyRow("Next action", symbol: "arrow.right.circle.fill", text: "One canonical instruction such as Address Review, Fix CI, Push Branch, or Start Issue.", accent: BeaconThemePreference.current().tokens.warning.color)
                    taxonomyRow("Order", symbol: "line.3.horizontal", text: "One persisted lane order is projected into every status and layout. Drag within a status; drop on Following or Parking Lot only to Resume or Ignore.", accent: BeaconThemePreference.current().tokens.textSecondary.color)
                    taxonomyRow("Evidence exceptions", symbol: "exclamationmark.bubble.fill", text: "Only actionable or uncertain signals appear as badges. Healthy clean, current, approved, and successful values are quiet; Stale means evidence exceeded the configured freshness window.", accent: BeaconThemePreference.current().tokens.warning.color)
                    taxonomyRow("Local context", symbol: "tag.fill", text: "Your tags and notes add context but never change Beacon's inferred status or next action.", accent: BeaconThemePreference.current().tokens.textSecondary.color)
                    taxonomyRow("Appearance", symbol: "paintpalette.fill", text: "All five themes preserve the same labels, symbols, status meanings, and Local/PR/Issue identities. Settings → Appearance changes only semantic presentation and persists across both Beacon surfaces.", accent: BeaconThemePreference.current().tokens.accent.color)
                    Divider()
                    Label("PR feedback · N", systemImage: "text.bubble.fill")
                        .font(BeaconTypography.semibold(11))
                        .foregroundStyle(BeaconThemePreference.current().tokens.warning.color)
                    Text("N is the number of unresolved pull request review threads, not issue comments. Hover the badge to see each file, reviewer, Markdown comment, timestamp, and direct GitHub link. Hover any card for its issue, PR, or local-work description.")
                        .font(BeaconTypography.regular(10))
                }
                .padding(14)
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .frame(width: 440, height: 430)
            .background(BeaconThemePreference.current().tokens.surfaceOverlay.color)
        }
    }

    private func taxonomyRow(_ title: String, symbol: String, text: String, accent: Color) -> some View {
        HStack(alignment: .top, spacing: 9) {
            Image(systemName: symbol)
                .foregroundStyle(accent)
                .frame(width: 20)
            VStack(alignment: .leading, spacing: 2) {
                Text(title).font(BeaconTypography.semibold(11))
                Text(text).font(BeaconTypography.regular(9)).foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
            }
        }
    }
}
