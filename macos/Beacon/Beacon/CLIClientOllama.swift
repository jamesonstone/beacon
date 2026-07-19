import Foundation

extension CLIClient: OllamaClientProtocol {
    func ollamaStatus() async throws -> OllamaStatus {
        try decodeOllama(
            OllamaStatus.self,
            from: try await execute(arguments: ["ollama", "models", "--json"])
        )
    }

    func setOllamaDefaultModel(_ model: String) async throws -> String {
        struct DefaultResponse: Decodable {
            let configuredModel: String

            enum CodingKeys: String, CodingKey {
                case configuredModel = "configured_model"
            }
        }
        let response = try decodeOllama(
            DefaultResponse.self,
            from: try await execute(arguments: ["ollama", "set-default", model, "--json"])
        )
        return response.configuredModel
    }

    func ollamaChat(
        model: String,
        context: String,
        messages: [OllamaChatMessage]
    ) async throws -> OllamaChatResponse {
        struct ChatInput: Encodable {
            let context: String
            let messages: [OllamaChatMessage]
        }
        let input: Data
        do {
            input = try JSONEncoder().encode(ChatInput(context: context, messages: messages))
        } catch {
            throw CLIClientError.invalidOutput(error.localizedDescription)
        }
        return try decodeOllama(
            OllamaChatResponse.self,
            from: try await execute(
                arguments: ["ollama", "chat", "--model", model, "--json"],
                standardInput: input
            )
        )
    }

    private func decodeOllama<Value: Decodable>(_ type: Value.Type, from data: Data) throws -> Value {
        do {
            return try JSONDecoder().decode(type, from: data)
        } catch {
            throw CLIClientError.invalidOutput(error.localizedDescription)
        }
    }
}
