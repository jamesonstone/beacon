import SwiftUI

struct NotesAssistantPanel: View {
    @Environment(\.beaconTheme) private var theme
    @Environment(\.colorSchemeContrast) private var colorSchemeContrast
    @ObservedObject var state: AppState
    let close: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 7) {
            HStack(spacing: 7) {
                Label("Notes AI", systemImage: "sparkles")
                    .font(BeaconTypography.semibold(10))
                    .foregroundStyle(theme.tokens.accent.color)
                Spacer()
                Button(role: .cancel, action: close) {
                    Label("Cancel", systemImage: "xmark")
                        .font(BeaconTypography.semibold(8))
                }
                .buttonStyle(.bordered)
                .keyboardShortcut(.cancelAction)
                .help("Exit the Notes assistant")
                .accessibilityLabel("Cancel Notes AI")
            }

            contextAttachment

            TextEditor(text: $state.notesAssistantPrompt)
                .font(BeaconTypography.regular(9))
                .foregroundStyle(theme.tokens.editorText.color)
                .scrollContentBackground(.hidden)
                .padding(5)
                .frame(height: 46)
                .background(theme.tokens.editorBackground.color, in: RoundedRectangle(cornerRadius: 7))
                .overlay {
                    RoundedRectangle(cornerRadius: 7)
                        .strokeBorder(theme.tokens.border.color, lineWidth: borderWidth)
                }
                .accessibilityLabel("Prompt for Notes AI")

            if let response = state.notesAssistantResponse {
                ScrollView {
                    VStack(alignment: .leading, spacing: 4) {
                        Label(response.model, systemImage: "sparkles")
                            .font(BeaconTypography.semibold(8))
                            .foregroundStyle(theme.tokens.accent.color)
                        Text(response.content)
                            .font(BeaconTypography.regular(9))
                            .foregroundStyle(theme.tokens.textPrimary.color)
                            .textSelection(.enabled)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
                .frame(maxHeight: .infinity)
                .padding(7)
                .background(theme.tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 7))
            } else if let message = state.ollamaError ?? state.ollamaNotice {
                Label(message, systemImage: state.ollamaError == nil ? "info.circle" : "exclamationmark.triangle.fill")
                    .font(BeaconTypography.regular(8))
                    .foregroundStyle(state.ollamaError == nil ? theme.tokens.info.color : theme.tokens.danger.color)
                    .lineLimit(3)
                    .frame(maxHeight: .infinity, alignment: .topLeading)
            } else {
                Spacer(minLength: 0)
            }

            HStack(spacing: 7) {
                if state.isLoadingOllamaModels {
                    ProgressView()
                        .controlSize(.small)
                        .accessibilityLabel("Loading Ollama models")
                }
                Picker("Model", selection: $state.ollamaSelectedModel) {
                    if state.ollamaModels.isEmpty {
                        Text("No local models").tag("")
                    } else {
                        ForEach(state.ollamaModels) { model in
                            Text(model.name).tag(model.name)
                        }
                    }
                }
                .labelsHidden()
                .pickerStyle(.menu)
                .frame(maxWidth: .infinity, alignment: .leading)
                .disabled(state.isLoadingOllamaModels || state.isSendingOllamaPrompt)
                .accessibilityLabel("Ollama model")

                Button {
                    Task { await state.sendNotesAssistantPrompt() }
                } label: {
                    if state.isSendingOllamaPrompt {
                        ProgressView()
                            .controlSize(.small)
                    } else {
                        Label("Send", systemImage: "arrow.up.circle.fill")
                            .font(BeaconTypography.semibold(9))
                    }
                }
                .buttonStyle(.borderedProminent)
                .tint(theme.tokens.info.color.opacity(0.82))
                .disabled(!state.canSendNotesAssistantPrompt)
                .keyboardShortcut(.return, modifiers: .command)
                .accessibilityLabel(state.isSendingOllamaPrompt ? "Waiting for Ollama" : "Send to Ollama")
            }
        }
        .padding(9)
        .background(theme.tokens.surfaceOverlay.color, in: RoundedRectangle(cornerRadius: 10))
        .overlay {
            RoundedRectangle(cornerRadius: 10)
                .strokeBorder(theme.tokens.borderStrong.color, lineWidth: borderWidth)
        }
        .shadow(color: .black.opacity(0.22), radius: 9, y: 4)
        .accessibilityElement(children: .contain)
        .accessibilityLabel(NotesAssistantPresentation.panelAccessibilityLabel)
    }

    private var borderWidth: CGFloat {
        colorSchemeContrast == .increased ? 1.1 : 0.7
    }

    @ViewBuilder
    private var contextAttachment: some View {
        if let source = state.notesAssistantContextSource {
            HStack(alignment: .top, spacing: 7) {
                VStack(alignment: .leading, spacing: 3) {
                    Label(source.title, systemImage: "paperclip")
                        .font(BeaconTypography.semibold(8))
                        .foregroundStyle(theme.tokens.info.color)
                    ScrollView {
                        Text(state.notesAssistantAttachment)
                            .font(BeaconTypography.regular(9))
                            .foregroundStyle(theme.tokens.textPrimary.color)
                            .textSelection(.enabled)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                    .frame(maxHeight: 44)
                }
                Spacer(minLength: 4)
                Button {
                    state.removeNotesAssistantContext()
                } label: {
                    Label("Remove", systemImage: "xmark.circle.fill")
                        .font(BeaconTypography.medium(8))
                }
                .buttonStyle(.plain)
                .foregroundStyle(theme.tokens.textSecondary.color)
                .disabled(state.isSendingOllamaPrompt)
                .help("Remove Notes context")
                .accessibilityLabel("Remove Notes context")
            }
            .padding(7)
            .background(theme.tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 7))
            .overlay {
                RoundedRectangle(cornerRadius: 7)
                    .strokeBorder(theme.tokens.border.color, lineWidth: borderWidth)
            }
        } else {
            Label("No note context attached", systemImage: "paperclip.badge.ellipsis")
                .font(BeaconTypography.regular(8))
                .foregroundStyle(theme.tokens.textMuted.color)
                .padding(.horizontal, 2)
                .accessibilityLabel("No Notes context attached")
        }
    }
}

extension MenuView {
    @ViewBuilder
    func notesAssistantOverlay(in size: CGSize) -> some View {
        if showingNotesAssistant, signalNotesExpanded {
            let panelSize = NotesAssistantPresentation.panelSize(in: size, surface: surface)
            HStack {
                Spacer(minLength: 0)
                NotesAssistantPanel(state: state) {
                    closeNotesAssistant()
                }
                .frame(width: panelSize.width, height: panelSize.height)
                .padding(.trailing, 28)
            }
            .padding(.top, 32)
        }
    }

    var notesAssistantHeaderButton: some View {
        Button {
            if showingNotesAssistant {
                closeNotesAssistant()
            } else {
                showNotesAssistant()
            }
        } label: {
            Label(
                NotesAssistantPresentation.buttonLabel,
                systemImage: "sparkles"
            )
            .font(BeaconTypography.semibold(9))
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

    func showNotesAssistant() {
        if !signalNotesExpanded {
            let restored = SignalNotesSize(rawValue: signalNotesLastExpandedSizeValue) ?? .half
            setSignalNotesSize(restored == .minimized ? .half : restored)
        }
        showingNotesAssistant = true
        let selection = state.notesSelectedText
        let note = state.notesDraft
        Task { await state.prepareNotesAssistant(selection: selection, note: note) }
    }

    func closeNotesAssistant() {
        showingNotesAssistant = false
        state.dismissNotesAssistant()
    }
}
