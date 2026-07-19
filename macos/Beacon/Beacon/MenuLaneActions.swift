import SwiftUI

extension MenuView {
    @ViewBuilder
    func tagChips(_ lane: WorkLane, tags: [String], accent: Color) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 5) {
                ForEach(tags, id: \.self) { tag in
                    HStack(spacing: 3) {
                        Text("#\(tag)")
                            .font(BeaconTypography.identifier(9, weight: .medium))
                        Button {
                            Task { await state.removeLaneTag(lane, tag: tag) }
                        } label: {
                            Image(systemName: "xmark")
                                .font(.system(size: 7, weight: .bold))
                        }
                        .buttonStyle(.plain)
                        .help("Remove \(tag)")
                    }
                    .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 3)
                    .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: Capsule())
                    .overlay { Capsule().strokeBorder(BeaconThemePreference.current().tokens.textSecondary.color.opacity(0.38), lineWidth: 0.6) }
                }
                Button {
                    beginAddingTag(to: lane)
                } label: {
                    Image(systemName: "plus")
                        .font(.system(size: 9, weight: .bold))
                        .frame(width: 20, height: 18)
                }
                .buttonStyle(.plain)
                .foregroundStyle(accent)
                .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: Capsule())
                .overlay { Capsule().strokeBorder(accent.opacity(0.35), lineWidth: 0.6) }
                .help("Add Tag")
                .accessibilityLabel("Add Tag")
            }
        }
    }

    @ViewBuilder
    func laneActions(_ lane: WorkLane) -> some View {
        Button("Move Up") { Task { await state.moveLane(lane.id, by: -1) } }
        Button("Move Down") { Task { await state.moveLane(lane.id, by: 1) } }
        Divider()
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
