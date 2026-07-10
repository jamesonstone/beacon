import AppKit
import SwiftUI

struct MenuView: View {
    @ObservedObject var state: AppState

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            header
            if let error = state.lastError {
                errorBanner(error)
            }
            if let snapshot = state.snapshot {
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 14) {
                        laneSection("Ready for Review", symbol: "checkmark.circle.fill", lanes: state.lanes(for: snapshot.groups.ready))
                        laneSection("Needs Action", symbol: "exclamationmark.triangle.fill", lanes: state.lanes(for: snapshot.groups.action))
                        laneSection("Waiting", symbol: "clock.fill", lanes: state.lanes(for: snapshot.groups.waiting))
                        laneSection("Idle", symbol: "circle", lanes: state.lanes(for: snapshot.groups.idle))
                    }
                }
                Text("Updated \(snapshot.generatedAt)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else if state.isScanning {
                ProgressView("Scanning repositories…")
                    .frame(maxWidth: .infinity, minHeight: 180)
            } else {
                ContentUnavailableView("No scan available", systemImage: "dot.radiowaves.left.and.right")
            }
            Divider()
            actions
        }
        .padding(14)
        .frame(width: 430, height: 540)
    }

    private var header: some View {
        HStack {
            VStack(alignment: .leading) {
                Text("Beacon").font(.headline)
                Text("\(state.readyCount) ready for review").font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            if state.isScanning { ProgressView().controlSize(.small) }
        }
    }

    private var actions: some View {
        HStack {
            Button("Scan Now") { Task { await state.scan() } }
                .disabled(state.isScanning)
            Button("Open Top Item") { state.openTopItem() }
                .disabled(state.snapshot?.lanes.isEmpty ?? true)
            Button("Open Config") { state.openConfig() }
            Spacer()
            Button("Quit") { NSApplication.shared.terminate(nil) }
        }
        .buttonStyle(.link)
    }

    @ViewBuilder
    private func laneSection(_ title: String, symbol: String, lanes: [WorkLane]) -> some View {
        if !lanes.isEmpty {
            VStack(alignment: .leading, spacing: 6) {
                Label(title, systemImage: symbol).font(.subheadline.weight(.semibold))
                ForEach(state.projectGroups(for: lanes)) { project in
                    projectHeader(project)
                    ForEach(project.lanes) { lane in
                        Button { state.open(lane) } label: {
                            laneRow(lane)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
    }

    private func projectHeader(_ project: ProjectLaneGroup) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(project.name).font(.caption.weight(.semibold))
            if let progress = project.progress {
                Text("\(progress.feature) · \(actionLabel(progress.phase))")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
            Spacer()
        }
        .padding(.top, 2)
    }

    private func laneRow(_ lane: WorkLane) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            HStack {
                Text(workItemTitle(lane)).fontWeight(.medium).lineLimit(1)
                Spacer()
                if let pullRequest = lane.pullRequest {
                    Text("PR #\(pullRequest.number)").foregroundStyle(.secondary)
                } else if let issue = lane.issue {
                    Text("Issue #\(issue.number)").foregroundStyle(.secondary)
                } else if !lane.branch.isEmpty {
                    Text(lane.branch).foregroundStyle(.secondary)
                }
            }
            Text(actionLabel(lane.nextAction)).font(.caption)
            if let progress = lane.progress {
                Text("Kit \(actionLabel(progress.phase)) · \(progress.summary)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
            evidenceBadges(lane)
        }
        .padding(8)
        .background(.quaternary.opacity(0.5), in: RoundedRectangle(cornerRadius: 7))
    }

    private func evidenceBadges(_ lane: WorkLane) -> some View {
        HStack(spacing: 4) {
            badge(lane.signals.worktree)
            badge("CI \(lane.signals.ci)")
            badge("Review \(lane.signals.review)")
            badge(lane.signals.freshness)
            if let feedback = lane.pullRequest?.feedback, feedback.unresolvedThreads > 0 {
                badge("\(feedback.unresolvedThreads) unresolved", emphasized: true)
            }
        }
        .lineLimit(1)
    }

    private func badge(_ text: String, emphasized: Bool = false) -> some View {
        Text(actionLabel(text))
            .font(.caption2)
            .foregroundStyle(emphasized ? Color.red : Color.secondary)
            .padding(.horizontal, 5)
            .padding(.vertical, 2)
            .background(
                emphasized ? Color.red.opacity(0.1) : Color.secondary.opacity(0.08),
                in: Capsule()
            )
    }

    private func errorBanner(_ message: String) -> some View {
        Label(message, systemImage: "exclamationmark.triangle.fill")
            .font(.caption)
            .foregroundStyle(.red)
            .padding(8)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(.red.opacity(0.1), in: RoundedRectangle(cornerRadius: 7))
    }

    private func actionLabel(_ action: String) -> String {
        action.replacingOccurrences(of: "_", with: " ").capitalized
    }

    private func workItemTitle(_ lane: WorkLane) -> String {
        lane.pullRequest?.title
            ?? lane.issue?.title
            ?? (lane.branch.isEmpty ? lane.repository : lane.branch)
    }
}
