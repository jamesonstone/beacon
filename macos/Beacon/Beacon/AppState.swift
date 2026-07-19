import Darwin
import Foundation

struct TrackingMutation {
    let projectID: String
    let tracked: Bool
}

final class ActivityCacheWatcher: @unchecked Sendable {
    private let source: DispatchSourceFileSystemObject

    init?(directory: URL, onChange: @escaping @Sendable () -> Void) {
        let descriptor = open(directory.path, O_EVTONLY)
        guard descriptor >= 0 else { return nil }
        let source = DispatchSource.makeFileSystemObjectSource(
            fileDescriptor: descriptor,
            eventMask: [.write, .rename, .delete],
            queue: DispatchQueue.global(qos: .utility)
        )
        self.source = source
        source.setEventHandler(handler: onChange)
        source.setCancelHandler { close(descriptor) }
        source.resume()
    }

    func cancel() {
        source.cancel()
    }

    deinit {
        source.cancel()
    }
}

@MainActor
final class AppState: ObservableObject {
    @Published var snapshot: BeaconSnapshot?
    @Published var isScanning = false
    @Published var lastError: String?
    @Published var mutatingProjects: Set<String> = []
    @Published var agentAvailable = false
    @Published var projectStatuses: [String: AgentProjectStatus] = [:]
    @Published var notesContent = ""
    @Published var notesPath = ""
    @Published var notesUpdatedAt: String?
    @Published var notesWorkspace: AgentNotesWorkspace?
    @Published var notesDraft = ""
    @Published var notesCurrentLine = ""
    @Published var notesSelectedText = ""
    @Published var notesAssistantAttachment = ""
    @Published var notesAssistantContextSource: NotesAssistantContextSource?
    @Published var notesAssistantPrompt = ""
    @Published var notesAssistantResponse: OllamaChatResponse?
    @Published var ollamaModels: [OllamaModel] = []
    @Published var ollamaConfiguredModel = ""
    @Published var ollamaSelectedModel = ""
    @Published var ollamaNotice: String?
    @Published var ollamaError: String?
    @Published var isLoadingOllamaModels = false
    @Published var isSavingOllamaDefault = false
    @Published var isSendingOllamaPrompt = false
    @Published var isSavingNotes = false
    @Published var notesError: String?
    @Published var repositorySyncReport: RepositorySyncReport?
    @Published var isCheckingRepositorySync = false
    @Published var isApplyingRepositorySync = false
    @Published var repositorySyncError: String?
    @Published var dependencyLimitsReport: DependencyLimitReport?
    @Published var isCheckingDependencyLimits = false
    @Published var dependencyLimitsError: String?
    @Published var externalActivity = ExternalActivitySnapshot.empty
    @Published var integrationHealth: [String: IntegrationHealthStatus] = [:]

    let agent: AgentClientProtocol
    private let installer: AgentLifecycleControllerProtocol?
    let notesFallback: CLIClientProtocol?
    let repositorySyncFallback: CLIClientProtocol?
    let dependencyLimitsClient: CLIClientProtocol?
    let externalActivityClient: CLIClientProtocol?
    let ollamaClient: OllamaClientProtocol
    private var subscriptionTask: Task<Void, Never>?
    var revisions: [String: UInt64] = [:]
    var activeScanID: String?
    var pendingTrackingStates: [String: Bool] = [:]
    var trackingQueue: [TrackingMutation] = []
    var trackingTask: Task<Void, Never>?
    var trackingFailures: [String: String] = [:]
    var externalActivityTask: Task<Void, Never>?
    var externalActivityReloadTask: Task<Void, Never>?
    var externalActivityExpiryTask: Task<Void, Never>?
    var activityCacheWatcher: ActivityCacheWatcher?
    let notesAutosave = SignalNotesAutosave()
    var notesDraftID = "general"
    var notesUseFallback = false
    var notesAssistantRequestGeneration = 0

    init(
        agent: AgentClientProtocol = AgentClient(),
        installer: AgentLifecycleControllerProtocol? = CLIClient(),
        notesFallback: CLIClientProtocol? = CLIClient(),
        repositorySyncFallback: CLIClientProtocol? = CLIClient(),
        dependencyLimitsClient: CLIClientProtocol? = CLIClient(),
        externalActivityClient: CLIClientProtocol? = nil,
        ollamaClient: OllamaClientProtocol = CLIClient()
    ) {
        self.agent = agent
        self.installer = installer
        self.notesFallback = notesFallback
        self.repositorySyncFallback = repositorySyncFallback
        self.dependencyLimitsClient = dependencyLimitsClient
        self.externalActivityClient = externalActivityClient
        self.ollamaClient = ollamaClient
    }

    convenience init(client: CLIClientProtocol) {
        self.init(
            agent: DirectAgentAdapter(client: client),
            installer: nil,
            notesFallback: client,
            repositorySyncFallback: client,
            dependencyLimitsClient: client,
            externalActivityClient: nil
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

    var dependencyUsagePercent: Int {
        dependencyLimitsReport?.highestUsagePercent ?? 0
    }

    var dependencyUsageLevel: DependencyUsageLevel {
        dependencyLimitsReport?.usageLevel ?? .unmeasured
    }

    func start() {
        guard subscriptionTask == nil else { return }
        startExternalActivityMonitoring()
        if let installer {
            do {
                try installer.startAgent()
                agentAvailable = true
                lastError = nil
            } catch {
                agentAvailable = false
                lastError = error.localizedDescription
            }
        }
        subscriptionTask = Task { [weak self] in
            await self?.listenForAgent()
        }
    }

    func stop() {
        subscriptionTask?.cancel()
        subscriptionTask = nil
        externalActivityTask?.cancel()
        externalActivityTask = nil
        externalActivityReloadTask?.cancel()
        externalActivityReloadTask = nil
        externalActivityExpiryTask?.cancel()
        externalActivityExpiryTask = nil
        activityCacheWatcher?.cancel()
        activityCacheWatcher = nil
    }

    @discardableResult
    func stopAgentSynchronously() -> Error? {
        guard let installer else { return nil }
        do {
            try installer.stopAgent()
            agentAvailable = false
            return nil
        } catch {
            return error
        }
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

}
