import Foundation

@MainActor
extension AppState {
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

    func checkDependencyLimits() async {
        guard !isCheckingDependencyLimits else { return }
        guard let dependencyLimitsClient else {
            dependencyLimitsError = "The bundled Beacon helper cannot inspect dependency limits."
            return
        }
        isCheckingDependencyLimits = true
        defer { isCheckingDependencyLimits = false }
        do {
            dependencyLimitsReport = try await dependencyLimitsClient.dependencyLimits()
            dependencyLimitsError = nil
        } catch {
            dependencyLimitsError = error.localizedDescription
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
