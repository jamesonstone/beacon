import Foundation

protocol CLIClientProtocol {
    func scan() async throws -> BeaconSnapshot
    func setProjectTracked(_ github: String, tracked: Bool) async throws
    func notes() async throws -> AgentNotes
    func setNotes(_ content: String) async throws -> AgentNotes
    func notesWorkspace() async throws -> AgentNotesWorkspace
    func notes(noteID: String) async throws -> AgentNotes
    func setNotes(_ content: String, noteID: String) async throws -> AgentNotesWorkspace
    func createNote(_ content: String) async throws -> AgentNotesWorkspace
    func openNote(_ noteID: String) async throws -> AgentNotesWorkspace
    func closeNote(_ noteID: String) async throws -> AgentNotesWorkspace
    func deleteNote(_ noteID: String) async throws -> AgentNotesWorkspace
    func repositorySync(refresh: Bool) async throws -> RepositorySyncReport
    func syncRepositories(_ projectIDs: [String]) async throws -> RepositorySyncReport
    func dependencyLimits() async throws -> DependencyLimitReport
    func externalActivity() async throws -> ExternalActivitySnapshot
    func pruneExternalActivity() async throws -> ExternalActivitySnapshot
    func integrationStatus(_ provider: String) async throws -> IntegrationHealthStatus
}

extension CLIClientProtocol {
    func notes() async throws -> AgentNotes { throw AgentClientError.command("signal notes are unavailable") }
    func setNotes(_ content: String) async throws -> AgentNotes { throw AgentClientError.command("signal notes are unavailable") }
    func notesWorkspace() async throws -> AgentNotesWorkspace { throw AgentClientError.command("signal note tabs are unavailable") }
    func notes(noteID: String) async throws -> AgentNotes {
        guard noteID == "general" else { throw AgentClientError.command("signal note tabs are unavailable") }
        return try await notes()
    }
    func setNotes(_ content: String, noteID: String) async throws -> AgentNotesWorkspace {
        guard noteID == "general" else { throw AgentClientError.command("signal note tabs are unavailable") }
        let note = try await setNotes(content)
        return AgentNotesWorkspace(version: 1, activeID: "general", openIDs: ["general"], tabs: [], active: note)
    }
    func createNote(_ content: String) async throws -> AgentNotesWorkspace { throw AgentClientError.command("signal note tabs are unavailable") }
    func openNote(_ noteID: String) async throws -> AgentNotesWorkspace { throw AgentClientError.command("signal note tabs are unavailable") }
    func closeNote(_ noteID: String) async throws -> AgentNotesWorkspace { throw AgentClientError.command("signal note tabs are unavailable") }
    func deleteNote(_ noteID: String) async throws -> AgentNotesWorkspace { throw AgentClientError.command("signal note deletion is unavailable") }
    func repositorySync(refresh: Bool) async throws -> RepositorySyncReport { throw AgentClientError.command("repository sync is unavailable") }
    func syncRepositories(_ projectIDs: [String]) async throws -> RepositorySyncReport { throw AgentClientError.command("repository sync is unavailable") }
    func dependencyLimits() async throws -> DependencyLimitReport { throw AgentClientError.command("dependency limits are unavailable") }
    func externalActivity() async throws -> ExternalActivitySnapshot { throw AgentClientError.command("external activity is unavailable") }
    func pruneExternalActivity() async throws -> ExternalActivitySnapshot { throw AgentClientError.command("external activity pruning is unavailable") }
    func integrationStatus(_ provider: String) async throws -> IntegrationHealthStatus { throw AgentClientError.command("integration health is unavailable") }
}

protocol AgentInstallerProtocol {
    func installAgent() async throws
}

protocol AgentLifecycleControllerProtocol: AgentInstallerProtocol {
    func startAgent() throws
    func stopAgent() throws
}

enum CLIClientError: LocalizedError {
    case helperMissing(String)
    case commandFailed(Int32, String)
    case invalidOutput(String)

    var errorDescription: String? {
        switch self {
        case .helperMissing(let path):
            return "Beacon CLI not found at \(path)"
        case .commandFailed(let status, let message):
            return "Beacon CLI exited with status \(status): \(message)"
        case .invalidOutput(let message):
            return "Beacon CLI returned invalid JSON: \(message)"
        }
    }
}
