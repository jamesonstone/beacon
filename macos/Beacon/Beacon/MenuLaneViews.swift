import SwiftUI

extension MenuView {
    @ViewBuilder
    func laneSection(_ title: String, symbol: String, accent: Color, lanes: [WorkLane]) -> some View {
        if !lanes.isEmpty {
            VStack(alignment: .leading, spacing: 6) {
                sectionHeader(title, symbol: symbol, accent: accent, count: lanes.count)
                ForEach(Array(lanes.enumerated()), id: \.element.id) { index, lane in
                    if index == 0 || lanes[index - 1].github != lane.github {
                        projectHeader(state.projectGroup(for: lane), accent: accent)
                    }
                    laneCard(lane)
                }
            }
        }
    }

    func projectHeader(_ project: ProjectLaneGroup, accent: Color) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(project.name)
                .font(BeaconTypography.bold(DashboardLanePresentation.projectNameSize))
                .foregroundStyle(accent)
                .lineLimit(1)
                .accessibilityAddTraits(.isHeader)
            if let progress = project.progress {
                Text("\(progress.feature) · \(actionLabel(progress.phase))")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                    .lineLimit(1)
            }
            let stage = state.stage(for: project.id)
            if stage != "ready" && stage != "cached" {
                Text(stage.replacingOccurrences(of: "_", with: " ").capitalized)
                    .font(BeaconTypography.medium(9))
                    .foregroundStyle(BeaconThemePreference.current().tokens.warning.color)
            }
            if let activity = state.activityChip(projectID: project.id) {
                externalActivityChip(activity)
            }
            Spacer()
        }
        .padding(.top, 2)
    }

    func laneCard(
        _ lane: WorkLane,
        density override: DashboardDensity? = nil,
        watermarkProjectName: String? = nil
    ) -> some View {
        let cardDensity = override ?? density
        return laneRow(
            lane,
            density: cardDensity,
            watermarkProjectName: watermarkProjectName
        )
            .contentShape(RoundedRectangle(cornerRadius: 9))
            .onTapGesture { state.open(lane) }
            .contextMenu { laneActions(lane) }
            .dropDestination(for: String.self) { laneIDs, _ in
                guard let draggedID = laneIDs.first else { return false }
                Task { await state.reorderLane(draggedID, before: lane.id) }
                return true
            }
            .accessibilityAction(named: "Move up") { Task { await state.moveLane(lane.id, by: -1) } }
            .accessibilityAction(named: "Move down") { Task { await state.moveLane(lane.id, by: 1) } }
            .richHoverPopover(
                enabled: RichHoverPresentation.cardDetailEnabled(
                    evidenceHoverLaneID: evidenceHoverLaneID,
                    laneID: lane.id
                )
            ) { laneDetailPanel(lane) }
    }

    func laneRow(
        _ lane: WorkLane,
        density: DashboardDensity,
        watermarkProjectName: String? = nil
    ) -> some View {
        let identity = DashboardLanePresentation.identity(for: lane)
        let accent = identity.accent.color
        return VStack(alignment: .leading, spacing: density.spacing) {
            HStack(spacing: 6) {
                if density != .comfortable {
                    projectGlyph(lane, accent: accent)
                }
                Text(workItemTitle(lane))
                    .font(BeaconTypography.semibold(density.titleSize))
                    .lineLimit(density.titleLines)
                Spacer()
                if let pullRequest = lane.pullRequest {
                    Label("PR #\(pullRequest.number)", systemImage: identity.symbol)
                        .font(BeaconTypography.identifier(10, weight: .medium))
                        .foregroundStyle(accent)
                } else if let issue = lane.issue {
                    Label("Issue #\(issue.number)", systemImage: identity.symbol)
                        .font(BeaconTypography.identifier(10, weight: .medium))
                        .foregroundStyle(accent)
                } else if !lane.branch.isEmpty {
                    Label("Local · \(lane.branch)", systemImage: identity.symbol)
                        .font(BeaconTypography.identifier(9, weight: .medium))
                        .foregroundStyle(accent)
                        .lineLimit(1)
                } else {
                    Label("Manual", systemImage: "pencil")
                        .font(BeaconTypography.medium(9))
                        .foregroundStyle(accent)
                }
                if DashboardLanePresentation.showsCheckoutWarning(for: lane) {
                    checkoutWarningButton(lane)
                }
                if DashboardLanePresentation.showsIgnoreAction(in: selectedDashboardTab) {
                    ignoreButton(lane, compact: density != .comfortable)
                }
                laneDragHandle(lane)
            }
            HStack {
                Text(actionLabel(lane.nextAction))
                    .font(BeaconTypography.medium(10))
                    .foregroundStyle(accent)
                Spacer()
                if let activity = state.activityChip(projectID: lane.github, laneID: lane.id) {
                    externalActivityChip(activity)
                }
            }
            if density != .dense, let attention = lane.attention {
                Text("\(attention.delta) · \(timeSinceActivity(lane.updatedAt))")
                    .font(BeaconTypography.identifier(9))
                    .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                    .lineLimit(1)
                if density == .comfortable {
                    tagChips(lane, tags: attention.tags ?? [], accent: accent)
                }
                if density == .comfortable, let note = attention.note, !note.isEmpty {
                    Label("\(note)\(attention.noteStale ? " · evidence changed" : "")", systemImage: "note.text")
                        .font(BeaconTypography.regular(9))
                        .foregroundStyle(attention.noteStale ? BeaconThemePreference.current().tokens.warning.color : BeaconThemePreference.current().tokens.textSecondary.color)
                        .lineLimit(2)
                }
            }
            if density == .comfortable, let progress = lane.progress {
                Text("Kit \(actionLabel(progress.phase)) · \(progress.summary)")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                    .lineLimit(1)
            }
            evidenceBadges(lane, condensed: density == .dense)
        }
        .padding(density.cardPadding)
        .background {
            ZStack {
                RoundedRectangle(cornerRadius: 9)
                    .fill(BeaconThemePreference.current().tokens.surface.color)
                if let watermarkProjectName,
                   !watermarkProjectName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    ProjectWatermark(projectName: watermarkProjectName, theme: theme)
                        .clipShape(RoundedRectangle(cornerRadius: 9))
                }
            }
        }
        .overlay {
            RoundedRectangle(cornerRadius: 9)
                .strokeBorder(interfaceBorderColor, lineWidth: colorSchemeContrast == .increased ? 1.2 : 0.8)
        }
        .shadow(color: accent.opacity(0.09), radius: 4, y: 2)
    }

    func laneDragHandle(_ lane: WorkLane) -> some View {
        Image(systemName: "line.3.horizontal")
            .font(.system(size: 10, weight: .semibold))
            .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
            .frame(width: 18, height: 22)
            .contentShape(Rectangle())
            .draggable(lane.id)
            .help(selectedDashboardTab == .parking
                ? "Drag to reorder in Parking Lot, or drop on Following to Resume"
                : "Drag to reorder within this project and work-item type, or drop on Parking Lot to Ignore")
            .accessibilityLabel("Reorder \(workItemTitle(lane))")
            .accessibilityHint(selectedDashboardTab == .parking
                ? "Use the card actions to move this item within Parking Lot"
                : "Use the card actions to move this item within its project and work-item type")
    }

    func externalActivityChip(_ activity: ExternalActivityChip) -> some View {
        let accent: Color
        switch activity.state {
        case "needs_attention": accent = BeaconThemePreference.current().tokens.warning.color
        case "working": accent = BeaconThemePreference.current().tokens.info.color
        default: accent = BeaconThemePreference.current().tokens.textSecondary.color
        }
        return Text(activity.label)
            .font(BeaconTypography.semibold(8))
            .foregroundStyle(accent)
            .lineLimit(1)
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: Capsule())
            .overlay { Capsule().strokeBorder(accent.opacity(0.42), lineWidth: 0.7) }
            .help(activity.state == "needs_attention"
                ? "Latest observed attention request; the provider may not report when it is resolved."
                : "Transient external agent activity; this does not change Beacon evidence or ordering.")
    }

    func ignoreButton(_ lane: WorkLane, compact: Bool) -> some View {
        Button {
            Task { await state.ignoreLane(lane) }
        } label: {
            Group {
                if compact {
                    Image(systemName: "pause.circle.fill")
                } else {
                    Label("Ignore", systemImage: "pause.circle.fill")
                }
            }
            .font(BeaconTypography.semibold(9))
            .padding(.horizontal, compact ? 5 : 7)
            .padding(.vertical, 4)
        }
        .buttonStyle(.plain)
        .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
        .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: Capsule())
        .overlay { Capsule().strokeBorder(BeaconThemePreference.current().tokens.textSecondary.color.opacity(0.4), lineWidth: 0.7) }
        .help("Move to Parking Lot")
        .accessibilityLabel("Ignore \(workItemTitle(lane))")
        .accessibilityHint("Moves this lane to the Parking Lot")
    }

    func checkoutWarningButton(_ lane: WorkLane) -> some View {
        let critical = DashboardLanePresentation.checkoutWarningIsCritical(for: lane)
        let accent = critical ? BeaconThemePreference.current().tokens.danger.color : BeaconThemePreference.current().tokens.warning.color
        return Button {
            showRepositorySync()
        } label: {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(BeaconTypography.semibold(10))
                .padding(5)
        }
        .buttonStyle(.plain)
        .foregroundStyle(accent)
        .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: Circle())
        .overlay { Circle().strokeBorder(accent.opacity(0.48), lineWidth: 0.7) }
        .help(lane.checkoutWarning?.message ?? "Merged branch remains checked out locally")
        .accessibilityLabel("Merged branch warning for \(workItemTitle(lane))")
        .accessibilityHint("Opens Repository Sync without changing the repository")
    }

    func projectGlyph(_ lane: WorkLane, accent: Color) -> some View {
        Text(String((lane.repository.isEmpty ? "?" : lane.repository).prefix(1)).uppercased())
            .font(BeaconTypography.bold(10))
            .foregroundStyle(accent)
            .frame(width: 24, height: 24)
            .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 7))
            .overlay {
                RoundedRectangle(cornerRadius: 7).strokeBorder(accent.opacity(0.45), lineWidth: 0.7)
            }
    }
}
