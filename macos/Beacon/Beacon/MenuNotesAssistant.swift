import SwiftUI

struct NotesAssistantPanel: View {
    @Environment(\.beaconTheme) private var theme
    @Environment(\.colorSchemeContrast) private var colorSchemeContrast
    @ObservedObject var state: AppState
    let mode: NotesAssistantMode
    let close: () -> Void
    @FocusState private var promptFocused: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            header
            conversation
            composer
        }
        .padding(mode == .compact ? 9 : 12)
        .background(theme.tokens.surfaceOverlay.color, in: RoundedRectangle(cornerRadius: 10))
        .overlay {
            RoundedRectangle(cornerRadius: 10)
                .strokeBorder(theme.tokens.borderStrong.color, lineWidth: borderWidth)
        }
        .shadow(color: .black.opacity(0.22), radius: 9, y: 4)
        .accessibilityElement(children: .contain)
        .accessibilityLabel(NotesAssistantPresentation.panelAccessibilityLabel)
        .task {
            await Task.yield()
            promptFocused = true
        }
    }

    private var header: some View {
        HStack(spacing: 7) {
            BeaconAIMark()
            Text(mode.title)
                .font(BeaconTypography.semibold(mode == .compact ? 10 : 12))
                .foregroundStyle(theme.tokens.accent.color)
            Spacer()
            Text(mode == .compact ? "⌘⇧I" : "⌘I")
                .font(BeaconTypography.regular(7))
                .foregroundStyle(theme.tokens.textMuted.color)
            Button(role: .cancel, action: close) {
                Label("Cancel", systemImage: "xmark")
                    .font(BeaconTypography.semibold(8))
            }
            .buttonStyle(.bordered)
            .keyboardShortcut(.cancelAction)
            .help("Exit and reset the Notes assistant")
            .accessibilityLabel("Cancel Notes AI")
        }
    }

    private var conversation: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 8) {
                    contextAttachment

                    if state.notesAssistantMessages.isEmpty,
                       state.ollamaError == nil,
                       state.ollamaNotice == nil {
                        emptyConversation
                    }

                    ForEach(state.notesAssistantMessages) { message in
                        messageRow(message)
                            .id(message.id)
                    }

                    if state.isSendingOllamaPrompt {
                        HStack(spacing: 6) {
                            ProgressView()
                                .controlSize(.small)
                            Text("Thinking locally…")
                                .font(BeaconTypography.regular(8))
                                .foregroundStyle(theme.tokens.textSecondary.color)
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Ollama is preparing a response")
                    }

                    statusMessage
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .onAppear {
                guard let messageID = state.notesAssistantMessages.last?.id else { return }
                proxy.scrollTo(messageID, anchor: .bottom)
            }
            .onChange(of: state.notesAssistantMessages.last?.id) { _, messageID in
                guard let messageID else { return }
                proxy.scrollTo(messageID, anchor: .bottom)
            }
        }
        .padding(7)
        .background(theme.tokens.surface.color, in: RoundedRectangle(cornerRadius: 7))
        .overlay {
            RoundedRectangle(cornerRadius: 7)
                .strokeBorder(theme.tokens.border.color, lineWidth: borderWidth)
        }
    }

    private var emptyConversation: some View {
        VStack(alignment: .leading, spacing: 4) {
            Label("Ready for a conversation", systemImage: "bubble.left.and.bubble.right")
                .font(BeaconTypography.semibold(9))
                .foregroundStyle(theme.tokens.success.color)
            Text("Ask a question below. Every turn stays visible here until you Cancel.")
                .font(BeaconTypography.regular(8))
                .foregroundStyle(theme.tokens.textMuted.color)
        }
        .padding(.vertical, 4)
    }

    @ViewBuilder
    private func messageRow(_ message: NotesAssistantMessage) -> some View {
        HStack(alignment: .top, spacing: 8) {
            if message.role == .user {
                Spacer(minLength: mode == .compact ? 18 : 48)
            }

            VStack(alignment: .leading, spacing: 5) {
                if message.role == .assistant {
                    HStack(spacing: 5) {
                        BeaconAIMark()
                            .scaleEffect(0.78)
                            .frame(width: 15, height: 15)
                        Text(message.model ?? "Local Ollama")
                            .font(BeaconTypography.semibold(8))
                            .foregroundStyle(theme.tokens.accent.color)
                    }
                    BeaconMarkdownDocument(message.content, baseSize: mode == .compact ? 8.5 : 9.5)
                } else {
                    Label("You", systemImage: "person.crop.circle.fill")
                        .font(BeaconTypography.semibold(8))
                        .foregroundStyle(theme.tokens.info.color)
                    Text(message.content)
                        .font(BeaconTypography.regular(mode == .compact ? 8.5 : 9.5))
                        .foregroundStyle(theme.tokens.textPrimary.color)
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .padding(8)
            .background(
                message.role == .user
                    ? theme.tokens.info.color.opacity(0.13)
                    : theme.tokens.surfaceRaised.color,
                in: RoundedRectangle(cornerRadius: 8)
            )
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(
                        message.role == .user ? theme.tokens.info.color.opacity(0.45) : theme.tokens.border.color,
                        lineWidth: borderWidth
                    )
            }

            if message.role == .assistant {
                Spacer(minLength: mode == .compact ? 8 : 36)
            }
        }
        .frame(maxWidth: .infinity)
        .accessibilityElement(children: .combine)
        .accessibilityLabel(message.role == .user ? "You: \(message.content)" : "AI: \(message.content)")
    }

    @ViewBuilder
    private var statusMessage: some View {
        if let message = state.ollamaError ?? state.ollamaNotice {
            Label(message, systemImage: state.ollamaError == nil ? "info.circle" : "exclamationmark.triangle.fill")
                .font(BeaconTypography.regular(8))
                .foregroundStyle(state.ollamaError == nil ? theme.tokens.info.color : theme.tokens.danger.color)
                .fixedSize(horizontal: false, vertical: true)
        }
    }

    private var composer: some View {
        VStack(alignment: .leading, spacing: 7) {
            TextEditor(text: $state.notesAssistantPrompt)
                .font(BeaconTypography.regular(mode == .compact ? 9 : 10))
                .foregroundStyle(theme.tokens.editorText.color)
                .scrollContentBackground(.hidden)
                .focused($promptFocused)
                .padding(5)
                .frame(height: mode == .compact ? 46 : 72)
                .background(theme.tokens.editorBackground.color, in: RoundedRectangle(cornerRadius: 7))
                .overlay {
                    RoundedRectangle(cornerRadius: 7)
                        .strokeBorder(theme.tokens.border.color, lineWidth: borderWidth)
                }
                .accessibilityLabel("Prompt for Notes AI")

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
        .padding(7)
        .background(theme.tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 7))
        .overlay {
            RoundedRectangle(cornerRadius: 7)
                .strokeBorder(theme.tokens.border.color, lineWidth: borderWidth)
        }
    }

    private var borderWidth: CGFloat {
        colorSchemeContrast == .increased ? 1.1 : 0.7
    }

    @ViewBuilder
    private var contextAttachment: some View {
        if let source = state.notesAssistantContextSource {
            HStack(alignment: .top, spacing: 7) {
                VStack(alignment: .leading, spacing: 5) {
                    Label(source.title, systemImage: "paperclip")
                        .font(BeaconTypography.semibold(8))
                        .foregroundStyle(theme.tokens.info.color)
                    Text(state.notesAssistantAttachment)
                        .font(BeaconTypography.regular(mode == .compact ? 8 : 9))
                        .foregroundStyle(theme.tokens.textPrimary.color)
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
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
