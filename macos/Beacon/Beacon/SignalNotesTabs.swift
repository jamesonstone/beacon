import SwiftUI

struct SignalNoteTabStrip: View {
    @ObservedObject var state: AppState

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 5) {
                ForEach(state.openNoteTabs) { tab in
                    SignalNoteTabButton(
                        tab: tab,
                        title: tab.id == state.activeNoteID ? state.activeNoteTitle : tab.title,
                        selected: tab.id == state.activeNoteID,
                        onSelect: { Task { await state.activateNote(tab.id) } },
                        onClose: tab.id == "general" ? nil : { Task { await state.closeNote(tab.id) } }
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
                .accessibilityLabel("New Signal Note tab")
            }
        }
        .accessibilityLabel("Open Signal Note tabs")
    }
}

private struct SignalNoteTabButton: View {
    let tab: AgentNoteTab
    let title: String
    let selected: Bool
    let onSelect: () -> Void
    let onClose: (() -> Void)?
    @State private var hovering = false
    @FocusState private var focused: Bool

    var body: some View {
        HStack(spacing: 3) {
            Button(action: onSelect) {
                HStack(spacing: 4) {
                    if tab.id == "general" {
                        Image(systemName: "pin.fill")
                            .font(.system(size: 7, weight: .semibold))
                    }
                    Text(title)
                        .font(BeaconTypography.medium(8))
                        .lineLimit(1)
                        .frame(maxWidth: 132)
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .focused($focused)

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
}

struct SignalNotePicker: View {
    @ObservedObject var state: AppState
    @State private var title = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 6) {
                TextField("First-line title", text: $title)
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
                    Label("Create from current General line", systemImage: "text.line.first.and.arrowtriangle.forward")
                        .lineLimit(1)
                }
                .buttonStyle(.plain)
                .font(BeaconTypography.medium(9))
                .foregroundStyle(BeaconPalette.mint)
                .help(state.notesCurrentLine)
            }

            if state.noteHistory.isEmpty {
                ContentUnavailableView(
                    "No detail notes yet",
                    systemImage: "doc.badge.plus",
                    description: Text("Create one above or from a line in General.")
                )
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                Text("Previously opened")
                    .font(BeaconTypography.semibold(8))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.76))
                ScrollView {
                    LazyVStack(spacing: 4) {
                        ForEach(state.noteHistory) { tab in
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
                                .padding(7)
                                .contentShape(Rectangle())
                                .background(Color.black.opacity(0.14), in: RoundedRectangle(cornerRadius: 6))
                            }
                            .buttonStyle(.plain)
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
