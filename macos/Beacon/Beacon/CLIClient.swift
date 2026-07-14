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
