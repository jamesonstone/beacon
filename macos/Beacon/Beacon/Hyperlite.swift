import Foundation
import SwiftUI

struct HyperliteItem: Equatable, Identifiable {
    let lane: WorkLane
    let projectName: String
    let group: HyperliteGroup
    let attention: Bool
    let ageDate: Date?
    let workingDate: Date?

    var id: String { lane.id }
}

enum HyperliteGroup: String, Equatable {
    case active
    case waiting
    case recent
}

enum HyperlitePresentation {
    static func items(snapshot: BeaconSnapshot, activity: ExternalActivitySnapshot) -> [HyperliteItem] {
        let projects = Dictionary(uniqueKeysWithValues: snapshot.projects.map { ($0.github, $0.name) })
        let lanes = Dictionary(uniqueKeysWithValues: snapshot.lanes.map { ($0.id, $0) })
        let groups: [(HyperliteGroup, [String])] = [
            (.active, snapshot.workingSet?.active ?? snapshot.groups.ready),
            (.waiting, snapshot.workingSet?.waiting ?? snapshot.groups.waiting),
            (.recent, snapshot.workingSet?.recent ?? snapshot.groups.action),
        ]
        var seen = Set<String>()
        var result: [HyperliteItem] = []
        for (group, ids) in groups {
            for id in ids {
                guard seen.insert(id).inserted, let lane = lanes[id] else { continue }
                let records = activity.records.filter { $0.laneID == lane.id || ($0.laneID == nil && $0.projectID == lane.github) }
                let workingDate = records
                    .filter { $0.state == "working" }
                    .compactMap { parseDate($0.observedAt) }
                    .min()
                result.append(HyperliteItem(
                    lane: lane,
                    projectName: projects[lane.github] ?? lane.repository,
                    group: group,
                    attention: needsAttention(lane),
                    ageDate: parseDate(lane.updatedAt),
                    workingDate: workingDate
                ))
            }
        }
        return result.sorted { lhs, rhs in
            if lhs.attention != rhs.attention { return lhs.attention && !rhs.attention }
            if lhs.group != rhs.group { return groupRank(lhs.group) < groupRank(rhs.group) }
            return (lhs.ageDate ?? .distantPast) > (rhs.ageDate ?? .distantPast)
        }
    }

    static func needsAttention(_ lane: WorkLane) -> Bool {
        if lane.reviewReady || !lane.blockers.isEmpty || !lane.warnings.isEmpty { return true }
        let action = lane.nextAction.lowercased()
        let waitingValues = ["wait", "waiting", "monitor", "none", "idle", "unknown"]
        return !waitingValues.contains { action.contains($0) }
    }

    static func ageLabel(for date: Date?, now: Date = Date()) -> String {
        guard let date else { return "age unknown" }
        let seconds = max(0, Int(now.timeIntervalSince(date)))
        if seconds < 60 { return "now" }
        if seconds < 3_600 { return "\(seconds / 60)m" }
        if seconds < 86_400 { return "\(seconds / 3_600)h" }
        return "\(seconds / 86_400)d"
    }

    static func groupLabel(_ group: HyperliteGroup) -> String {
        switch group {
        case .active: "Active"
        case .waiting: "Waiting"
        case .recent: "Recently active"
        }
    }

    private static func groupRank(_ group: HyperliteGroup) -> Int {
        switch group {
        case .active: 0
        case .waiting: 1
        case .recent: 2
        }
    }

    private static func parseDate(_ value: String) -> Date? {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.date(from: value) ?? ISO8601DateFormatter().date(from: value)
    }
}

struct HyperliteView: View {
    @Environment(\.beaconTheme) private var theme
    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor
    @ObservedObject var state: AppState
    let openDashboard: () -> Void

    private var items: [HyperliteItem] {
        guard let snapshot = state.snapshot else { return [] }
        return HyperlitePresentation.items(snapshot: snapshot, activity: state.externalActivity)
    }

    private var attentionItems: [HyperliteItem] { items.filter(\.attention) }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            header
            if state.lastError != nil {
                statusLine("Evidence refresh reported an error", symbol: "exclamationmark.triangle.fill", color: theme.tokens.danger.color)
            }
            if state.isScanning {
                statusLine("Refreshing evidence…", symbol: "arrow.clockwise", color: theme.tokens.info.color)
            }
            if let snapshot = state.snapshot {
                if attentionItems.isEmpty {
                    statusLine("Nothing needs attention", symbol: "checkmark.circle.fill", color: theme.tokens.success.color)
                } else {
                    section("Needs attention", count: attentionItems.count, symbol: "exclamationmark.triangle.fill", color: theme.tokens.warning.color, items: attentionItems)
                }
                let otherItems = items.filter { !$0.attention }
                if !otherItems.isEmpty {
                    section("Other active work", count: otherItems.count, symbol: "bolt.fill", color: theme.tokens.info.color, items: otherItems)
                }
                if items.isEmpty {
                    statusLine("No active work in the current snapshot", symbol: "moon.stars", color: theme.tokens.textMuted.color)
                }
                Text("Evidence updated \(HyperlitePresentation.ageLabel(for: HyperlitePresentationDate.parse(snapshot.generatedAt)))")
                    .font(BeaconTypography.identifier(8))
                    .foregroundStyle(theme.tokens.textMuted.color)
            } else if !state.isScanning {
                statusLine("No scan available", symbol: "dot.radiowaves.left.and.right", color: theme.tokens.info.color)
            }
            footer
        }
        .padding(12)
        .frame(width: 360)
        .background(theme.tokens.canvas.color)
        .environment(\.beaconTheme, theme)
        .font(BeaconTypography.regular(10))
        .foregroundStyle(theme.tokens.textPrimary.color)
        .symbolRenderingMode(differentiateWithoutColor ? .monochrome : .hierarchical)
    }

    private var header: some View {
        HStack(spacing: 7) {
            BeaconRocketMark()
            VStack(alignment: .leading, spacing: 1) {
                Text("Hyperlite").font(BeaconTypography.bold(16))
                Text("What needs your attention?")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(theme.tokens.textSecondary.color)
            }
            Spacer()
            Text("\(items.count)")
                .font(BeaconTypography.counter(16, weight: .heavy))
                .foregroundStyle(attentionItems.isEmpty ? theme.tokens.success.color : theme.tokens.warning.color)
                .accessibilityLabel("\(attentionItems.count) items need attention, \(items.count) active items total")
        }
    }

    private func section(_ title: String, count: Int, symbol: String, color: Color, items: [HyperliteItem]) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            Label("\(title) · \(count)", systemImage: symbol)
                .font(BeaconTypography.semibold(10))
                .foregroundStyle(color)
            ForEach(items) { item in row(item) }
        }
    }

    private func row(_ item: HyperliteItem) -> some View {
        Button { state.open(item.lane) } label: {
            HStack(alignment: .top, spacing: 7) {
                Circle()
                    .fill(item.attention ? theme.tokens.warning.color : theme.tokens.info.color)
                    .frame(width: 7, height: 7)
                    .padding(.top, 4)
                VStack(alignment: .leading, spacing: 2) {
                    HStack(spacing: 5) {
                        Text(item.projectName).font(BeaconTypography.semibold(10)).lineLimit(1)
                        Text(HyperlitePresentation.groupLabel(item.group))
                            .font(BeaconTypography.identifier(8))
                            .foregroundStyle(theme.tokens.textMuted.color)
                        Spacer()
                        Text(HyperlitePresentation.ageLabel(for: item.ageDate))
                            .font(BeaconTypography.identifier(8))
                            .foregroundStyle(theme.tokens.textMuted.color)
                    }
                    Text(workTitle(item.lane))
                        .font(BeaconTypography.regular(9))
                        .lineLimit(1)
                    HStack(spacing: 5) {
                        Text(actionLabel(item.lane.nextAction))
                            .font(BeaconTypography.medium(9))
                            .foregroundStyle(item.attention ? theme.tokens.warning.color : theme.tokens.textSecondary.color)
                        if let workingDate = item.workingDate {
                            Text("· working \(HyperlitePresentation.ageLabel(for: workingDate))")
                                .font(BeaconTypography.identifier(8))
                                .foregroundStyle(theme.tokens.info.color)
                                .help("External activity has been observed for this long; Beacon does not infer a task start time.")
                        }
                    }
                }
            }
            .padding(.vertical, 3)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel("\(item.projectName), \(workTitle(item.lane)), \(actionLabel(item.lane.nextAction))")
        .accessibilityHint("Opens the work item")
    }

    private var footer: some View {
        HStack(spacing: 7) {
            Button { Task { await state.scan() } } label: {
                Label(state.isScanning ? "Refreshing" : "Refresh", systemImage: "arrow.clockwise")
            }
            .buttonStyle(.bordered)
            .controlSize(.small)
            .disabled(state.isScanning)
            Button("Open full dashboard", action: openDashboard)
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
            Spacer()
        }
    }

    private func statusLine(_ title: String, symbol: String, color: Color) -> some View {
        Label(title, systemImage: symbol)
            .font(BeaconTypography.medium(9))
            .foregroundStyle(color)
            .padding(.vertical, 3)
    }

    private func workTitle(_ lane: WorkLane) -> String {
        lane.pullRequest?.title ?? lane.issue?.title ?? (lane.branch.isEmpty ? lane.repository : lane.branch)
    }

    private func actionLabel(_ action: String) -> String {
        action.replacingOccurrences(of: "_", with: " ").capitalized
    }
}

private enum HyperlitePresentationDate {
    static func parse(_ value: String) -> Date? {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.date(from: value) ?? ISO8601DateFormatter().date(from: value)
    }
}
