import AppKit
import Foundation
import SwiftUI

extension MenuView {
    var signalNotesPanel: some View {
        VStack(alignment: .leading, spacing: 7) {
            signalNotesHeader

            if signalNotesExpanded {
                SignalNoteTabStrip(state: state, onDeleteNote: requestNoteDeletion)

                if state.activeNoteID == "new" {
                    SignalNotePicker(state: state, onDeleteNote: requestNoteDeletion)
                } else {
                    liveMarkdownEditor
                }

                HStack(spacing: 8) {
                    if let error = state.notesError {
                        Label(error, systemImage: "exclamationmark.triangle.fill")
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconThemePreference.current().tokens.danger.color)
                            .lineLimit(1)
                    } else if state.notesAreDirty {
                        Label(
                            state.isSavingNotes ? "Saving…" : "Autosaves after 3 seconds",
                            systemImage: state.isSavingNotes ? "arrow.triangle.2.circlepath" : "clock.badge.checkmark"
                        )
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                    } else if let updatedAt = state.notesUpdatedAt {
                        Label(
                            SignalNotesPresentation.savedLabel(age: timeSinceActivity(updatedAt)),
                            systemImage: "checkmark.circle.fill"
                        )
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconThemePreference.current().tokens.success.color)
                    } else {
                        Text("Markdown · local only")
                            .font(BeaconTypography.regular(8))
                            .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                    }
                    Spacer()

                    Button("Revert") {
                        state.revertNotes()
                    }
                    .buttonStyle(.plain)
                    .font(BeaconTypography.medium(9))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
                    .disabled(!state.notesAreDirty || state.isSavingNotes)

                    Button {
                        notesEditorFocused = false
                        Task { await state.saveNotes(state.notesDraft) }
                    } label: {
                        Label("Save", systemImage: "sparkles")
                            .font(BeaconTypography.semibold(9))
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(BeaconThemePreference.current().tokens.info.color.opacity(0.72))
                    .disabled(!state.notesAreDirty || state.isSavingNotes)
                    .keyboardShortcut("s", modifiers: .command)
                }
            }
        }
        .padding(.horizontal, 9)
        .padding(.vertical, 7)
        .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 10))
        .overlay {
            RoundedRectangle(cornerRadius: 10)
                .strokeBorder(interfaceBorderColor, lineWidth: colorSchemeContrast == .increased ? 1.1 : 0.7)
        }
        .overlay {
            GeometryReader { proxy in
                notesAssistantOverlay(in: proxy.size)
            }
        }
    }

    private var signalNotesHeader: some View {
        HStack(spacing: 7) {
            NotesSolarSystemMark()
            VStack(alignment: .leading, spacing: 1) {
                Text("Notes")
                    .font(BeaconTypography.semibold(11))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textPrimary.color)
                if !signalNotesExpanded {
                    Text(notesPreview)
                        .font(BeaconTypography.regular(8))
                        .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                        .lineLimit(1)
                }
            }
            Spacer()
            if state.isSavingNotes {
                ProgressView()
                    .controlSize(.mini)
                    .tint(BeaconThemePreference.current().tokens.info.color)
            }
            if signalNotesExpanded {
                notesAssistantHeaderButton
            }
            if viewMode.locksNotesAtHalfHeight {
                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 9, weight: .semibold))
                    .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                    .frame(width: 20, height: 20)
                    .help("Notes stay at half height in Fit Following")
                    .accessibilityLabel("Notes fixed at half height")
            } else {
                Button {
                    withAnimation(reduceMotion ? nil : .easeInOut(duration: 0.18)) {
                        toggleSignalNotesSize()
                    }
                } label: {
                    Image(systemName: "chevron.up.chevron.down")
                        .font(.system(size: 9, weight: .semibold))
                        .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                        .rotationEffect(signalNotesExpanded ? .degrees(180) : .zero)
                        .frame(width: 20, height: 20)
                }
                .buttonStyle(.plain)
                .help(signalNotesExpanded ? "Minimize Notes" : "Restore Notes")
                .accessibilityLabel(signalNotesExpanded ? "Minimize Notes" : "Restore Notes")
            }
        }
        .contentShape(Rectangle())
        .onTapGesture(count: 2) {
            withAnimation(reduceMotion ? nil : .easeInOut(duration: 0.22)) {
                cycleSignalNotesSize()
            }
        }
        .accessibilityAction(named: "Cycle Notes size") {
            cycleSignalNotesSize()
        }
    }

    var signalNotesSize: SignalNotesSize {
        SignalNotesSize(rawValue: signalNotesSizeValue) ?? .half
    }

    var signalNotesExpanded: Bool { signalNotesSize.isExpanded }

    func signalNotesHeight(in availableHeight: CGFloat) -> CGFloat? {
        DashboardViewModePresentation.notesHeight(
            in: availableHeight,
            mode: viewMode,
            size: signalNotesSize
        )
    }

    func toggleSignalNotesSize() {
        guard !viewMode.locksNotesAtHalfHeight else { return }
        if signalNotesSize == .minimized {
            let restored = SignalNotesSize(rawValue: signalNotesLastExpandedSizeValue) ?? .half
            setSignalNotesSize(restored == .minimized ? .half : restored)
        } else {
            setSignalNotesSize(.minimized)
        }
    }

    func cycleSignalNotesSize() {
        guard !viewMode.locksNotesAtHalfHeight else { return }
        setSignalNotesSize(signalNotesSize.nextCycled)
    }

    func setSignalNotesSize(_ size: SignalNotesSize) {
        signalNotesSizeValue = size.rawValue
        if size == .minimized {
            closeNotesAssistant()
        }
        if size.isExpanded {
            signalNotesLastExpandedSizeValue = size.rawValue
        }
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
            selectedText: Binding(
                get: { state.notesSelectedText },
                set: { state.notesSelectedText = $0 }
            ),
            accessibilityLabel: "Live Markdown notes"
        )
        .padding(8)
        .frame(minHeight: surface == .menu ? 120 : 180, maxHeight: .infinity)
        .background(BeaconThemePreference.current().tokens.surfaceOverlay.color, in: RoundedRectangle(cornerRadius: 8))
        .overlay {
            RoundedRectangle(cornerRadius: 8)
                .strokeBorder(interfaceBorderColor, lineWidth: colorSchemeContrast == .increased ? 1.1 : 0.7)
        }
        .contextMenu {
            if state.activeNoteID == "general", !state.notesCurrentLine.isEmpty {
                Button {
                    Task { await state.createNoteFromCurrentLine() }
                } label: {
                    Label(
                        SignalNotesPresentation.createFromGeneralLabel,
                        systemImage: SignalNotesPresentation.createFromGeneralSymbol
                    )
                }
            }
            Button("New Detail Note") {
                Task { await state.showNewNotePicker() }
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: NSTextView.didChangeSelectionNotification)) { notification in
            guard let textView = notification.object as? NSTextView,
                  textView.accessibilityLabel() == "Live Markdown notes" else { return }
            state.notesSelectedText = LiveMarkdownSelection.text(
                in: textView.string,
                range: textView.selectedRange()
            )
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
