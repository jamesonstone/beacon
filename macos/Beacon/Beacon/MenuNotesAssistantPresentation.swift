import SwiftUI

extension MenuView {
    @ViewBuilder
    func notesAssistantOverlay(in size: CGSize) -> some View {
        if state.notesAssistantMode == .compact, signalNotesExpanded {
            let panelSize = NotesAssistantPresentation.panelSize(in: size, surface: surface)
            HStack {
                Spacer(minLength: 0)
                NotesAssistantPanel(state: state, mode: .compact) {
                    closeNotesAssistant()
                }
                .frame(width: panelSize.width, height: panelSize.height)
                .padding(.trailing, 28)
            }
            .padding(.top, 32)
        }
    }

    @ViewBuilder
    func notesAssistantConversationOverlay(in size: CGSize) -> some View {
        if state.notesAssistantMode == .conversation {
            let panelSize = NotesAssistantPresentation.conversationPanelSize(in: size, surface: surface)
            HStack {
                Spacer(minLength: 0)
                NotesAssistantPanel(state: state, mode: .conversation) {
                    closeNotesAssistant()
                }
                .frame(width: panelSize.width, height: panelSize.height)
                .padding(12)
            }
            .transition(reduceMotion ? .opacity : .move(edge: .trailing).combined(with: .opacity))
            .zIndex(20)
        }
    }

    var notesAssistantHeaderButton: some View {
        Button {
            if state.notesAssistantMode != nil {
                closeNotesAssistant()
            } else {
                showNotesAssistant(.compact)
            }
        } label: {
            HStack(spacing: 5) {
                BeaconAIMark()
                Text(NotesAssistantPresentation.buttonLabel)
                    .font(BeaconTypography.semibold(9))
            }
            .padding(.horizontal, 7)
            .frame(height: 24)
        }
        .buttonStyle(.borderedProminent)
        .tint(theme.tokens.info.color.opacity(0.78))
        .help(NotesAssistantPresentation.buttonAccessibilityLabel)
        .accessibilityLabel(NotesAssistantPresentation.buttonAccessibilityLabel)
    }

    var notesAssistantCommandDetail: String {
        switch NotesAssistantPresentation.context(
            selection: state.notesSelectedText,
            note: state.notesDraft
        )?.source {
        case .selection: "Attach the current Notes selection"
        case .note: "Attach the entire current note"
        case nil: "Start without note context"
        }
    }

    func showNotesAssistant(_ mode: NotesAssistantMode) {
        if state.notesAssistantMode == mode { return }
        if !signalNotesExpanded {
            let restored = SignalNotesSize(rawValue: signalNotesLastExpandedSizeValue) ?? .half
            setSignalNotesSize(restored == .minimized ? .half : restored)
        }
        var shouldRefresh = false
        withAnimation(reduceMotion ? nil : .easeInOut(duration: 0.18)) {
            shouldRefresh = state.presentNotesAssistant(
                mode,
                selection: state.notesSelectedText,
                note: state.notesDraft
            )
        }
        if shouldRefresh {
            Task { await state.refreshOllamaModels() }
        }
    }

    func closeNotesAssistant() {
        withAnimation(reduceMotion ? nil : .easeInOut(duration: 0.18)) {
            state.dismissNotesAssistant()
        }
    }
}
