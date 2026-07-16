import Foundation

@MainActor
extension AppState {
    func activityChip(projectID: String, laneID: String? = nil) -> ExternalActivityChip? {
        let records = externalActivity.records.filter { record in
            guard record.projectID == projectID else { return false }
            if let laneID {
                return record.laneID == laneID
            }
            return record.laneID == nil || record.laneID?.isEmpty == true
        }
        return ExternalActivityPresentation.chip(for: records)
    }

    func refreshExternalContext() async {
        guard let externalActivityClient else { return }
        do {
            applyExternalActivity(try await externalActivityClient.externalActivity())
            ensureActivityCacheWatcher()
        } catch {
            // External activity is optional context. Older helpers and missing
            // caches must not affect the evidence-backed dashboard.
        }
        await refreshIntegrationHealth()
    }

    func refreshIntegrationHealth() async {
        guard let externalActivityClient else { return }
        var statuses: [String: IntegrationHealthStatus] = [:]
        for provider in ["codex", "claude-code"] {
            if let status = try? await externalActivityClient.integrationStatus(provider) {
                statuses[provider] = status
            }
        }
        integrationHealth = statuses
    }

    func applyExternalActivity(_ snapshot: ExternalActivitySnapshot) {
        guard snapshot.version == 1 else { return }
        externalActivity = snapshot
        externalActivityExpiryTask?.cancel()
        externalActivityExpiryTask = nil
        guard let value = snapshot.nextExpiry,
              let expiry = Self.externalActivityDate(value) else { return }
        let delay = max(0, expiry.timeIntervalSinceNow)
        externalActivityExpiryTask = Task { [weak self] in
            if delay > 0 {
                let nanoseconds = UInt64(delay * 1_000_000_000)
                try? await Task.sleep(nanoseconds: nanoseconds)
            }
            guard !Task.isCancelled else { return }
            await self?.pruneExternalActivity()
        }
    }

    static func externalActivityDate(_ value: String) -> Date? {
        let fractional = ISO8601DateFormatter()
        fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return fractional.date(from: value) ?? ISO8601DateFormatter().date(from: value)
    }

    static func externalActivityCacheDirectory(
        environment: [String: String] = ProcessInfo.processInfo.environment,
        home: URL = FileManager.default.homeDirectoryForCurrentUser
    ) -> URL {
        let cacheHome: URL
        if let value = environment["XDG_CACHE_HOME"], !value.isEmpty {
            cacheHome = URL(fileURLWithPath: value, isDirectory: true)
        } else {
            cacheHome = home.appendingPathComponent(".cache", isDirectory: true)
        }
        return cacheHome.appendingPathComponent("beacon", isDirectory: true)
    }

    func startExternalActivityMonitoring() {
        guard externalActivityClient != nil, externalActivityTask == nil else { return }
        externalActivityTask = Task { [weak self] in
            await self?.refreshExternalContext()
        }
    }

    private func ensureActivityCacheWatcher() {
        guard activityCacheWatcher == nil else { return }
        let directory = Self.externalActivityCacheDirectory()
        guard FileManager.default.fileExists(atPath: directory.path) else { return }
        activityCacheWatcher = ActivityCacheWatcher(directory: directory) { [weak self] in
            Task { @MainActor [weak self] in
                self?.scheduleExternalContextReload()
            }
        }
    }

    private func scheduleExternalContextReload() {
        externalActivityReloadTask?.cancel()
        externalActivityReloadTask = Task { [weak self] in
            try? await Task.sleep(for: .milliseconds(75))
            guard !Task.isCancelled else { return }
            await self?.refreshExternalContext()
        }
    }

    private func pruneExternalActivity() async {
        guard let externalActivityClient else { return }
        do {
            applyExternalActivity(try await externalActivityClient.pruneExternalActivity())
        } catch {
            // Go is the expiry authority. Keep the last normalized view and
            // retry on the next cache event rather than hiding records in Swift.
        }
    }
}
