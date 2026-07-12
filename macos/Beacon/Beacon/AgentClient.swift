import Darwin
import Foundation

struct AgentProjectStatus: Codable, Equatable {
    let projectID: String
    let name: String
    let path: String
    let trackingState: String
    let stage: String
    let revision: UInt64
    let updatedAt: String
    let mutedAt: String?
    let lastProbeAt: String?

    enum CodingKeys: String, CodingKey {
        case name, path, stage, revision
        case projectID = "project_id"
        case trackingState = "tracking_state"
        case updatedAt = "updated_at"
        case mutedAt = "muted_at"
        case lastProbeAt = "last_probe_at"
    }
}

struct AgentStatusDetails: Codable, Equatable {
    let running: Bool
    let pid: Int
    let startedAt: String?
    let refreshing: Bool
    let scanID: String?
    let projectCount: Int
    let socket: String

    enum CodingKeys: String, CodingKey {
        case running, pid, refreshing, socket
        case startedAt = "started_at"
        case scanID = "scan_id"
        case projectCount = "project_count"
    }
}

struct AgentEvent: Codable, Equatable {
    let protocolVersion: Int
    let requestID: String?
    let type: String
    let scanID: String?
    let projectID: String?
    let revision: UInt64?
    let stage: String?
    let generatedAt: String
    let message: String?
    let snapshot: BeaconSnapshot?
    let projects: [AgentProjectStatus]?
    let status: AgentStatusDetails?

    enum CodingKeys: String, CodingKey {
        case type, revision, stage, message, snapshot, projects, status
        case protocolVersion = "protocol_version"
        case requestID = "request_id"
        case scanID = "scan_id"
        case projectID = "project_id"
        case generatedAt = "generated_at"
    }
}

protocol AgentClientProtocol {
    func snapshot() async throws -> AgentEvent
    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error>
    func refresh(project: String?) async throws -> String
    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent
    func status() async throws -> AgentStatusDetails
    func setLaneAttention(_ id: String, state: String) async throws -> AgentEvent
    func setLanePinned(_ id: String, pinned: Bool) async throws -> AgentEvent
    func setLaneNote(_ id: String, note: String) async throws -> AgentEvent
    func addLaneTag(_ id: String, tag: String) async throws -> AgentEvent
    func removeLaneTag(_ id: String, tag: String) async throws -> AgentEvent
    func markLaneSeen(_ id: String) async throws -> AgentEvent
    func addManualLane(_ title: String) async throws -> AgentEvent
}

extension AgentClientProtocol {
    func setLaneAttention(_ id: String, state: String) async throws -> AgentEvent { throw AgentClientError.command("lane attention is unavailable") }
    func setLanePinned(_ id: String, pinned: Bool) async throws -> AgentEvent { throw AgentClientError.command("lane pinning is unavailable") }
    func setLaneNote(_ id: String, note: String) async throws -> AgentEvent { throw AgentClientError.command("lane notes are unavailable") }
    func addLaneTag(_ id: String, tag: String) async throws -> AgentEvent { throw AgentClientError.command("lane tags are unavailable") }
    func removeLaneTag(_ id: String, tag: String) async throws -> AgentEvent { throw AgentClientError.command("lane tags are unavailable") }
    func markLaneSeen(_ id: String) async throws -> AgentEvent { throw AgentClientError.command("lane acknowledgement is unavailable") }
    func addManualLane(_ title: String) async throws -> AgentEvent { throw AgentClientError.command("manual lanes are unavailable") }
}

enum AgentClientError: LocalizedError {
    case connection(String)
    case invalidResponse(String)
    case command(String)

    var errorDescription: String? {
        switch self {
        case .connection(let message): "Beacon background agent is unavailable: \(message)"
        case .invalidResponse(let message): "Beacon agent returned invalid data: \(message)"
        case .command(let message): "Beacon agent command failed: \(message)"
        }
    }
}

actor AgentClient: AgentClientProtocol {
    private let socketPath: String

    init(socketPath: String = AgentClient.defaultSocketPath()) {
        self.socketPath = socketPath
    }

    func snapshot() async throws -> AgentEvent {
        try await request(type: "get_snapshot")
    }

    func refresh(project: String?) async throws -> String {
        let event = try await request(
            type: project == nil ? "refresh_all" : "refresh_project",
            projectID: project
        )
        guard event.type != "project_failed" else {
            throw AgentClientError.command(event.message ?? "refresh failed")
        }
        return event.scanID ?? ""
    }

    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent {
        let event = try await request(
            type: "set_tracking_state",
            projectID: github,
            trackingState: tracked ? "tracked" : "muted"
        )
        guard event.type != "project_failed" else {
            throw AgentClientError.command(event.message ?? "tracking update failed")
        }
        return event
    }

    func setLaneAttention(_ id: String, state: String) async throws -> AgentEvent {
        try await request(type: "set_lane_attention", laneID: id, attentionState: state)
    }

    func setLanePinned(_ id: String, pinned: Bool) async throws -> AgentEvent {
        try await request(type: "set_lane_pinned", laneID: id, pinned: pinned)
    }

    func setLaneNote(_ id: String, note: String) async throws -> AgentEvent {
        try await request(type: "set_lane_note", laneID: id, note: note)
    }

    func addLaneTag(_ id: String, tag: String) async throws -> AgentEvent {
        try await request(type: "add_lane_tag", laneID: id, tag: tag)
    }

    func removeLaneTag(_ id: String, tag: String) async throws -> AgentEvent {
        try await request(type: "remove_lane_tag", laneID: id, tag: tag)
    }

    func markLaneSeen(_ id: String) async throws -> AgentEvent {
        try await request(type: "mark_lane_seen", laneID: id)
    }

    func addManualLane(_ title: String) async throws -> AgentEvent {
        try await request(type: "add_manual_lane", title: title)
    }

    func status() async throws -> AgentStatusDetails {
        let event = try await request(type: "get_agent_status")
        guard let status = event.status else {
            throw AgentClientError.invalidResponse("missing status")
        }
        return status
    }

    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error> {
        let path = socketPath
        return AsyncThrowingStream { continuation in
            let task = Task.detached(priority: .userInitiated) {
                do {
                    let socket = try UnixSocket(path: path)
                    defer { socket.close() }
                    try socket.send(try Self.requestData(type: "subscribe"))
                    while !Task.isCancelled {
                        let data = try socket.readLine()
                        let event = try JSONDecoder().decode(AgentEvent.self, from: data)
                        guard event.protocolVersion == 1 else {
                            throw AgentClientError.invalidResponse("unsupported protocol \(event.protocolVersion)")
                        }
                        continuation.yield(event)
                    }
                    continuation.finish()
                } catch {
                    continuation.finish(throwing: error)
                }
            }
            continuation.onTermination = { _ in task.cancel() }
        }
    }

    private func request(type: String, projectID: String? = nil, trackingState: String? = nil, laneID: String? = nil, attentionState: String? = nil, pinned: Bool? = nil, note: String? = nil, tag: String? = nil, title: String? = nil) async throws -> AgentEvent {
        let path = socketPath
        let payload = try Self.requestData(type: type, projectID: projectID, trackingState: trackingState, laneID: laneID, attentionState: attentionState, pinned: pinned, note: note, tag: tag, title: title)
        return try await Task.detached(priority: .userInitiated) {
            let socket = try UnixSocket(path: path)
            defer { socket.close() }
            try socket.send(payload)
            let event = try JSONDecoder().decode(AgentEvent.self, from: socket.readLine())
            guard event.protocolVersion == 1 else {
                throw AgentClientError.invalidResponse("unsupported protocol \(event.protocolVersion)")
            }
            return event
        }.value
    }

    private static func requestData(type: String, projectID: String? = nil, trackingState: String? = nil, laneID: String? = nil, attentionState: String? = nil, pinned: Bool? = nil, note: String? = nil, tag: String? = nil, title: String? = nil) throws -> Data {
        var object: [String: Any] = [
            "protocol_version": 1,
            "request_id": UUID().uuidString.lowercased(),
            "type": type,
        ]
        object["project_id"] = projectID
        object["tracking_state"] = trackingState
        object["lane_id"] = laneID
        object["attention_state"] = attentionState
        object["pinned"] = pinned
        object["note"] = note
        object["tag"] = tag
        object["title"] = title
        var data = try JSONSerialization.data(withJSONObject: object)
        data.append(0x0A)
        return data
    }

    static func defaultSocketPath() -> String {
        if let cache = ProcessInfo.processInfo.environment["XDG_CACHE_HOME"], !cache.isEmpty {
            return URL(fileURLWithPath: cache).appendingPathComponent("beacon/agent.sock").path
        }
        return FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent(".cache/beacon/agent.sock").path
    }
}

private final class UnixSocket: @unchecked Sendable {
    private var descriptor: Int32

    init(path: String) throws {
        descriptor = Darwin.socket(AF_UNIX, SOCK_STREAM, 0)
        guard descriptor >= 0 else {
            throw AgentClientError.connection(String(cString: strerror(errno)))
        }
        var address = sockaddr_un()
        address.sun_family = sa_family_t(AF_UNIX)
        let maximum = MemoryLayout.size(ofValue: address.sun_path)
        guard path.utf8.count + 1 <= maximum else {
            close()
            throw AgentClientError.connection("socket path is too long")
        }
        path.withCString { source in
            withUnsafeMutablePointer(to: &address.sun_path) { tuple in
                let destination = UnsafeMutableRawPointer(tuple).assumingMemoryBound(to: CChar.self)
                strlcpy(destination, source, maximum)
            }
        }
        let length = socklen_t(MemoryLayout<sa_family_t>.size + path.utf8.count + 1)
        let result = withUnsafePointer(to: &address) { pointer in
            pointer.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                Darwin.connect(descriptor, $0, length)
            }
        }
        guard result == 0 else {
            let message = String(cString: strerror(errno))
            close()
            throw AgentClientError.connection(message)
        }
    }

    func send(_ data: Data) throws {
        try data.withUnsafeBytes { bytes in
            guard let base = bytes.baseAddress else { return }
            var offset = 0
            while offset < bytes.count {
                let count = Darwin.write(descriptor, base.advanced(by: offset), bytes.count - offset)
                guard count > 0 else {
                    throw AgentClientError.connection(String(cString: strerror(errno)))
                }
                offset += count
            }
        }
    }

    func readLine() throws -> Data {
        var data = Data()
        var byte: UInt8 = 0
        while true {
            let count = Darwin.read(descriptor, &byte, 1)
            guard count > 0 else {
                throw AgentClientError.connection(count == 0 ? "connection closed" : String(cString: strerror(errno)))
            }
            if byte == 0x0A { return data }
            data.append(byte)
        }
    }

    func close() {
        if descriptor >= 0 {
            Darwin.close(descriptor)
            descriptor = -1
        }
    }
}
