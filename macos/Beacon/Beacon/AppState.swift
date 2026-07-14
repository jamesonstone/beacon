import Foundation

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
    @Published private(set) var notesContent = ""
    @Published private(set) var notesPath = ""
    @Published private(set) var notesUpdatedAt: String?
    @Published private(set) var isSavingNotes = false
    @Published private(set) var notesError: String?
    @Published private(set) var repositorySyncReport: RepositorySyncReport?
    @Published private(set) var isCheckingRepositorySync = false
    @Published private(set) var isApplyingRepositorySync = false
    @Published private(set) var repositorySyncError: String?

    private let agent: AgentClientProtocol
    private let installer: AgentInstallerProtocol?
    private let repositorySyncFallback: CLIClientProtocol?
    private var subscriptionTask: Task<Void, Never>?
    private var revisions: [String: UInt64] = [:]
    private var activeScanID: String?
    private var pendingTrackingStates: [String: Bool] = [:]
    private var trackingQueue: [TrackingMutation] = []
    private var trackingTask: Task<Void, Never>?
    private var trackingFailures: [String: String] = [:]

    init(
        agent: AgentClientProtocol = AgentClient(),
        installer: AgentInstallerProtocol? = CLIClient(),
        repositorySyncFallback: CLIClientProtocol? = CLIClient()
    ) {
        self.agent = agent
        self.installer = installer
        self.repositorySyncFallback = repositorySyncFallback
    }

    convenience init(client: CLIClientProtocol) {
        self.init(
            agent: DirectAgentAdapter(client: client),
            installer: nil,
            repositorySyncFallback: client
        )
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

    var repositoriesNeedingSync: [RepositorySyncItem] {
        (repositorySyncReport?.repositories ?? []).filter(\.needsUpdate)
    }

    var safeRepositoryUpdates: [RepositorySyncItem] {
        repositoriesNeedingSync.filter(\.canUpdate)
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

    func setProjectFollowed(_ project: BeaconProject, followed: Bool) {
        guard !mutatingProjects.contains(project.github) else { return }
        pendingTrackingStates[project.github] = followed
        mutatingProjects.insert(project.github)
        trackingFailures.removeValue(forKey: project.github)
        trackingQueue.append(TrackingMutation(projectID: project.github, tracked: followed))
        startTrackingQueue()
    }

    func setProjectTracked(_ project: BeaconProject, tracked: Bool) {
        setProjectFollowed(project, followed: tracked)
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

    func loadNotes() async {
        do {
            apply(try await agent.notes())
            notesError = nil
        } catch {
            notesError = error.localizedDescription
        }
    }

    func saveNotes(_ content: String) async {
        guard !isSavingNotes else { return }
        isSavingNotes = true
        defer { isSavingNotes = false }
        do {
            apply(try await agent.setNotes(content))
            notesError = nil
        } catch {
            notesError = error.localizedDescription
        }
    }

    func checkRepositorySync(refresh: Bool) async {
        guard !isCheckingRepositorySync, !isApplyingRepositorySync else { return }
        isCheckingRepositorySync = true
        defer { isCheckingRepositorySync = false }
        do {
            repositorySyncReport = try await repositorySyncReport(refresh: refresh)
            repositorySyncError = nil
        } catch {
            repositorySyncError = error.localizedDescription
        }
    }

    func syncRepositories(_ projectIDs: [String]) async {
        guard !projectIDs.isEmpty, !isCheckingRepositorySync, !isApplyingRepositorySync else { return }
        isApplyingRepositorySync = true
        defer { isApplyingRepositorySync = false }
        do {
            let report = try await repositorySyncReport(applying: projectIDs)
            mergeRepositorySync(report)
            repositorySyncError = nil
        } catch {
            repositorySyncError = error.localizedDescription
        }
    }

    private func repositorySyncReport(refresh: Bool) async throws -> RepositorySyncReport {
        do {
            let event = try await agent.repositorySync(refresh: refresh)
            guard let report = event.repositorySync else {
                throw AgentClientError.invalidResponse("missing repository sync report")
            }
            return report
        } catch {
            guard shouldUseRepositorySyncFallback(for: error), let repositorySyncFallback else {
                throw error
            }
            return try await repositorySyncFallback.repositorySync(refresh: refresh)
        }
    }

    private func repositorySyncReport(applying projectIDs: [String]) async throws -> RepositorySyncReport {
        do {
            let event = try await agent.syncRepositories(projectIDs)
            guard let report = event.repositorySync else {
                throw AgentClientError.invalidResponse("missing repository sync result")
            }
            return report
        } catch {
            guard shouldUseRepositorySyncFallback(for: error), let repositorySyncFallback else {
                throw error
            }
            return try await repositorySyncFallback.syncRepositories(projectIDs)
        }
    }

    private func shouldUseRepositorySyncFallback(for error: Error) -> Bool {
        guard let agentError = error as? AgentClientError else { return false }
        switch agentError {
        case .connection:
            return true
        case .command(let message):
            return message.contains("unknown field")
                || message.contains("unknown agent request")
                || message.contains("repository sync is unavailable")
        case .invalidResponse:
            return false
        }
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

    func presentedFollowState(for project: BeaconProject) -> String {
        guard let followed = pendingTrackingStates[project.github] else {
            return project.effectiveFollowState
        }
        return followed ? "following" : "quiet"
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

    private func listenForAgent() async {
        while !Task.isCancelled {
            do {
                let stream = try await agent.subscribe()
                agentAvailable = true
                lastError = nil
                await loadNotes()
                await checkRepositorySync(refresh: false)
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
        if let notes = event.notes {
            notesContent = notes.content
            notesPath = notes.path
            notesUpdatedAt = notes.updatedAt
            notesError = nil
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

    private func mergeRepositorySync(_ report: RepositorySyncReport) {
        var merged = Dictionary(uniqueKeysWithValues: (repositorySyncReport?.repositories ?? []).map { ($0.projectID, $0) })
        for repository in report.repositories {
            merged[repository.projectID] = repository
        }
        repositorySyncReport = RepositorySyncReport(
            checkedAt: report.checkedAt,
            fetchAttempted: report.fetchAttempted,
            repositories: merged.values.sorted { $0.projectID < $1.projectID }
        )
    }
}
