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
                .foregroundStyle(BeaconPalette.cyan)
                .background(BeaconPalette.softGradient(BeaconPalette.cyan), in: RoundedRectangle(cornerRadius: 6))
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
                .foregroundStyle(BeaconPalette.coral.opacity(0.9))
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
                .foregroundStyle(BeaconPalette.lavender.opacity(0.9))
                .help("Close \(title)")
                .accessibilityLabel("Close \(title) tab")
            }
        }
        .foregroundStyle(selected ? BeaconPalette.mint : BeaconPalette.lavender)
        .padding(.horizontal, 7)
        .frame(height: 24)
        .background(
            selected ? BeaconPalette.softGradient(BeaconPalette.mint) : BeaconPalette.softGradient(BeaconPalette.lavender),
            in: RoundedRectangle(cornerRadius: 6)
        )
        .overlay {
            RoundedRectangle(cornerRadius: 6)
                .strokeBorder((selected ? BeaconPalette.mint : BeaconPalette.lavender).opacity(selected ? 0.62 : 0.24), lineWidth: 0.7)
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
                    .tint(BeaconPalette.cyan.opacity(0.72))
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
                .foregroundStyle(BeaconPalette.mint)
                .help(state.notesCurrentLine)
            }

            if state.noteHistory.isEmpty {
                EmptyNotesSpaceView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                Text("Previously opened")
                    .font(BeaconTypography.semibold(8))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.76))
                ScrollView {
                    LazyVStack(spacing: 4) {
                        ForEach(state.noteHistory) { tab in
                            HStack(spacing: 5) {
                                Button {
                                    Task { await state.activateNote(tab.id) }
                                } label: {
                                    HStack(spacing: 7) {
                                        Image(systemName: tab.isOpen ? "rectangle.on.rectangle" : "doc.text")
                                            .foregroundStyle(tab.isOpen ? BeaconPalette.mint : BeaconPalette.cyan)
                                        VStack(alignment: .leading, spacing: 1) {
                                            Text(tab.title)
                                                .font(BeaconTypography.medium(9))
                                                .foregroundStyle(BeaconPalette.mint)
                                                .lineLimit(1)
                                            Text(tab.id)
                                                .font(BeaconTypography.regular(7))
                                                .foregroundStyle(BeaconPalette.lavender.opacity(0.68))
                                        }
                                        Spacer()
                                        if tab.isOpen {
                                            Text("OPEN")
                                                .font(BeaconTypography.bold(7))
                                                .foregroundStyle(BeaconPalette.mint)
                                        }
                                    }
                                    .contentShape(Rectangle())
                                }
                                .buttonStyle(.plain)
                                .frame(maxWidth: .infinity)

                                Button { onDeleteNote(tab) } label: {
                                    Image(systemName: "trash")
                                        .font(.system(size: 9, weight: .semibold))
                                        .foregroundStyle(BeaconPalette.coral)
                                        .frame(width: 24, height: 24)
                                        .contentShape(Rectangle())
                                }
                                .buttonStyle(.plain)
                                .help("Delete \(tab.title)")
                                .accessibilityLabel("Delete \(tab.title) note")
                            }
                            .padding(7)
                            .background(Color.black.opacity(0.14), in: RoundedRectangle(cornerRadius: 6))
                        }
                    }
                }
            }
        }
        .padding(8)
        .frame(minHeight: 120, maxHeight: .infinity)
        .background(Color.black.opacity(0.22), in: RoundedRectangle(cornerRadius: 8))
        .overlay {
            RoundedRectangle(cornerRadius: 8)
                .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.cyan), lineWidth: 0.7)
        }
    }

    private func create() {
        let candidate = title
        title = ""
        Task { await state.createNote(title: candidate) }
    }
}
