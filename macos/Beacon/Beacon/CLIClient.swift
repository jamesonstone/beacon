import Foundation

protocol CLIClientProtocol {
    func scan() async throws -> BeaconSnapshot
    func setProjectTracked(_ github: String, tracked: Bool) async throws
    func notes() async throws -> AgentNotes
    func setNotes(_ content: String) async throws -> AgentNotes
    func repositorySync(refresh: Bool) async throws -> RepositorySyncReport
    func syncRepositories(_ projectIDs: [String]) async throws -> RepositorySyncReport
}

extension CLIClientProtocol {
    func notes() async throws -> AgentNotes { throw AgentClientError.command("signal notes are unavailable") }
    func setNotes(_ content: String) async throws -> AgentNotes { throw AgentClientError.command("signal notes are unavailable") }
    func repositorySync(refresh: Bool) async throws -> RepositorySyncReport { throw AgentClientError.command("repository sync is unavailable") }
    func syncRepositories(_ projectIDs: [String]) async throws -> RepositorySyncReport { throw AgentClientError.command("repository sync is unavailable") }
}

protocol AgentInstallerProtocol {
    func installAgent() async throws
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

struct CLIClient: CLIClientProtocol, AgentInstallerProtocol {
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

    func setNotes(_ content: String) async throws -> AgentNotes {
        try decodeNotes(try await execute(
            arguments: ["notes", "set", "--json"],
            standardInput: Data(content.utf8)
        ))
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

    func installAgent() async throws {
        _ = try await execute(arguments: ["agent", "install"])
    }

    private func decodeNotes(_ data: Data) throws -> AgentNotes {
        do {
            return try JSONDecoder().decode(AgentNotes.self, from: data)
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
            environment["PATH"] = CLIClient.commandPath(existing: environment["PATH"])
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
                let message = String(data: errorData, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "unknown error"
                throw CLIClientError.commandFailed(process.terminationStatus, message)
            }
            return outputData
        }.value
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
