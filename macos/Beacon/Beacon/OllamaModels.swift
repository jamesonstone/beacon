import Foundation

struct OllamaModel: Codable, Hashable, Identifiable {
    let name: String
    let size: Int64
    let details: OllamaModelDetails

    var id: String { name }
}

struct OllamaModelDetails: Codable, Hashable {
    let format: String
    let family: String?
    let parameterSize: String?
    let quantizationLevel: String?

    enum CodingKeys: String, CodingKey {
        case format
        case family
        case parameterSize = "parameter_size"
        case quantizationLevel = "quantization_level"
    }
}

struct OllamaStatus: Codable, Equatable {
    let models: [OllamaModel]
    let configuredModel: String

    enum CodingKeys: String, CodingKey {
        case models
        case configuredModel = "configured_model"
    }
}

struct OllamaChatResponse: Codable, Equatable {
    let model: String
    let content: String
}

protocol OllamaClientProtocol {
    func ollamaStatus() async throws -> OllamaStatus
    func setOllamaDefaultModel(_ model: String) async throws -> String
    func ollamaChat(model: String, context: String, prompt: String) async throws -> OllamaChatResponse
}

enum NotesAssistantContextSource: Equatable {
    case selection
    case note

    var title: String {
        switch self {
        case .selection: "Notes selection"
        case .note: "Entire current note"
        }
    }
}

struct NotesAssistantContext: Equatable {
    let text: String
    let source: NotesAssistantContextSource
}

enum NotesAssistantPresentation {
    static let buttonLabel = "AI"
    static let buttonAccessibilityLabel = "Ask AI About Current Note"
    static let panelAccessibilityLabel = "Ollama Notes assistant"
    static let quickSwitcherTitle = "Ask AI About Current Note"
    static let quickSwitcherKeywords = "ai ollama assistant chat prompt signal notes"

    static func hasUsableSelection(_ selection: String) -> Bool {
        !selection.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    static func resolvedModel(configured: String, models: [OllamaModel]) -> String {
        if models.contains(where: { $0.name == configured }) {
            return configured
        }
        return models.first?.name ?? ""
    }

    static func context(selection: String, note: String) -> NotesAssistantContext? {
        if hasUsableSelection(selection) {
            return NotesAssistantContext(text: selection, source: .selection)
        }
        if hasUsableSelection(note) {
            return NotesAssistantContext(text: note, source: .note)
        }
        return nil
    }

    static func panelSize(in available: CGSize, surface: DashboardSurface) -> CGSize {
        let maximumWidth: CGFloat = surface == .menu ? 350 : 410
        let width = max(220, min(maximumWidth, available.width - 24))
        let height = max(160, min(surface == .menu ? 250 : 320, available.height - 40))
        return CGSize(width: width, height: height)
    }
}
