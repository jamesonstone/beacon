import SwiftUI

struct SignalNoteTabStrip: View {
    @ObservedObject var state: AppState
    let onDeleteNote: (AgentNoteTab) -> Void

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 5) {
                ForEach(state.openNoteTabs) { tab in
                    let pinned = state.pinnedNoteIDs.contains(tab.id)
                    SignalNoteTabButton(
                        tab: tab,
                        title: tab.id == state.activeNoteID ? state.activeNoteTitle : tab.title,
                        selected: tab.id == state.activeNoteID,
                        pinned: pinned,
                        onSelect: { Task { await state.activateNote(tab.id) } },
                        onTogglePin: tab.id == "general" || tab.id == "new" ? nil : {
                            Task<Void, Never> { await state.setNotePinned(tab.id, pinned: !pinned) }
                        },
                        onClose: tab.id == "general" || pinned ? nil : { Task { await state.closeNote(tab.id) } },
                        onDelete: tab.id == "general" || tab.id == "new" ? nil : { onDeleteNote(tab) },
                        onMovePinned: { source, target in
                            Task<Void, Never> { await state.movePinnedNote(source, before: target) }
                        }
                    )
                }
                Button {
                    Task { await state.showNewNotePicker() }
                } label: {
                    Image(systemName: "plus")
                        .font(.system(size: 9, weight: .bold))
                        .frame(width: 22, height: 22)
                }
                .buttonStyle(.plain)
                .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 6))
                .help("New Tab")
                .accessibilityLabel("New Note tab")
            }
        }
        .accessibilityLabel("Open Note tabs")
    }
}

private struct SignalNoteTabButton: View {
    let tab: AgentNoteTab
    let title: String
    let selected: Bool
    let pinned: Bool
    let onSelect: () -> Void
    let onTogglePin: (() -> Void)?
    let onClose: (() -> Void)?
    let onDelete: (() -> Void)?
    let onMovePinned: (String, String) -> Void
    @State private var hovering = false
    @FocusState private var focused: Bool

    @ViewBuilder
    var body: some View {
        if pinned, tab.id == "general" {
            tabPill.dropDestination(for: String.self, action: handleDrop)
        } else if pinned {
            tabPill
                .draggable(tab.id)
                .dropDestination(for: String.self, action: handleDrop)
        } else {
            tabPill
        }
    }

    private var tabPill: some View {
        HStack(spacing: 3) {
            if tab.id == "general" {
                Image(systemName: "pin.fill")
                    .font(.system(size: 7, weight: .semibold))
                    .accessibilityLabel("General is permanently pinned")
            } else if let onTogglePin {
                Button(action: onTogglePin) {
                    Image(systemName: pinned ? "pin.fill" : "pin")
                        .font(.system(size: 7, weight: .semibold))
                        .frame(width: 13, height: 13)
                }
                .buttonStyle(.plain)
                .help(pinned ? "Unpin \(title)" : "Pin \(title)")
                .accessibilityLabel(pinned ? "Unpin \(title) note" : "Pin \(title) note")
            }

            Button(action: onSelect) {
                Text(title)
                    .font(BeaconTypography.medium(8))
                    .lineLimit(1)
                    .frame(maxWidth: 132)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .focused($focused)

            if let onDelete, hovering || focused {
                Button(action: onDelete) {
                    Image(systemName: "trash")
                        .font(.system(size: 7, weight: .bold))
                        .frame(width: 13, height: 13)
                }
                .buttonStyle(.plain)
                .foregroundStyle(BeaconThemePreference.current().tokens.danger.color)
                .help("Delete \(title)")
                .accessibilityLabel("Delete \(title) note")
            }

            if let onClose, hovering || focused {
                Button(action: onClose) {
                    Image(systemName: "xmark")
                        .font(.system(size: 7, weight: .bold))
                        .frame(width: 13, height: 13)
                }
                .buttonStyle(.plain)
                .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                .help("Close \(title)")
                .accessibilityLabel("Close \(title) tab")
            }
        }
        .foregroundStyle(selected ? BeaconThemePreference.current().tokens.success.color : BeaconThemePreference.current().tokens.textSecondary.color)
        .padding(.horizontal, 7)
        .frame(height: 24)
        .background(
            selected
                ? BeaconThemePreference.current().tokens.surfaceRaised.color
                : BeaconThemePreference.current().tokens.surface.color,
            in: RoundedRectangle(cornerRadius: 6)
        )
        .overlay {
            RoundedRectangle(cornerRadius: 6)
                .strokeBorder(
                    selected
                        ? BeaconThemePreference.current().tokens.success.color
                        : BeaconThemePreference.current().tokens.border.color,
                    lineWidth: selected ? 1 : 0.7
                )
        }
        .onHover { hovering = $0 }
    }

    private func handleDrop(_ noteIDs: [String], _ location: CGPoint) -> Bool {
        guard let noteID = noteIDs.first, noteID != tab.id else { return false }
        onMovePinned(noteID, tab.id)
        return true
    }
}

struct SignalNotePicker: View {
    @ObservedObject var state: AppState
    let onDeleteNote: (AgentNoteTab) -> Void
    @State private var title = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 6) {
                TextField("title", text: $title)
                    .textFieldStyle(.roundedBorder)
                    .onSubmit { create() }
                Button("Create", action: create)
                    .buttonStyle(.borderedProminent)
                    .tint(BeaconThemePreference.current().tokens.info.color.opacity(0.72))
            }

            if !state.notesCurrentLine.isEmpty {
                Button {
                    Task { await state.createNoteFromCurrentLine() }
                } label: {
                    Label(
                        SignalNotesPresentation.createFromGeneralLabel,
                        systemImage: SignalNotesPresentation.createFromGeneralSymbol
                    )
                        .lineLimit(1)
                }
                .buttonStyle(.plain)
                .font(BeaconTypography.medium(9))
                .foregroundStyle(BeaconThemePreference.current().tokens.success.color)
                .help(state.notesCurrentLine)
            }

            if state.noteHistory.isEmpty {
                EmptyNotesSpaceView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                Text("Previously opened")
                    .font(BeaconTypography.semibold(8))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                ScrollView {
                    LazyVStack(spacing: 4) {
                        ForEach(state.noteHistory) { tab in
                            HStack(spacing: 5) {
                                Button {
                                    Task { await state.activateNote(tab.id) }
                                } label: {
                                    HStack(spacing: 7) {
                                        Image(systemName: tab.isOpen ? "rectangle.on.rectangle" : "doc.text")
                                            .foregroundStyle(tab.isOpen ? BeaconThemePreference.current().tokens.success.color : BeaconThemePreference.current().tokens.info.color)
                                        VStack(alignment: .leading, spacing: 1) {
                                            Text(tab.title)
                                                .font(BeaconTypography.medium(9))
                                                .foregroundStyle(BeaconThemePreference.current().tokens.success.color)
                                                .lineLimit(1)
                                            Text(tab.id)
                                                .font(BeaconTypography.regular(7))
                                                .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                                        }
                                        Spacer()
                                        if tab.isOpen {
                                            Text("OPEN")
                                                .font(BeaconTypography.bold(7))
                                                .foregroundStyle(BeaconThemePreference.current().tokens.success.color)
                                        }
                                    }
                                    .contentShape(Rectangle())
                                }
                                .buttonStyle(.plain)
                                .frame(maxWidth: .infinity)

                                Button { onDeleteNote(tab) } label: {
                                    Image(systemName: "trash")
                                        .font(.system(size: 9, weight: .semibold))
                                        .foregroundStyle(BeaconThemePreference.current().tokens.danger.color)
                                        .frame(width: 24, height: 24)
                                        .contentShape(Rectangle())
                                }
                                .buttonStyle(.plain)
                                .help("Delete \(tab.title)")
                                .accessibilityLabel("Delete \(tab.title) note")
                            }
                            .padding(7)
                            .background(BeaconThemePreference.current().tokens.surfaceOverlay.color, in: RoundedRectangle(cornerRadius: 6))
                        }
                    }
                }
            }
        }
        .padding(8)
        .frame(minHeight: 120, maxHeight: .infinity)
        .background(BeaconThemePreference.current().tokens.surfaceOverlay.color, in: RoundedRectangle(cornerRadius: 8))
        .overlay {
            RoundedRectangle(cornerRadius: 8)
                .strokeBorder(BeaconThemePreference.current().tokens.borderStrong.color, lineWidth: 0.7)
        }
    }

    private func create() {
        let candidate = title
        title = ""
        Task { await state.createNote(title: candidate) }
    }
}
