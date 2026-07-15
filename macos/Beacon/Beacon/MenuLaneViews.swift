import SwiftUI

extension MenuView {
    @ViewBuilder
    func laneSection(_ title: String, symbol: String, accent: Color, lanes: [WorkLane]) -> some View {
        if !lanes.isEmpty {
            VStack(alignment: .leading, spacing: 6) {
                sectionHeader(title, symbol: symbol, accent: accent, count: lanes.count)
                ForEach(state.projectGroups(for: lanes)) { project in
                    projectHeader(project, accent: accent)
                    ForEach(project.lanes) { lane in
                        laneCard(lane)
                    }
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
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.85))
                    .lineLimit(1)
            }
            let stage = state.stage(for: project.id)
            if stage != "ready" && stage != "cached" {
                Text(stage.replacingOccurrences(of: "_", with: " ").capitalized)
                    .font(BeaconTypography.medium(9))
                    .foregroundStyle(BeaconPalette.gold)
            }
            Spacer()
        }
        .padding(.top, 2)
    }

    func laneCard(_ lane: WorkLane, compact: Bool = false) -> some View {
        laneRow(lane, compact: compact)
            .contentShape(RoundedRectangle(cornerRadius: 9))
            .onTapGesture { state.open(lane) }
            .contextMenu { laneActions(lane) }
    }

    func laneRow(_ lane: WorkLane, compact: Bool = false) -> some View {
        let accent = DashboardLanePresentation.identity(for: lane).accent.color
        return VStack(alignment: .leading, spacing: 5) {
            HStack {
                if compact {
                    projectGlyph(lane, accent: accent)
                }
                Text(workItemTitle(lane))
                    .font(BeaconTypography.semibold(
                        compact ? 11 : DashboardLanePresentation.laneTitleSize
                    ))
                    .lineLimit(compact ? 2 : 1)
                Spacer()
                if let pullRequest = lane.pullRequest {
                    Text("PR #\(pullRequest.number)").font(BeaconTypography.medium(10)).foregroundStyle(accent)
                } else if let issue = lane.issue {
                    Text("Issue #\(issue.number)").font(BeaconTypography.medium(10)).foregroundStyle(accent)
                } else if !lane.branch.isEmpty {
                    Text(lane.branch).font(BeaconTypography.medium(9)).foregroundStyle(accent).lineLimit(1)
                }
            }
            Text(actionLabel(lane.nextAction))
                .font(BeaconTypography.medium(10))
                .foregroundStyle(accent)
            if let attention = lane.attention {
                Text("\(attention.delta) · \(timeSinceActivity(lane.updatedAt))")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconPalette.cyan)
                    .lineLimit(1)
                tagChips(lane, tags: attention.tags ?? [], accent: accent)
                if let note = attention.note, !note.isEmpty {
                    Label("\(note)\(attention.noteStale ? " · evidence changed" : "")", systemImage: "note.text")
                        .font(BeaconTypography.regular(9))
                        .foregroundStyle(attention.noteStale ? BeaconPalette.gold : BeaconPalette.lavender)
                        .lineLimit(2)
                }
            }
            if !compact, let progress = lane.progress {
                Text("Kit \(actionLabel(progress.phase)) · \(progress.summary)")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.85))
                    .lineLimit(1)
            }
            evidenceBadges(lane)
        }
        .padding(compact ? 9 : 10)
        .background(BeaconPalette.softGradient(accent), in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9)
                .strokeBorder(BeaconPalette.borderGradient(accent), lineWidth: 0.8)
        }
        .shadow(color: accent.opacity(0.09), radius: 4, y: 2)
    }

    func projectGlyph(_ lane: WorkLane, accent: Color) -> some View {
        Text(String((lane.repository.isEmpty ? "?" : lane.repository).prefix(1)).uppercased())
            .font(BeaconTypography.bold(10))
            .foregroundStyle(accent)
            .frame(width: 24, height: 24)
            .background(BeaconPalette.softGradient(accent), in: RoundedRectangle(cornerRadius: 7))
            .overlay {
                RoundedRectangle(cornerRadius: 7).strokeBorder(accent.opacity(0.45), lineWidth: 0.7)
            }
    }

    @ViewBuilder
    func tagChips(_ lane: WorkLane, tags: [String], accent: Color) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 5) {
                ForEach(tags, id: \.self) { tag in
                    HStack(spacing: 3) {
                        Text("#\(tag)")
                            .font(BeaconTypography.medium(9))
                        Button {
                            Task { await state.removeLaneTag(lane, tag: tag) }
                        } label: {
                            Image(systemName: "xmark")
                                .font(.system(size: 7, weight: .bold))
                        }
                        .buttonStyle(.plain)
                        .help("Remove \(tag)")
                    }
                    .foregroundStyle(BeaconPalette.lavender)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 3)
                    .background(BeaconPalette.softGradient(BeaconPalette.lavender), in: Capsule())
                    .overlay { Capsule().strokeBorder(BeaconPalette.lavender.opacity(0.38), lineWidth: 0.6) }
                }
                Button {
                    beginAddingTag(to: lane)
                } label: {
                    Label("Tag", systemImage: "plus")
                        .font(BeaconTypography.medium(9))
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3)
                }
                .buttonStyle(.plain)
                .foregroundStyle(accent)
                .background(BeaconPalette.softGradient(accent), in: Capsule())
                .overlay { Capsule().strokeBorder(accent.opacity(0.35), lineWidth: 0.6) }
            }
        }
    }

    @ViewBuilder
    func laneActions(_ lane: WorkLane) -> some View {
        if lane.attention?.state == "parked" {
            Button("Resume") { Task { await state.setLaneAttention(lane, state: "active") } }
        } else {
            Button("Park") { Task { await state.setLaneAttention(lane, state: "parked") } }
        }
        Button(lane.attention?.pinned == true ? "Unpin" : "Pin") {
            Task { await state.setLanePinned(lane, pinned: lane.attention?.pinned != true) }
        }
        Button("Add Tag") { beginAddingTag(to: lane) }
        if let tags = lane.attention?.tags, !tags.isEmpty {
            Menu("Remove Tag") {
                ForEach(tags, id: \.self) { tag in
                    Button(tag) { Task { await state.removeLaneTag(lane, tag: tag) } }
                }
            }
        }
        if lane.attention?.note?.isEmpty == false {
            Button("Clear Legacy Note") { Task { await state.setLaneNote(lane, note: "") } }
        }
        Button("Mark Seen") { Task { await state.markLaneSeen(lane) } }
    }

    func beginAddingTag(to lane: WorkLane) {
        tagLane = lane
        tagText = ""
        showingTagEditor = true
    }
}
