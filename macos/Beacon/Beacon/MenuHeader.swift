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

private struct TaxonomyInfoButton: View {
    @State private var isPresented = false

    var body: some View {
        Button {
            isPresented.toggle()
        } label: {
            Image(systemName: "info.circle.fill")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(BeaconPalette.lavender)
                .frame(width: 28, height: 28)
                .background(BeaconPalette.softGradient(BeaconPalette.lavender), in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(BeaconPalette.lavender.opacity(0.35), lineWidth: 0.7)
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
                        .foregroundStyle(BeaconPalette.mint)
                    Text("Every project and work item uses the same hierarchy. Color reinforces these labels and symbols, but never replaces them.")
                        .font(BeaconTypography.regular(10))
                    taxonomyRow("Membership", symbol: "star.fill", text: "Following is explicit. Activity can place a non-followed project in Recently Updated; otherwise it remains Quiet.", accent: BeaconPalette.mint)
                    taxonomyRow("Identity", symbol: "number", text: "Mint means Local work, cyan means Pull Request, and pink means Issue. Identity color is never status.", accent: BeaconPalette.cyan)
                    taxonomyRow("Attention", symbol: "scope", text: "Active, Waiting, Recently Active, or Parked. Ignore moves only that lane to Parking Lot; Resume returns it to Go-derived attention without changing project membership.", accent: BeaconPalette.mint)
                    taxonomyRow("Next action", symbol: "arrow.right.circle.fill", text: "One canonical instruction such as Address Review, Fix CI, Push Branch, or Start Issue.", accent: BeaconPalette.gold)
                    taxonomyRow("Order", symbol: "line.3.horizontal", text: "One persisted lane order is projected into every status and layout. Drag within a status; drop on Following or Parking Lot only to Resume or Ignore.", accent: BeaconPalette.lavender)
                    taxonomyRow("Evidence exceptions", symbol: "exclamationmark.bubble.fill", text: "Only actionable or uncertain signals appear as badges. Healthy clean, current, approved, and successful values are quiet; Stale means evidence exceeded the configured freshness window.", accent: BeaconPalette.pink)
                    taxonomyRow("Local context", symbol: "tag.fill", text: "Your tags and notes add context but never change Beacon's inferred status or next action.", accent: BeaconPalette.lavender)
                    Divider()
                    Label("PR feedback · N", systemImage: "text.bubble.fill")
                        .font(BeaconTypography.semibold(11))
                        .foregroundStyle(BeaconPalette.pink)
                    Text("N is the number of unresolved pull request review threads, not issue comments. Hover the badge to see each file, reviewer, Markdown comment, timestamp, and direct GitHub link. Hover any card for its issue, PR, or local-work description.")
                        .font(BeaconTypography.regular(10))
                }
                .padding(14)
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .frame(width: 420, height: 390)
            .background(BeaconPalette.panelBackground)
        }
    }

    private func taxonomyRow(_ title: String, symbol: String, text: String, accent: Color) -> some View {
        HStack(alignment: .top, spacing: 9) {
            Image(systemName: symbol)
                .foregroundStyle(accent)
                .frame(width: 20)
            VStack(alignment: .leading, spacing: 2) {
                Text(title).font(BeaconTypography.semibold(11))
                Text(text).font(BeaconTypography.regular(9)).foregroundStyle(BeaconPalette.lavender)
            }
        }
    }
}
