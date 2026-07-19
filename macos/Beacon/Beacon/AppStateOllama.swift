import Foundation

extension AppState {
    func prepareNotesAssistant(selection: String) async {
        guard NotesAssistantPresentation.hasUsableSelection(selection) else { return }
        notesAssistantAttachment = selection
        notesAssistantPrompt = ""
        notesAssistantResponse = nil
        ollamaError = nil
        await refreshOllamaModels()
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
        let selection = notesAssistantAttachment
        let prompt = notesAssistantPrompt.trimmingCharacters(in: .whitespacesAndNewlines)
        guard NotesAssistantPresentation.hasUsableSelection(selection),
              !prompt.isEmpty,
              ollamaModels.contains(where: { $0.name == model }),
              !isSendingOllamaPrompt else { return }
        isSendingOllamaPrompt = true
        notesAssistantResponse = nil
        ollamaError = nil
        defer { isSendingOllamaPrompt = false }
        do {
            notesAssistantResponse = try await ollamaClient.ollamaChat(
                model: model,
                selection: selection,
                prompt: prompt
            )
        } catch {
            ollamaError = error.localizedDescription
        }
    }

    var canSendNotesAssistantPrompt: Bool {
        NotesAssistantPresentation.hasUsableSelection(notesAssistantAttachment) &&
            !notesAssistantPrompt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
            ollamaModels.contains(where: { $0.name == ollamaSelectedModel }) &&
            !isSendingOllamaPrompt
    }
}
