import Foundation
import SwiftUI

private struct EvidenceBadgeDescriptor: Identifiable {
    let id: String
    let value: String
    let text: String
    let symbol: String
    let accent: Color
    let help: String
    let emphasized: Bool
    let feedback: FeedbackSummary?
}

extension MenuView {
    func evidenceBadges(_ lane: WorkLane, condensed: Bool = false) -> some View {
        let descriptors = evidenceBadgeDescriptors(lane)
        return HStack(spacing: 4) {
            ForEach(Array(descriptors.prefix(condensed ? 1 : descriptors.count))) { descriptor in
                dismissibleBadge(
                    lane,
                    dimension: descriptor.id,
                    value: descriptor.value,
                    text: descriptor.text,
                    symbol: descriptor.symbol,
                    accent: descriptor.accent,
                    help: descriptor.help,
                    emphasized: descriptor.emphasized,
                    feedback: descriptor.feedback
                )
            }
        }
        .lineLimit(1)
    }

    private func evidenceBadgeDescriptors(_ lane: WorkLane) -> [EvidenceBadgeDescriptor] {
        var result: [EvidenceBadgeDescriptor] = []
        func add(_ id: String, _ value: String, _ text: String, _ symbol: String, _ accent: Color, _ help: String, emphasized: Bool = false, feedback: FeedbackSummary? = nil) {
            result.append(EvidenceBadgeDescriptor(
                id: id, value: value, text: text, symbol: symbol,
                accent: accent, help: help, emphasized: emphasized, feedback: feedback
            ))
        }

        switch lane.signals.worktree {
        case "conflicted": add("worktree", "conflicted", "Local conflicts", "exclamationmark.triangle.fill", BeaconPalette.coral, "The local worktree has unresolved merge conflicts.", emphasized: true)
        case "dirty": add("worktree", "dirty", "Local changes", "pencil.and.outline", BeaconPalette.gold, "The local worktree has staged, unstaged, or untracked changes.")
        case "unavailable": add("worktree", "unavailable", "Local unavailable", "externaldrive.badge.questionmark", BeaconPalette.pink, "Beacon could not inspect the local worktree; refresh or inspect the repository.")
        default: break
        }
        switch lane.signals.merge {
        case "conflicting": add("merge", "conflicting", "Merge conflict", "arrow.triangle.merge", BeaconPalette.coral, "GitHub reports that this pull request conflicts with its base branch.", emphasized: true)
        case "blocked": add("merge", "blocked", "Merge blocked", "lock.trianglebadge.exclamationmark", BeaconPalette.coral, "GitHub reports that this pull request cannot merge yet.")
        default: break
        }
        switch lane.signals.ci {
        case "failure": add("ci", "failure", "CI failed", "xmark.octagon.fill", BeaconPalette.coral, "One or more pull request checks failed.", emphasized: true)
        case "pending": add("ci", "pending", "CI pending", "clock.fill", BeaconPalette.gold, "Pull request checks are still queued or running.")
        case "unknown": add("ci", "unknown", "CI unknown", "questionmark.diamond.fill", BeaconPalette.pink, "Beacon could not determine the current pull request check state.")
        default: break
        }
        if let feedback = lane.pullRequest?.feedback, feedback.unresolvedThreads > 0 {
            add(
                "pr-feedback", String(feedback.unresolvedThreads),
                EvidenceTaxonomy.pullRequestFeedbackLabel(feedback.unresolvedThreads), "text.bubble.fill", BeaconPalette.pink,
                "\(feedback.unresolvedThreads) unresolved pull request review \(feedback.unresolvedThreads == 1 ? "thread" : "threads"). Hover for the file, comment text, author, timestamp, and individual GitHub links.",
                emphasized: true, feedback: feedback
            )
        } else {
            switch lane.signals.review {
            case "changes_requested", "feedback_pending": add("review", lane.signals.review, "Review changes", "person.crop.circle.badge.exclamationmark", BeaconPalette.pink, "A reviewer requested changes that still need attention.", emphasized: true)
            case "review_required": add("review", "review_required", "Review needed", "person.crop.circle.badge.clock", BeaconPalette.gold, "This pull request still needs a review.")
            case "unknown": add("review", "unknown", "Review unknown", "person.crop.circle.badge.questionmark", BeaconPalette.pink, "Beacon could not determine the current review state.")
            default: break
            }
        }
        switch lane.signals.publication {
        case "no_upstream": add("publication", "no_upstream", "No upstream", "arrow.up.to.line.compact", BeaconPalette.gold, "The local branch has no configured upstream branch.")
        case "unpushed": add("publication", "unpushed", "Not pushed", "arrow.up.circle.fill", BeaconPalette.gold, "The local branch has commits that are not on its upstream branch.")
        case "behind": add("publication", "behind", "Behind remote", "arrow.down.circle.fill", BeaconPalette.gold, "The local branch is behind its upstream branch.")
        case "diverged": add("publication", "diverged", "Branch diverged", "arrow.triangle.branch", BeaconPalette.coral, "The local branch and its upstream both have unique commits.", emphasized: true)
        case "unknown": add("publication", "unknown", "Publish unknown", "questionmark.circle.fill", BeaconPalette.pink, "Beacon could not determine whether this branch is published.")
        default: break
        }
        if lane.signals.freshness == "stale" {
            add("freshness", "stale", "Stale", "clock.badge.exclamationmark.fill", BeaconPalette.pink, "The evidence has not materially changed within Beacon's freshness window.")
        }
        return result
    }

    @ViewBuilder
    func dismissibleBadge(
        _ lane: WorkLane,
        dimension: String,
        value: String,
        text: String,
        symbol: String,
        accent: Color,
        help: String,
        emphasized: Bool = false,
        feedback: FeedbackSummary? = nil
    ) -> some View {
        let key = EvidenceBadgeDismissals.key(laneID: lane.id, dimension: dimension, value: value)
        if !dismissedEvidenceBadges.contains(key) {
            if let feedback {
                DismissibleEvidenceBadge(text: text, symbol: symbol, accent: accent, emphasized: emphasized) {
                    dismissEvidenceBadge(key)
                }
                .help(help)
                .onHover { hovered in
                    if hovered {
                        evidenceHoverLaneID = lane.id
                    } else if evidenceHoverLaneID == lane.id {
                        evidenceHoverLaneID = nil
                    }
                }
                .richHoverPopover { reviewFeedbackPanel(lane, feedback: feedback) }
            } else {
                DismissibleEvidenceBadge(text: text, symbol: symbol, accent: accent, emphasized: emphasized) {
                    dismissEvidenceBadge(key)
                }
                .help(help)
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
