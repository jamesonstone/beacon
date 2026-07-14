import Foundation
import SwiftUI

enum SignalNotesPresentation {
    static let expandedByDefault = true
    static let expandedHeightFraction = 0.5
    static let autosaveDelay: Duration = .seconds(3)

    static func savedLabel(age: String) -> String {
        "Saved \(age)"
    }
}

@MainActor
final class SignalNotesAutosave: ObservableObject {
    private let delay: Duration
    private var pendingTask: Task<Void, Never>?
    private var generation = 0

    init(delay: Duration = SignalNotesPresentation.autosaveDelay) {
        self.delay = delay
    }

    func schedule(
        content: String,
        save: @escaping @MainActor (String) async -> Void
    ) {
        generation += 1
        let scheduledGeneration = generation
        pendingTask?.cancel()
        let delay = delay
        pendingTask = Task { [weak self] in
            do {
                try await Task.sleep(for: delay)
            } catch {
                return
            }
            guard !Task.isCancelled, self?.generation == scheduledGeneration else { return }
            await save(content)
            if self?.generation == scheduledGeneration {
                self?.pendingTask = nil
            }
        }
    }

    func cancel() {
        generation += 1
        pendingTask?.cancel()
        pendingTask = nil
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
                SignalNoteTabStrip(state: state)

                if state.activeNoteID == "new" {
                    SignalNotePicker(state: state)
                } else {
                    liveMarkdownEditor
                }

                HStack(spacing: 8) {
                    if let error = state.notesError {
                        Label(error, systemImage: "exclamationmark.triangle.fill")
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconPalette.coral)
                            .lineLimit(1)
                    } else if state.notesAreDirty {
                        Label(
                            state.isSavingNotes ? "Saving…" : "Autosaves after 3 seconds",
                            systemImage: state.isSavingNotes ? "arrow.triangle.2.circlepath" : "clock.badge.checkmark"
                        )
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconPalette.cyan.opacity(0.82))
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
                        state.revertNotes()
                    }
                    .buttonStyle(.plain)
                    .font(BeaconTypography.medium(9))
                    .foregroundStyle(BeaconPalette.lavender)
                    .disabled(!state.notesAreDirty || state.isSavingNotes)

                    Button {
                        notesEditorFocused = false
                        Task { await state.saveNotes(state.notesDraft) }
                    } label: {
                        Label("Save", systemImage: "sparkles")
                            .font(BeaconTypography.semibold(9))
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(BeaconPalette.cyan.opacity(0.72))
                    .disabled(!state.notesAreDirty || state.isSavingNotes)
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

    private var liveMarkdownEditor: some View {
        LiveMarkdownEditor(
            text: Binding(
                get: { state.notesDraft },
                set: { state.updateNotesDraft($0) }
            ),
            isFocused: $notesEditorFocused,
            currentLine: Binding(
                get: { state.notesCurrentLine },
                set: { state.updateNotesCurrentLine($0) }
            ),
            accessibilityLabel: "Live Markdown signal notes"
        )
        .padding(8)
        .frame(minHeight: surface == .menu ? 120 : 180, maxHeight: .infinity)
        .background(Color.black.opacity(0.22), in: RoundedRectangle(cornerRadius: 8))
        .overlay {
            RoundedRectangle(cornerRadius: 8)
                .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.cyan), lineWidth: 0.7)
        }
        .contextMenu {
            if state.activeNoteID == "general", !state.notesCurrentLine.isEmpty {
                Button("Create Detail From Current Line") {
                    Task { await state.createNoteFromCurrentLine() }
                }
            }
            Button("New Detail Note") {
                Task { await state.showNewNotePicker() }
            }
        }
    }

    private var notesPreview: String {
        let preview = state.notesContent
            .split(whereSeparator: { $0.isNewline })
            .map(String.init)
            .first(where: { !$0.trimmingCharacters(in: .whitespaces).isEmpty })
        return preview ?? "A tiny orbit for ideas in flight."
    }

}
