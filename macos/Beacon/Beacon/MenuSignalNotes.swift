import SwiftUI

enum SignalNotesPresentation {
    static func savedLabel(age: String) -> String {
        "Saved \(age)"
    }
}

extension MenuView {
    var signalNotesPanel: some View {
        VStack(alignment: .leading, spacing: 7) {
            Button {
                withAnimation(.easeInOut(duration: 0.18)) {
                    signalNotesExpanded.toggle()
                }
            } label: {
                HStack(spacing: 7) {
                    Image(systemName: "pencil.and.scribble")
                        .foregroundStyle(BeaconPalette.neonGradient)
                        .shadow(color: BeaconPalette.pink.opacity(0.45), radius: 2)
                    VStack(alignment: .leading, spacing: 1) {
                        Text("Signal Notes")
                            .font(BeaconTypography.semibold(11))
                            .foregroundStyle(BeaconPalette.borderGradient(BeaconPalette.pink))
                        Text(signalNotesExpanded ? "Catch the sparks before they drift away." : notesPreview)
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconPalette.lavender.opacity(0.78))
                            .lineLimit(1)
                    }
                    Spacer()
                    if state.isSavingNotes {
                        ProgressView()
                            .controlSize(.mini)
                            .tint(BeaconPalette.cyan)
                    }
                    Image(systemName: "chevron.up.chevron.down")
                        .font(.system(size: 9, weight: .semibold))
                        .foregroundStyle(BeaconPalette.cyan)
                        .rotationEffect(signalNotesExpanded ? .degrees(180) : .zero)
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityLabel(signalNotesExpanded ? "Collapse Signal Notes" : "Expand Signal Notes")

            if signalNotesExpanded {
                TextEditor(text: $notesDraft)
                    .focused($notesEditorFocused)
                    .font(BeaconTypography.regular(10))
                    .foregroundStyle(BeaconPalette.mint)
                    .scrollContentBackground(.hidden)
                    .padding(6)
                    .frame(minHeight: 72, maxHeight: surface == .menu ? 92 : 130)
                    .background(Color.black.opacity(0.22), in: RoundedRectangle(cornerRadius: 8))
                    .overlay {
                        RoundedRectangle(cornerRadius: 8)
                            .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.cyan), lineWidth: 0.7)
                    }
                    .accessibilityLabel("Markdown signal notes")

                HStack(spacing: 8) {
                    if let error = state.notesError {
                        Label(error, systemImage: "exclamationmark.triangle.fill")
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconPalette.coral)
                            .lineLimit(1)
                    } else if let updatedAt = state.notesUpdatedAt {
                        Label(
                            SignalNotesPresentation.savedLabel(age: timeSinceActivity(updatedAt)),
                            systemImage: "checkmark.circle.fill"
                        )
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconPalette.mint.opacity(0.82))
                    } else {
                        Text("Markdown · local only")
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconPalette.lavender.opacity(0.72))
                    }
                    Spacer()
                    Button("Revert") {
                        notesDraft = state.notesContent
                    }
                    .buttonStyle(.plain)
                    .font(BeaconTypography.medium(9))
                    .foregroundStyle(BeaconPalette.lavender)
                    .disabled(notesDraft == state.notesContent || state.isSavingNotes)

                    Button {
                        notesEditorFocused = false
                        Task { await state.saveNotes(notesDraft) }
                    } label: {
                        Label("Save", systemImage: "sparkles")
                            .font(BeaconTypography.semibold(9))
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(BeaconPalette.cyan.opacity(0.72))
                    .disabled(notesDraft == state.notesContent || state.isSavingNotes)
                    .keyboardShortcut("s", modifiers: .command)
                }
            }
        }
        .padding(.horizontal, 9)
        .padding(.vertical, 7)
        .background(BeaconPalette.softGradient(BeaconPalette.pink), in: RoundedRectangle(cornerRadius: 10))
        .overlay {
            RoundedRectangle(cornerRadius: 10)
                .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.pink), lineWidth: 0.7)
        }
        .shadow(color: BeaconPalette.pink.opacity(0.10), radius: 5, y: 2)
    }

    private var notesPreview: String {
        let preview = state.notesContent
            .split(whereSeparator: \Character.isNewline)
            .map(String.init)
            .first(where: { !$0.trimmingCharacters(in: .whitespaces).isEmpty })
        return preview ?? "A tiny orbit for ideas in flight."
    }
}
