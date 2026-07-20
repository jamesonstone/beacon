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

    func reorderLane(_ draggedID: String, before targetID: String) async {
        guard draggedID != targetID,
              let snapshot,
              let workingSet = snapshot.workingSet,
              let group = laneGroup(for: draggedID, in: workingSet),
              group == laneGroup(for: targetID, in: workingSet),
              group == "parked" || sameFollowingOrderPartition(draggedID, targetID, in: snapshot)
        else { return }

        let visibleIDs = workingSet.active + workingSet.waiting + workingSet.recent + workingSet.parked
        var order = (workingSet.order ?? []) + visibleIDs.filter { !(workingSet.order ?? []).contains($0) }
        order.removeAll { $0 == draggedID }
        guard let targetIndex = order.firstIndex(of: targetID) else { return }
        order.insert(draggedID, at: targetIndex)
        await applyLaneMutation { try await agent.reorderLanes(order) }
    }

    func moveLane(_ laneID: String, by offset: Int) async {
        guard offset != 0,
              let snapshot,
              let workingSet = snapshot.workingSet,
              let group = laneGroup(for: laneID, in: workingSet)
        else { return }

        let visibleIDs = workingSet.active + workingSet.waiting + workingSet.recent + workingSet.parked
        var order = (workingSet.order ?? []) + visibleIDs.filter { !(workingSet.order ?? []).contains($0) }
        let peers = laneIDs(in: group, from: workingSet).filter {
            group == "parked" || sameFollowingOrderPartition(laneID, $0, in: snapshot)
        }
        guard let peerIndex = peers.firstIndex(of: laneID) else { return }
        let destination = peerIndex + offset
        guard peers.indices.contains(destination) else { return }

        let targetID = peers[destination]
        order.removeAll { $0 == laneID }
        guard let targetIndex = order.firstIndex(of: targetID) else { return }
        order.insert(laneID, at: offset < 0 ? targetIndex : targetIndex + 1)
        await applyLaneMutation { try await agent.reorderLanes(order) }
    }

    private func laneGroup(for laneID: String, in workingSet: WorkingSetGroups) -> String? {
        if workingSet.active.contains(laneID) { return "active" }
        if workingSet.waiting.contains(laneID) { return "waiting" }
        if workingSet.recent.contains(laneID) { return "recent" }
        if workingSet.parked.contains(laneID) { return "parked" }
        return nil
    }

    private func laneIDs(in group: String, from workingSet: WorkingSetGroups) -> [String] {
        switch group {
        case "active": return workingSet.active
        case "waiting": return workingSet.waiting
        case "recent": return workingSet.recent
        case "parked": return workingSet.parked
        default: return []
        }
    }

    private func sameFollowingOrderPartition(
        _ leftID: String,
        _ rightID: String,
        in snapshot: BeaconSnapshot
    ) -> Bool {
        guard let left = snapshot.lanes.first(where: { $0.id == leftID }),
              let right = snapshot.lanes.first(where: { $0.id == rightID })
        else { return false }
        return left.github == right.github
            && DashboardLanePresentation.identity(for: left) == DashboardLanePresentation.identity(for: right)
    }

    func moveLane(_ laneID: String, to tab: DashboardTab) async {
        guard let lane = snapshot?.lanes.first(where: { $0.id == laneID }) else { return }
        switch tab {
        case .parking where lane.attention?.state != "parked":
            await setLaneAttention(lane, state: "parked")
        case .following where lane.attention?.state == "parked":
            await setLaneAttention(lane, state: "active")
        default:
            return
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
}
