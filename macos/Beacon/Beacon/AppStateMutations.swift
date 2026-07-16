import Foundation

@MainActor
extension AppState {
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

    func ignoreLane(_ lane: WorkLane) async {
        await setLaneAttention(lane, state: "parked")
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

    func presentedFollowState(for project: BeaconProject) -> String {
        guard let followed = pendingTrackingStates[project.github] else {
            return project.effectiveFollowState
        }
        return followed ? "following" : "quiet"
    }
}
