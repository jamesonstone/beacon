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
            HStack(spacing: 0) {
                Spacer(minLength: 0)
                NotesAssistantPanel(state: state, mode: .conversation) {
                    closeNotesAssistant()
                }
                .frame(width: panelSize.width, height: panelSize.height)
            }
            .frame(width: size.width, height: size.height)
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
            BeaconAIMark(animated: true)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .frame(
            width: SignalNotesPresentation.headerControlSize,
            height: SignalNotesPresentation.headerControlSize
        )
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

    func toggleNotesAssistantConversation() {
        switch NotesAssistantPresentation.conversationToggleAction(currentMode: state.notesAssistantMode) {
        case .show:
            showNotesAssistant(.conversation)
        case .dismiss:
            closeNotesAssistant()
        }
    }

    func closeNotesAssistant() {
        withAnimation(reduceMotion ? nil : .easeInOut(duration: 0.18)) {
            state.dismissNotesAssistant()
        }
    }
}
