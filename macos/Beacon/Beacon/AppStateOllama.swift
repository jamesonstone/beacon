import Foundation

extension AppState {
    @discardableResult
    func presentNotesAssistant(
        _ mode: NotesAssistantMode,
        selection: String,
        note: String
    ) -> Bool {
        guard notesAssistantMode != mode else { return false }
        let shouldPrepare = NotesAssistantPresentation.shouldPrepareSession(currentMode: notesAssistantMode)
        if shouldPrepare {
            prepareNotesAssistant(selection: selection, note: note)
        }
        notesAssistantMode = mode
        return shouldPrepare
    }

    func prepareNotesAssistant(selection: String, note: String) {
        notesAssistantRequestGeneration += 1
        isSendingOllamaPrompt = false
        let context = NotesAssistantPresentation.context(selection: selection, note: note)
        notesAssistantAttachment = context?.text ?? ""
        notesAssistantContextSource = context?.source
        notesAssistantPrompt = ""
        notesAssistantMessages = []
        ollamaError = nil
        ollamaNotice = nil
    }

    func removeNotesAssistantContext() {
        notesAssistantAttachment = ""
        notesAssistantContextSource = nil
    }

    func dismissNotesAssistant() {
        notesAssistantRequestGeneration += 1
        notesAssistantMode = nil
        notesAssistantAttachment = ""
        notesAssistantContextSource = nil
        notesAssistantPrompt = ""
        notesAssistantMessages = []
        ollamaError = nil
        ollamaNotice = nil
        isSendingOllamaPrompt = false
    }

    func refreshOllamaModels() async {
        guard !isLoadingOllamaModels else { return }
        isLoadingOllamaModels = true
        defer { isLoadingOllamaModels = false }
        do {
            let status = try await ollamaClient.ollamaStatus()
            ollamaModels = status.models
            ollamaConfiguredModel = status.configuredModel
            ollamaSelectedModel = NotesAssistantPresentation.resolvedModel(
                configured: status.configuredModel,
                models: status.models
            )
            if status.models.isEmpty {
                ollamaError = "No local Ollama models are installed."
            } else {
                ollamaError = nil
            }
            if !status.configuredModel.isEmpty,
               !status.models.contains(where: { $0.name == status.configuredModel }),
               let fallback = status.models.first?.name {
                ollamaNotice = "Configured model \(status.configuredModel) is unavailable; using \(fallback)."
            } else {
                ollamaNotice = nil
            }
        } catch {
            ollamaModels = []
            ollamaSelectedModel = ""
            ollamaNotice = nil
            ollamaError = error.localizedDescription
        }
    }

    func setOllamaDefaultModel(_ model: String) async {
        guard ollamaModels.contains(where: { $0.name == model }) else {
            ollamaError = "The selected Ollama model is not installed locally."
            return
        }
        guard !isSavingOllamaDefault else { return }
        isSavingOllamaDefault = true
        defer { isSavingOllamaDefault = false }
        do {
            ollamaConfiguredModel = try await ollamaClient.setOllamaDefaultModel(model)
            ollamaSelectedModel = model
            ollamaNotice = nil
            ollamaError = nil
        } catch {
            ollamaError = error.localizedDescription
        }
    }

    func sendNotesAssistantPrompt() async {
        let model = ollamaSelectedModel
        let context = notesAssistantAttachment
        let prompt = notesAssistantPrompt.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !prompt.isEmpty,
              ollamaModels.contains(where: { $0.name == model }),
              !isSendingOllamaPrompt else { return }
        notesAssistantRequestGeneration += 1
        let requestGeneration = notesAssistantRequestGeneration
        let userMessage = NotesAssistantMessage(role: .user, content: prompt)
        notesAssistantMessages.append(userMessage)
        let requestMessages = notesAssistantMessages.map(\.chatMessage)
        notesAssistantPrompt = ""
        isSendingOllamaPrompt = true
        ollamaError = nil
        defer {
            if notesAssistantRequestGeneration == requestGeneration {
                isSendingOllamaPrompt = false
            }
        }
        do {
            let response = try await ollamaClient.ollamaChat(
                model: model,
                context: context,
                messages: requestMessages
            )
            guard notesAssistantRequestGeneration == requestGeneration else { return }
            notesAssistantMessages.append(NotesAssistantMessage(
                role: .assistant,
                content: response.content,
                model: response.model
            ))
        } catch {
            guard notesAssistantRequestGeneration == requestGeneration else { return }
            notesAssistantMessages.removeAll { $0.id == userMessage.id }
            notesAssistantPrompt = notesAssistantPrompt.isEmpty
                ? prompt
                : prompt + "\n\n" + notesAssistantPrompt
            ollamaError = error.localizedDescription
        }
    }

    var canSendNotesAssistantPrompt: Bool {
        !notesAssistantPrompt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
            ollamaModels.contains(where: { $0.name == ollamaSelectedModel }) &&
            !isSendingOllamaPrompt
    }
}
