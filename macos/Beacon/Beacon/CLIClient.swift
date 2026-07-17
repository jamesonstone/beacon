import Foundation

struct CLIClient: CLIClientProtocol, AgentLifecycleControllerProtocol {
    private let executableURL: URL

    init(executableURL: URL = CLIClient.defaultExecutableURL()) {
        self.executableURL = executableURL
    }

    func scan() async throws -> BeaconSnapshot {
        let outputData = try await execute(arguments: ["scan", "--json"])
        do {
            return try JSONDecoder().decode(BeaconSnapshot.self, from: outputData)
        } catch {
            throw CLIClientError.invalidOutput(error.localizedDescription)
        }
    }

    func setProjectTracked(_ github: String, tracked: Bool) async throws {
        let command = tracked ? "track" : "untrack"
        _ = try await execute(arguments: ["projects", command, github])
    }

    func notes() async throws -> AgentNotes {
        try decodeNotes(try await execute(arguments: ["notes", "--json"]))
    }

    func notesWorkspace() async throws -> AgentNotesWorkspace {
        try decodeNotesWorkspace(try await execute(arguments: ["notes", "list", "--json"]))
    }

    func notes(noteID: String) async throws -> AgentNotes {
        try decodeNotes(try await execute(arguments: ["notes", "show", "--note", noteID, "--json"]))
    }

    func setNotes(_ content: String) async throws -> AgentNotes {
        try decodeNotes(try await execute(
            arguments: ["notes", "set", "--json"],
            standardInput: Data(content.utf8)
        ))
    }

    func setNotes(_ content: String, noteID: String) async throws -> AgentNotesWorkspace {
        _ = try await execute(
            arguments: ["notes", "set", "--note", noteID, "--json"],
            standardInput: Data(content.utf8)
        )
        return try await notesWorkspace()
    }

    func createNote(_ content: String) async throws -> AgentNotesWorkspace {
        try decodeNotesWorkspace(try await execute(
            arguments: ["notes", "new", "--json"],
            standardInput: Data(content.utf8)
        ))
    }

    func openNote(_ noteID: String) async throws -> AgentNotesWorkspace {
        try decodeNotesWorkspace(try await execute(arguments: ["notes", "open", noteID, "--json"]))
    }

    func closeNote(_ noteID: String) async throws -> AgentNotesWorkspace {
        try decodeNotesWorkspace(try await execute(arguments: ["notes", "close", noteID, "--json"]))
    }

    func deleteNote(_ noteID: String) async throws -> AgentNotesWorkspace {
        try decodeNotesWorkspace(try await execute(arguments: ["notes", "delete", noteID, "--json"]))
    }

    func setNotePinned(_ noteID: String, pinned: Bool) async throws -> AgentNotesWorkspace {
        let command = pinned ? "pin" : "unpin"
        return try decodeNotesWorkspace(try await execute(arguments: ["notes", command, noteID, "--json"]))
    }

    func reorderPinnedNotes(_ noteIDs: [String]) async throws -> AgentNotesWorkspace {
        try decodeNotesWorkspace(try await execute(arguments: ["notes", "reorder-pinned"] + noteIDs + ["--json"]))
    }

    func repositorySync(refresh: Bool) async throws -> RepositorySyncReport {
        var arguments = ["sync", "check", "--json"]
        if !refresh {
            arguments.append("--no-fetch")
        }
        return try decodeRepositorySync(try await execute(arguments: arguments))
    }

    func syncRepositories(_ projectIDs: [String]) async throws -> RepositorySyncReport {
        try decodeRepositorySync(try await execute(arguments: ["sync", "apply", "--json", "--yes"] + projectIDs))
    }

    func dependencyLimits() async throws -> DependencyLimitReport {
        do {
            return try JSONDecoder().decode(
                DependencyLimitReport.self,
                from: try await execute(arguments: ["limits", "--json"])
            )
        } catch let error as CLIClientError {
            throw error
        } catch {
            throw CLIClientError.invalidOutput(error.localizedDescription)
        }
    }

    func externalActivity() async throws -> ExternalActivitySnapshot {
        try decodeExternalActivity(try await execute(arguments: ["activity", "list", "--json"]))
    }

    func pruneExternalActivity() async throws -> ExternalActivitySnapshot {
        try decodeExternalActivity(try await execute(arguments: ["activity", "prune", "--json"]))
    }

    func integrationStatus(_ provider: String) async throws -> IntegrationHealthStatus {
        do {
            return try JSONDecoder().decode(
                IntegrationHealthStatus.self,
                from: try await execute(arguments: ["integrations", "status", provider, "--json"])
            )
        } catch let error as CLIClientError {
            throw error
        } catch {
            throw CLIClientError.invalidOutput(error.localizedDescription)
        }
    }

    func installAgent() async throws {
        _ = try await execute(arguments: ["agent", "install"])
    }

    func startAgent() throws {
        _ = try executeSynchronously(arguments: ["agent", "start"])
    }

    func stopAgent() throws {
        _ = try executeSynchronously(arguments: ["agent", "stop"])
    }

    private func decodeNotes(_ data: Data) throws -> AgentNotes {
        do {
            return try JSONDecoder().decode(AgentNotes.self, from: data)
        } catch {
            throw CLIClientError.invalidOutput(error.localizedDescription)
        }
    }

    private func decodeNotesWorkspace(_ data: Data) throws -> AgentNotesWorkspace {
        do {
            return try JSONDecoder().decode(AgentNotesWorkspace.self, from: data)
        } catch {
            throw CLIClientError.invalidOutput(error.localizedDescription)
        }
    }

    private func decodeRepositorySync(_ data: Data) throws -> RepositorySyncReport {
        do {
            return try JSONDecoder().decode(RepositorySyncReport.self, from: data)
        } catch {
            throw CLIClientError.invalidOutput(error.localizedDescription)
        }
    }

    private func decodeExternalActivity(_ data: Data) throws -> ExternalActivitySnapshot {
        do {
            return try JSONDecoder().decode(ExternalActivitySnapshot.self, from: data)
        } catch {
            throw CLIClientError.invalidOutput(error.localizedDescription)
        }
    }

    private func execute(arguments: [String], standardInput: Data? = nil) async throws -> Data {
        let executableURL = executableURL
        return try await Task.detached(priority: .userInitiated) {
            try Self.executeBlocking(
                executableURL: executableURL,
                arguments: arguments,
                standardInput: standardInput
            )
        }.value
    }

    private func executeSynchronously(
        arguments: [String],
        standardInput: Data? = nil
    ) throws -> Data {
        try Self.executeBlocking(
            executableURL: executableURL,
            arguments: arguments,
            standardInput: standardInput
        )
    }

    private static func executeBlocking(
        executableURL: URL,
        arguments: [String],
        standardInput: Data?
    ) throws -> Data {
        guard FileManager.default.isExecutableFile(atPath: executableURL.path) else {
            throw CLIClientError.helperMissing(executableURL.path)
        }
        let process = Process()
        let standardOutput = Pipe()
        let standardError = Pipe()
        process.executableURL = executableURL
        process.arguments = arguments
        process.standardOutput = standardOutput
        process.standardError = standardError
        let inputPipe = standardInput == nil ? nil : Pipe()
        process.standardInput = inputPipe
        var environment = ProcessInfo.processInfo.environment
        environment["PATH"] = commandPath(existing: environment["PATH"])
        process.environment = environment

        try process.run()
        if let standardInput, let inputPipe {
            try inputPipe.fileHandleForWriting.write(contentsOf: standardInput)
            try inputPipe.fileHandleForWriting.close()
        }
        let outputData = standardOutput.fileHandleForReading.readDataToEndOfFile()
        let errorData = standardError.fileHandleForReading.readDataToEndOfFile()
        process.waitUntilExit()
        guard process.terminationStatus == 0 else {
            let message = String(data: errorData, encoding: .utf8)?
                .trimmingCharacters(in: .whitespacesAndNewlines) ?? "unknown error"
            throw CLIClientError.commandFailed(process.terminationStatus, message)
        }
        return outputData
    }

    static func defaultExecutableURL() -> URL {
        if let override = UserDefaults.standard.string(forKey: "BeaconCLIPath"), !override.isEmpty {
            return URL(fileURLWithPath: override)
        }
        return Bundle.main.bundleURL.appendingPathComponent("Contents/MacOS/beacon-cli")
    }

    static func commandPath(existing: String?) -> String {
        let required = ["/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"]
        let current = (existing ?? "").split(separator: ":").map(String.init)
        return (required + current).reduce(into: [String]()) { result, item in
            if !result.contains(item) {
                result.append(item)
            }
        }.joined(separator: ":")
    }
}
