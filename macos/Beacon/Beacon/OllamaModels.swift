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

enum OllamaChatRole: String, Codable, Equatable {
    case user
    case assistant
}

struct OllamaChatMessage: Codable, Equatable {
    let role: OllamaChatRole
    let content: String
}

struct NotesAssistantMessage: Identifiable, Equatable {
    let id: UUID
    let role: OllamaChatRole
    let content: String
    let model: String?

    init(
        id: UUID = UUID(),
        role: OllamaChatRole,
        content: String,
        model: String? = nil
    ) {
        self.id = id
        self.role = role
        self.content = content
        self.model = model
    }

    var chatMessage: OllamaChatMessage {
        OllamaChatMessage(role: role, content: content)
    }
}

enum NotesAssistantRequestHistory {
    static let maxMessageCount = 128
    static let maxByteCount = 2 * 1024 * 1024
    static let maxUserMessageByteCount = 16 * 1024
    static let truncationNotice =
        "Older turns remain visible but were omitted from this request to stay within local limits."

    static func boundedMessages(from transcript: [NotesAssistantMessage]) -> [OllamaChatMessage] {
        guard !transcript.isEmpty else { return [] }

        var startIndex = 0
        // A valid request starts and ends with a user turn, so its count is odd.
        let maximumOddMessageCount = maxMessageCount - 1
        while transcript.count - startIndex > maximumOddMessageCount {
            startIndex += 2
        }

        var byteCount = transcript[startIndex...].reduce(0) { partialResult, message in
            partialResult + message.content.utf8.count
        }
        while byteCount > maxByteCount, startIndex + 2 < transcript.count {
            byteCount -= transcript[startIndex].content.utf8.count
            byteCount -= transcript[startIndex + 1].content.utf8.count
            startIndex += 2
        }

        return transcript[startIndex...].map(\.chatMessage)
    }
}

protocol OllamaClientProtocol {
    func ollamaStatus() async throws -> OllamaStatus
    func setOllamaDefaultModel(_ model: String) async throws -> String
    func ollamaChat(
        model: String,
        context: String,
        messages: [OllamaChatMessage]
    ) async throws -> OllamaChatResponse
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

enum NotesAssistantMode: Equatable {
    case compact
    case conversation

    var title: String {
        switch self {
        case .compact: "Notes AI"
        case .conversation: "Beacon AI Conversation"
        }
    }
}

enum NotesAssistantConversationAction: Equatable {
    case show
    case dismiss
}

enum NotesAssistantPresentation {
    static let buttonSymbol = "brain.head.profile"
    static let buttonAnimationDuration: TimeInterval = 5.6
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

    static func conversationPanelSize(in available: CGSize, surface _: DashboardSurface) -> CGSize {
        CGSize(width: available.width / 2, height: available.height)
    }

    static func conversationToggleAction(
        currentMode: NotesAssistantMode?
    ) -> NotesAssistantConversationAction {
        currentMode == .conversation ? .dismiss : .show
    }

    static func shouldPrepareSession(currentMode: NotesAssistantMode?) -> Bool {
        currentMode == nil
    }
}
