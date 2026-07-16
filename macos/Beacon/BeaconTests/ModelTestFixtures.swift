import Foundation
@testable import Beacon

func externalActivityRecord(
    provider: String,
    state: String,
    session: String,
    project: String = "owner/repo",
    lane: String? = "lane-31"
) -> ExternalActivityRecord {
    ExternalActivityRecord(
        provider: provider,
        state: state,
        sessionKey: session,
        projectID: project,
        laneID: lane,
        observedAt: "2026-07-16T12:00:00Z",
        expiresAt: "2026-07-16T13:00:00Z"
    )
}
