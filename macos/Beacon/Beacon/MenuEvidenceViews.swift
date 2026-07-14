import Foundation
import SwiftUI

extension MenuView {
    func evidenceBadges(_ lane: WorkLane) -> some View {
        HStack(spacing: 4) {
            dismissibleBadge(lane, dimension: "worktree", value: lane.signals.worktree, text: lane.signals.worktree, accent: signalColor(lane.signals.worktree))
            dismissibleBadge(lane, dimension: "ci", value: lane.signals.ci, text: "CI \(lane.signals.ci)", accent: signalColor(lane.signals.ci))
            dismissibleBadge(lane, dimension: "review", value: lane.signals.review, text: "Review \(lane.signals.review)", accent: signalColor(lane.signals.review))
            dismissibleBadge(lane, dimension: "freshness", value: lane.signals.freshness, text: lane.signals.freshness, accent: signalColor(lane.signals.freshness))
            if let feedback = lane.pullRequest?.feedback, feedback.unresolvedThreads > 0 {
                dismissibleBadge(
                    lane,
                    dimension: "unresolved-feedback",
                    value: String(feedback.unresolvedThreads),
                    text: "\(feedback.unresolvedThreads) unresolved",
                    accent: BeaconPalette.pink,
                    emphasized: true
                )
            }
        }
        .lineLimit(1)
    }

    @ViewBuilder
    func dismissibleBadge(
        _ lane: WorkLane,
        dimension: String,
        value: String,
        text: String,
        accent: Color,
        emphasized: Bool = false
    ) -> some View {
        let key = EvidenceBadgeDismissals.key(laneID: lane.id, dimension: dimension, value: value)
        if !dismissedEvidenceBadges.contains(key) {
            DismissibleEvidenceBadge(text: actionLabel(text), accent: accent, emphasized: emphasized) {
                dismissEvidenceBadge(key)
            }
        }
    }

    func timeSinceActivity(_ value: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let date = formatter.date(from: value) ?? ISO8601DateFormatter().date(from: value)
        guard let date else { return "activity unknown" }
        return RelativeDateTimeFormatter().localizedString(for: date, relativeTo: Date())
    }

    func errorBanner(_ message: String) -> some View {
        Label(message, systemImage: "exclamationmark.triangle.fill")
            .font(BeaconTypography.regular(10))
            .foregroundStyle(BeaconPalette.pink)
            .padding(8)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(BeaconPalette.softGradient(BeaconPalette.pink), in: RoundedRectangle(cornerRadius: 8))
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.pink), lineWidth: 0.8)
            }
    }

    func signalColor(_ signal: String) -> Color {
        switch signal.lowercased() {
        case "clean", "success", "approved", "current", "published", "open":
            BeaconPalette.mint
        case "pending", "review_required", "draft", "behind", "none", "not_local":
            BeaconPalette.gold
        case "failure", "failed", "dirty", "conflicted", "conflicting", "changes_requested":
            BeaconPalette.coral
        case "stale", "diverged", "unknown", "unavailable":
            BeaconPalette.pink
        default:
            BeaconPalette.cyan
        }
    }

    func actionLabel(_ action: String) -> String {
        action.replacingOccurrences(of: "_", with: " ").capitalized
    }

    func workItemTitle(_ lane: WorkLane) -> String {
        lane.pullRequest?.title
            ?? lane.issue?.title
            ?? (lane.branch.isEmpty ? lane.repository : lane.branch)
    }
}
