import AppKit
import Foundation

struct ProjectLaneGroup: Identifiable, Equatable {
    let id: String
    let name: String
    let progress: ProgressDetails?
    let lanes: [WorkLane]
}

private struct TrackingMutation {
    let projectID: String
    let tracked: Bool
}

@MainActor
final class AppState: ObservableObject {
    @Published private(set) var snapshot: BeaconSnapshot?
    @Published private(set) var isScanning = false
    @Published private(set) var lastError: String?
    @Published private(set) var mutatingProjects: Set<String> = []
    @Published private(set) var agentAvailable = false
    @Published private(set) var projectStatuses: [String: AgentProjectStatus] = [:]
    @Published private(set) var reactivationMessage: String?

    private let agent: AgentClientProtocol
    private let installer: AgentInstallerProtocol?
    private var subscriptionTask: Task<Void, Never>?
    private var revisions: [String: UInt64] = [:]
    private var activeScanID: String?
    private var pendingTrackingStates: [String: Bool] = [:]
    private var trackingQueue: [TrackingMutation] = []
    private var trackingTask: Task<Void, Never>?
    private var trackingFailures: [String: String] = [:]

    init(agent: AgentClientProtocol = AgentClient(), installer: AgentInstallerProtocol? = CLIClient()) {
        self.agent = agent
        self.installer = installer
    }

    convenience init(client: CLIClientProtocol) {
        self.init(agent: DirectAgentAdapter(client: client), installer: nil)
    }

    var readyCount: Int { snapshot?.summary.reviewReady ?? 0 }

    var inProgressCount: Int {
        if let working = snapshot?.workingSet {
            return working.active.count + working.waiting.count + working.recent.count
        }
        guard let groups = snapshot?.groups else { return 0 }
        return groups.ready.count + groups.action.count + groups.waiting.count
    }

    var quietProjectCount: Int { quietProjectGroups().count }

    var queuedTrackingCount: Int { mutatingProjects.count }

    var untrackedProjectCount: Int {
        snapshot?.summary.untrackedProjects ?? untrackedProjects.count
    }

    var trackedProjects: [BeaconProject] {
        (snapshot?.projects ?? []).filter {
            pendingTrackingStates[$0.github] ?? $0.isTracked
        }
    }

    var untrackedProjects: [BeaconProject] {
        (snapshot?.projects ?? []).filter {
            !(pendingTrackingStates[$0.github] ?? $0.isTracked)
        }
    }

    var loadingProjects: [AgentProjectStatus] {
        let loaded = Set((snapshot?.projects ?? []).map(\.github))
        return projectStatuses.values
            .filter { $0.trackingState == "tracked" && !loaded.contains($0.projectID) }
            .sorted {
                if $0.name != $1.name { return $0.name < $1.name }
                return $0.projectID < $1.projectID
            }
    }

    func start() {
        guard subscriptionTask == nil else { return }
        subscriptionTask = Task { [weak self] in
            await self?.listenForAgent()
        }
    }

    func stop() {
        subscriptionTask?.cancel()
        subscriptionTask = nil
    }

    func scan() async {
        guard !isScanning else { return }
        isScanning = true
        do {
            activeScanID = try await agent.refresh(project: nil)
            let cached = try await agent.snapshot()
            apply(cached)
            if activeScanID == "direct" {
                activeScanID = nil
                isScanning = false
            } else {
                reconcile(try await agent.status())
            }
        } catch {
            isScanning = false
            activeScanID = nil
            lastError = error.localizedDescription
        }
    }

    func setProjectTracked(_ project: BeaconProject, tracked: Bool) {
        guard !mutatingProjects.contains(project.github) else { return }
        pendingTrackingStates[project.github] = tracked
        mutatingProjects.insert(project.github)
        trackingFailures.removeValue(forKey: project.github)
        trackingQueue.append(TrackingMutation(projectID: project.github, tracked: tracked))
        startTrackingQueue()
    }

    func setLaneAttention(_ lane: WorkLane, state: String) async {
        await applyLaneMutation { try await agent.setLaneAttention(lane.id, state: state) }
    }

    func setLanePinned(_ lane: WorkLane, pinned: Bool) async {
        await applyLaneMutation { try await agent.setLanePinned(lane.id, pinned: pinned) }
    }

    func setLaneNote(_ lane: WorkLane, note: String) async {
        await applyLaneMutation { try await agent.setLaneNote(lane.id, note: note) }
    }

    func addLaneTag(_ lane: WorkLane, tag: String) async {
        await applyLaneMutation { try await agent.addLaneTag(lane.id, tag: tag) }
    }

    func removeLaneTag(_ lane: WorkLane, tag: String) async {
        await applyLaneMutation { try await agent.removeLaneTag(lane.id, tag: tag) }
    }

    func markLaneSeen(_ lane: WorkLane) async {
        await applyLaneMutation { try await agent.markLaneSeen(lane.id) }
    }

    func addManualLane(_ title: String) async {
        await applyLaneMutation { try await agent.addManualLane(title) }
    }

    private func applyLaneMutation(_ operation: () async throws -> AgentEvent) async {
        do {
            apply(try await operation())
            lastError = nil
        } catch {
            lastError = error.localizedDescription
        }
    }

    private func startTrackingQueue() {
        guard trackingTask == nil else { return }
        trackingTask = Task { [weak self] in
            await self?.drainTrackingQueue()
        }
    }

    private func drainTrackingQueue() async {
        while !Task.isCancelled, !trackingQueue.isEmpty {
            let mutation = trackingQueue.removeFirst()
            do {
                let event = try await agent.setProjectTracked(
                    mutation.projectID,
                    tracked: mutation.tracked
                )
                apply(event)
            } catch {
                trackingFailures[mutation.projectID] = error.localizedDescription
            }
            pendingTrackingStates.removeValue(forKey: mutation.projectID)
            mutatingProjects.remove(mutation.projectID)
            if !trackingFailures.isEmpty {
                lastError = trackingFailureSummary()
            }
        }
        trackingTask = nil
    }

    private func trackingFailureSummary() -> String {
        trackingFailures
            .sorted { $0.key < $1.key }
            .map { "\($0.key): \($0.value)" }
            .joined(separator: "\n")
    }

    func isMutating(_ project: BeaconProject) -> Bool {
        mutatingProjects.contains(project.github)
    }

    func enableAgent() async {
        guard let installer else {
            lastError = "The bundled Beacon helper cannot install the background agent."
            return
        }
        do {
            try await installer.installAgent()
            lastError = nil
            agentAvailable = true
            stop()
            start()
        } catch {
            lastError = error.localizedDescription
        }
    }

    func stage(for projectID: String) -> String {
        projectStatuses[projectID]?.stage ?? "cached"
    }

    func lanes(for identifiers: [String]) -> [WorkLane] {
        let lanesByID = Dictionary(uniqueKeysWithValues: (snapshot?.lanes ?? []).map { ($0.id, $0) })
        return identifiers.compactMap { lanesByID[$0] }
    }

    func projectGroups(for lanes: [WorkLane]) -> [ProjectLaneGroup] {
        var remaining = lanes
        var groups: [ProjectLaneGroup] = []

        for project in snapshot?.projects ?? [] {
            let matching = remaining.filter {
                $0.github == project.github || project.laneIDs.contains($0.id)
            }
            guard !matching.isEmpty else { continue }
            let matchingIDs = Set(matching.map(\.id))
            remaining.removeAll { matchingIDs.contains($0.id) }
            groups.append(ProjectLaneGroup(
                id: project.github,
                name: project.name,
                progress: project.progress,
                lanes: matching
            ))
        }

        while let first = remaining.first {
            let matching = remaining.filter { $0.github == first.github }
            let matchingIDs = Set(matching.map(\.id))
            remaining.removeAll { matchingIDs.contains($0.id) }
            groups.append(ProjectLaneGroup(
                id: first.github,
                name: first.repository,
                progress: nil,
                lanes: matching
            ))
        }
        return groups
    }

    func quietProjectGroups(matching query: String = "") -> [ProjectLaneGroup] {
        guard let snapshot else { return [] }
        let activeProjects = Set(lanes(for: snapshot.groups.ready + snapshot.groups.action + snapshot.groups.waiting).map(\.github))
        let quietLanes = lanes(for: snapshot.groups.idle).filter { !activeProjects.contains($0.github) }
        let groups = projectGroups(for: quietLanes)
        let normalizedQuery = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalizedQuery.isEmpty else { return groups }
        return groups.filter { group in
            group.name.lowercased().contains(normalizedQuery)
                || group.id.lowercased().contains(normalizedQuery)
                || group.lanes.contains { lane in
                    lane.repository.lowercased().contains(normalizedQuery)
                        || lane.branch.lowercased().contains(normalizedQuery)
                        || lane.pullRequest?.title.lowercased().contains(normalizedQuery) == true
                        || lane.issue?.title.lowercased().contains(normalizedQuery) == true
                }
        }
    }

    func open(_ lane: WorkLane) {
        if let target = Self.openTarget(for: lane) {
            NSWorkspace.shared.open(target)
        }
    }

    func openTopItem() {
        guard let lane = topLane() else { return }
        open(lane)
    }

    func topLane() -> WorkLane? {
        guard let snapshot else { return nil }
        let identifiers: [String]
        if let working = snapshot.workingSet {
            identifiers = working.active + working.waiting + working.recent
        } else {
            identifiers = snapshot.groups.ready + snapshot.groups.action + snapshot.groups.waiting
        }
        return lanes(for: identifiers).first { Self.openTarget(for: $0) != nil }
    }

    static func openTarget(for lane: WorkLane) -> URL? {
        if let pullRequest = lane.pullRequest, let url = webURL(pullRequest.url) {
            return url
        }
        if let issue = lane.issue, let url = webURL(issue.url) {
            return url
        }
        if let worktree = lane.worktree {
            return URL(fileURLWithPath: worktree.path)
        }
        return nil
    }

    func openConfig() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let config = home.appendingPathComponent(".config/beacon/config.yaml")
        if FileManager.default.fileExists(atPath: config.path) {
            NSWorkspace.shared.open(config)
        } else {
            NSWorkspace.shared.open(config.deletingLastPathComponent())
        }
    }

    private static func webURL(_ value: String) -> URL? {
        guard let url = URL(string: value), ["http", "https"].contains(url.scheme?.lowercased()) else {
            return nil
        }
        return url
    }

    private func listenForAgent() async {
        while !Task.isCancelled {
            do {
                let stream = try await agent.subscribe()
                agentAvailable = true
                lastError = nil
                for try await event in stream {
                    guard !Task.isCancelled else { return }
                    apply(event)
                    if event.type == "snapshot" || event.type == "heartbeat" {
                        reconcile(try await agent.status())
                    }
                }
            } catch {
                agentAvailable = false
                lastError = error.localizedDescription
            }
            try? await Task.sleep(for: .seconds(2))
        }
    }

    private func apply(_ event: AgentEvent) {
        guard event.protocolVersion == 1 else {
            lastError = "Beacon agent returned invalid data: unsupported protocol \(event.protocolVersion)"
            return
        }
        if activeScanID == nil, let eventScanID = event.scanID,
           !eventScanID.isEmpty, event.type != "scan_completed" {
            activeScanID = eventScanID
            isScanning = true
        }
        if let activeScanID, let eventScanID = event.scanID,
           !eventScanID.isEmpty, eventScanID != activeScanID,
           event.type != "heartbeat" {
            return
        }
        if let projectID = event.projectID, let revision = event.revision {
            guard revision >= revisions[projectID, default: 0] else { return }
            revisions[projectID] = revision
            if let existing = projectStatuses[projectID] {
                projectStatuses[projectID] = AgentProjectStatus(
                    projectID: existing.projectID,
                    name: existing.name,
                    path: existing.path,
                    trackingState: existing.trackingState,
                    stage: event.stage ?? existing.stage,
                    revision: revision,
                    updatedAt: event.generatedAt,
                    mutedAt: existing.mutedAt,
                    lastProbeAt: existing.lastProbeAt
                )
            }
        }
        for status in event.projects ?? [] {
            guard status.revision >= revisions[status.projectID, default: 0] else { continue }
            revisions[status.projectID] = status.revision
            projectStatuses[status.projectID] = status
        }
        if let latest = event.snapshot {
            guard latest.schemaVersion == 3 else {
                lastError = "Beacon CLI returned invalid JSON: unsupported schema version \(latest.schemaVersion)"
                return
            }
            snapshot = latest
            if event.type != "project_failed" {
                lastError = nil
            }
        }
        if event.type == "project_reactivated" {
            reactivationMessage = "Reactivated: \(event.message ?? "new project activity")"
        }
        if event.type == "project_failed" {
            lastError = event.message ?? "Project refresh failed — showing previous result"
        }
        if event.type == "scan_completed", activeScanID == nil || activeScanID == event.scanID {
            isScanning = false
            activeScanID = nil
        }
    }

    private func reconcile(_ status: AgentStatusDetails) {
        agentAvailable = status.running
        isScanning = status.refreshing
        activeScanID = status.refreshing ? status.scanID : nil
    }
}

private actor DirectAgentAdapter: AgentClientProtocol {
    let client: CLIClientProtocol
    var latest: BeaconSnapshot?

    init(client: CLIClientProtocol) {
        self.client = client
    }

    func snapshot() async throws -> AgentEvent {
        if latest == nil { latest = try await client.scan() }
        return event(type: "snapshot", snapshot: latest)
    }

    func subscribe() async throws -> AsyncThrowingStream<AgentEvent, Error> {
        let initial = try await snapshot()
        return AsyncThrowingStream { continuation in
            continuation.yield(initial)
            continuation.finish()
        }
    }

    func refresh(project: String?) async throws -> String {
        latest = try await client.scan()
        return "direct"
    }

    func setProjectTracked(_ github: String, tracked: Bool) async throws -> AgentEvent {
        try await client.setProjectTracked(github, tracked: tracked)
        latest = try await client.scan()
        return event(type: "tracking_changed", snapshot: latest)
    }

    func status() async throws -> AgentStatusDetails {
        AgentStatusDetails(running: true, pid: 0, startedAt: nil, refreshing: false, scanID: nil, projectCount: latest?.projects.count ?? 0, socket: "direct")
    }

    private func event(type: String, snapshot: BeaconSnapshot?) -> AgentEvent {
        AgentEvent(protocolVersion: 1, requestID: nil, type: type, scanID: nil, projectID: nil, revision: nil, stage: "ready", generatedAt: snapshot?.generatedAt ?? "", message: nil, snapshot: snapshot, projects: nil, status: nil)
    }
}
