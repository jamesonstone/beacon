import AppKit
import SwiftUI

struct MenuView: View {
    @ObservedObject var state: AppState
    @State private var showingQuietProjects = false
    @State private var showingProjectTracking = false
    @State private var projectTrackingTab = ProjectTrackingTab.tracked
    @State private var quietSearch = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            header
            if let error = state.lastError {
                errorBanner(error)
            }
            if let reactivated = state.snapshot?.tracking?.autoReactivated, !reactivated.isEmpty {
                reactivationBanner(reactivated)
            }
            if let snapshot = state.snapshot {
                if showingProjectTracking {
                    ProjectTrackingView(state: state, selectedTab: $projectTrackingTab) {
                        showingProjectTracking = false
                    }
                } else if showingQuietProjects {
                    quietProjects
                } else {
                    activeDashboard(snapshot)
                }
                Text("Updated \(snapshot.generatedAt)")
                    .font(.caption2)
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.9))
            } else if state.isScanning {
                ProgressView("Scanning repositories…")
                    .tint(BeaconPalette.cyan)
                    .frame(maxWidth: .infinity, minHeight: 180)
            } else {
                ContentUnavailableView("No scan available", systemImage: "dot.radiowaves.left.and.right")
                    .symbolRenderingMode(.palette)
                    .foregroundStyle(BeaconPalette.cyan, BeaconPalette.lavender)
            }
            Divider()
            actions
        }
        .padding(14)
        .frame(width: 430, height: 540)
        .background(BeaconPalette.panelBackground)
    }

    private var header: some View {
        HStack {
            HStack(spacing: 8) {
                Image(systemName: "sparkles")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(BeaconPalette.neonGradient)
                    .shadow(color: BeaconPalette.cyan.opacity(0.55), radius: 2)
                VStack(alignment: .leading, spacing: 3) {
                    Text("Beacon")
                        .font(.headline)
                        .foregroundStyle(BeaconPalette.neonGradient)
                    Text("\(state.readyCount) ready for review")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(BeaconPalette.mint)
                }
            }
            Spacer()
            if state.isScanning {
                ProgressView()
                    .controlSize(.small)
                    .tint(BeaconPalette.cyan)
            }
        }
    }

    private var actions: some View {
        HStack {
            Button { Task { await state.scan() } } label: {
                Label("Scan Now", systemImage: "arrow.clockwise")
            }
            .tint(BeaconPalette.cyan)
            .disabled(state.isScanning || state.isProjectMutationInProgress)
            Button { state.openTopItem() } label: {
                Label("Open Top", systemImage: "arrow.up.forward.app")
            }
            .tint(BeaconPalette.mint)
            .disabled(state.inProgressCount == 0)
            Button { state.openConfig() } label: {
                Label("Config", systemImage: "slider.horizontal.3")
            }
            .tint(BeaconPalette.lavender)
            Button {
                showingQuietProjects = false
                projectTrackingTab = .tracked
                showingProjectTracking = true
            } label: {
                Label("Projects", systemImage: "checklist")
            }
            .tint(BeaconPalette.gold)
            Spacer()
            Button { NSApplication.shared.terminate(nil) } label: {
                Image(systemName: "power")
            }
            .tint(BeaconPalette.pink)
            .help("Quit Beacon")
        }
        .buttonStyle(.link)
    }

    private func activeDashboard(_ snapshot: BeaconSnapshot) -> some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 14) {
                laneSection(
                    "Ready for Review",
                    symbol: "checkmark.circle.fill",
                    accent: BeaconPalette.mint,
                    lanes: state.lanes(for: snapshot.groups.ready)
                )
                laneSection(
                    "Needs Action",
                    symbol: "exclamationmark.triangle.fill",
                    accent: BeaconPalette.coral,
                    lanes: state.lanes(for: snapshot.groups.action)
                )
                laneSection(
                    "Waiting",
                    symbol: "clock.fill",
                    accent: BeaconPalette.gold,
                    lanes: state.lanes(for: snapshot.groups.waiting)
                )
                if state.quietProjectCount > 0 {
                    Button {
                        quietSearch = ""
                        showingQuietProjects = true
                    } label: {
                        HStack(spacing: 9) {
                            Image(systemName: "moon.stars.fill")
                                .foregroundStyle(BeaconPalette.lavender)
                                .shadow(color: BeaconPalette.lavender.opacity(0.45), radius: 2)
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Quiet Projects")
                                    .font(.subheadline.weight(.semibold))
                                Text("\(state.quietProjectCount) idle project\(state.quietProjectCount == 1 ? "" : "s")")
                                    .font(.caption2)
                                    .foregroundStyle(BeaconPalette.lavender.opacity(0.85))
                            }
                            Spacer()
                            Image(systemName: "chevron.right")
                                .foregroundStyle(BeaconPalette.cyan)
                        }
                        .padding(10)
                        .background(BeaconPalette.softGradient(BeaconPalette.lavender), in: RoundedRectangle(cornerRadius: 9))
                        .overlay {
                            RoundedRectangle(cornerRadius: 9)
                                .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.lavender), lineWidth: 0.8)
                        }
                    }
                    .buttonStyle(.plain)
                }
                if state.untrackedProjectCount > 0 {
                    Button {
                        projectTrackingTab = .untracked
                        showingProjectTracking = true
                    } label: {
                        HStack(spacing: 9) {
                            Image(systemName: "eye.slash.fill")
                                .foregroundStyle(BeaconPalette.pink)
                            Text("\(state.untrackedProjectCount) Untracked Project\(state.untrackedProjectCount == 1 ? "" : "s")")
                                .font(.subheadline.weight(.semibold))
                            Spacer()
                            Image(systemName: "chevron.right")
                                .foregroundStyle(BeaconPalette.cyan)
                        }
                        .padding(10)
                        .background(BeaconPalette.softGradient(BeaconPalette.pink), in: RoundedRectangle(cornerRadius: 9))
                        .overlay {
                            RoundedRectangle(cornerRadius: 9)
                                .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.pink), lineWidth: 0.8)
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private var quietProjects: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Button {
                    showingQuietProjects = false
                    quietSearch = ""
                } label: {
                    Label("Dashboard", systemImage: "chevron.left")
                }
                .buttonStyle(.plain)
                .foregroundStyle(BeaconPalette.cyan)
                Spacer()
                Text("\(state.quietProjectCount) quiet")
                    .font(.caption.weight(.medium))
                    .foregroundStyle(BeaconPalette.lavender)
            }
            TextField("Search quiet projects", text: $quietSearch)
                .textFieldStyle(.roundedBorder)
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 8) {
                    let groups = state.quietProjectGroups(matching: quietSearch)
                    if groups.isEmpty {
                        ContentUnavailableView.search(text: quietSearch)
                            .foregroundStyle(BeaconPalette.lavender)
                    } else {
                        ForEach(groups) { project in
                            projectHeader(project, accent: BeaconPalette.lavender)
                            ForEach(project.lanes) { lane in
                                Button { state.open(lane) } label: {
                                    laneRow(lane, accent: BeaconPalette.lavender)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func laneSection(_ title: String, symbol: String, accent: Color, lanes: [WorkLane]) -> some View {
        if !lanes.isEmpty {
            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 6) {
                    Image(systemName: symbol)
                        .foregroundStyle(accent)
                        .shadow(color: accent.opacity(0.45), radius: 2)
                    Text(title)
                        .foregroundStyle(BeaconPalette.borderGradient(accent))
                }
                .font(.subheadline.weight(.semibold))
                ForEach(state.projectGroups(for: lanes)) { project in
                    projectHeader(project, accent: accent)
                    ForEach(project.lanes) { lane in
                        Button { state.open(lane) } label: {
                            laneRow(lane, accent: accent)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
    }

    private func projectHeader(_ project: ProjectLaneGroup, accent: Color) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(project.name)
                .font(.caption.weight(.semibold))
                .foregroundStyle(BeaconPalette.borderGradient(accent))
            if let progress = project.progress {
                Text("\(progress.feature) · \(actionLabel(progress.phase))")
                    .font(.caption2)
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.85))
                    .lineLimit(1)
            }
            Spacer()
        }
        .padding(.top, 2)
    }

    private func laneRow(_ lane: WorkLane, accent: Color) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            HStack {
                Text(workItemTitle(lane))
                    .fontWeight(.medium)
                    .lineLimit(1)
                Spacer()
                if let pullRequest = lane.pullRequest {
                    Text("PR #\(pullRequest.number)").foregroundStyle(accent)
                } else if let issue = lane.issue {
                    Text("Issue #\(issue.number)").foregroundStyle(accent)
                } else if !lane.branch.isEmpty {
                    Text(lane.branch).foregroundStyle(accent)
                }
            }
            Text(actionLabel(lane.nextAction))
                .font(.caption.weight(.medium))
                .foregroundStyle(accent)
            if let progress = lane.progress {
                Text("Kit \(actionLabel(progress.phase)) · \(progress.summary)")
                    .font(.caption2)
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.85))
                    .lineLimit(1)
            }
            evidenceBadges(lane)
        }
        .padding(8)
        .background(BeaconPalette.softGradient(accent), in: RoundedRectangle(cornerRadius: 8))
        .overlay {
            RoundedRectangle(cornerRadius: 8)
                .strokeBorder(BeaconPalette.borderGradient(accent), lineWidth: 0.8)
        }
        .shadow(color: accent.opacity(0.09), radius: 4, y: 2)
        .contentShape(RoundedRectangle(cornerRadius: 8))
    }

    private func evidenceBadges(_ lane: WorkLane) -> some View {
        HStack(spacing: 4) {
            badge(lane.signals.worktree, accent: signalColor(lane.signals.worktree))
            badge("CI \(lane.signals.ci)", accent: signalColor(lane.signals.ci))
            badge("Review \(lane.signals.review)", accent: signalColor(lane.signals.review))
            badge(lane.signals.freshness, accent: signalColor(lane.signals.freshness))
            if let feedback = lane.pullRequest?.feedback, feedback.unresolvedThreads > 0 {
                badge("\(feedback.unresolvedThreads) unresolved", accent: BeaconPalette.pink, emphasized: true)
            }
        }
        .lineLimit(1)
    }

    private func badge(_ text: String, accent: Color, emphasized: Bool = false) -> some View {
        Text(actionLabel(text))
            .font(.caption2.weight(.medium))
            .foregroundStyle(accent)
            .padding(.horizontal, 5)
            .padding(.vertical, 2)
            .background(
                BeaconPalette.softGradient(accent),
                in: Capsule()
            )
            .overlay {
                Capsule()
                    .strokeBorder(accent.opacity(emphasized ? 0.8 : 0.34), lineWidth: 0.6)
            }
            .shadow(color: emphasized ? accent.opacity(0.28) : .clear, radius: 2)
    }

    private func errorBanner(_ message: String) -> some View {
        Label(message, systemImage: "exclamationmark.triangle.fill")
            .font(.caption)
            .foregroundStyle(BeaconPalette.pink)
            .padding(8)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(BeaconPalette.softGradient(BeaconPalette.pink), in: RoundedRectangle(cornerRadius: 8))
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.pink), lineWidth: 0.8)
            }
    }

    private func reactivationBanner(_ projects: [String]) -> some View {
        Label("Automatically tracking \(projects.joined(separator: ", "))", systemImage: "bolt.fill")
            .font(.caption)
            .foregroundStyle(BeaconPalette.mint)
            .padding(8)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(BeaconPalette.softGradient(BeaconPalette.mint), in: RoundedRectangle(cornerRadius: 8))
    }

    private func signalColor(_ signal: String) -> Color {
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

    private func actionLabel(_ action: String) -> String {
        action.replacingOccurrences(of: "_", with: " ").capitalized
    }

    private func workItemTitle(_ lane: WorkLane) -> String {
        lane.pullRequest?.title
            ?? lane.issue?.title
            ?? (lane.branch.isEmpty ? lane.repository : lane.branch)
    }
}
