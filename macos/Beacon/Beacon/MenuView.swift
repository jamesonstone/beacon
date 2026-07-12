import AppKit
import SwiftUI

enum DashboardSurface {
    case menu
    case window
}

struct MenuView: View {
    @ObservedObject var state: AppState
    @ObservedObject var loginItem: LoginItemController
    let surface: DashboardSurface
    let openDashboard: () -> Void
    @State private var showingQuietProjects = false
    @State private var showingParkedLanes = false
    @State private var showingProjectTracking = false
    @State private var projectTrackingTab = ProjectTrackingTab.tracked
    @State private var quietSearch = ""
    @State private var noteLane: WorkLane?
    @State private var noteText = ""
    @State private var showingNoteEditor = false
    @State private var manualTitle = ""
    @State private var showingManualEditor = false

    var body: some View {
        Group {
            if surface == .menu {
                dashboard
                    .frame(width: 430, height: 540)
            } else {
                dashboard
                    .frame(
                        minWidth: 430,
                        maxWidth: .infinity,
                        minHeight: 540,
                        maxHeight: .infinity
                    )
            }
        }
        .onAppear { loginItem.refresh() }
        .alert("Lane note", isPresented: $showingNoteEditor) {
            TextField("Short context note", text: $noteText)
            Button("Save") { if let lane = noteLane { Task { await state.setLaneNote(lane, note: noteText) } } }
            Button("Cancel", role: .cancel) {}
        }
        .alert("Add manual lane", isPresented: $showingManualEditor) {
            TextField("Planning or research lane", text: $manualTitle)
            Button("Add") { Task { await state.addManualLane(manualTitle) } }
            Button("Cancel", role: .cancel) {}
        }
    }

    private var dashboard: some View {
        VStack(alignment: .leading, spacing: 12) {
            header
            if let error = state.lastError {
                errorBanner(error)
            }
            if let error = loginItem.errorMessage {
                errorBanner("Open at Login: \(error)")
            }
            if !state.agentAvailable {
                enableAgentBanner
            }
            if let message = state.reactivationMessage {
                Label(message, systemImage: "bolt.fill")
                    .font(.caption)
                    .foregroundStyle(BeaconPalette.mint)
            }
            if let reactivated = state.snapshot?.tracking?.autoReactivated, !reactivated.isEmpty {
                reactivationBanner(reactivated)
            }
            if let snapshot = state.snapshot {
                Picker("Projects", selection: $projectTrackingTab) {
                    ForEach(ProjectTrackingTab.allCases) { tab in
                        Text(tab.rawValue).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                if projectTrackingTab == .untracked {
                    ProjectTrackingView(state: state, selectedTab: $projectTrackingTab, onClose: {
                        projectTrackingTab = .tracked
                    }, showsTabPicker: false)
                } else if showingProjectTracking {
                    ProjectTrackingView(state: state, selectedTab: $projectTrackingTab, onClose: {
                        showingProjectTracking = false
                    })
                } else if showingParkedLanes {
                    parkedLanes(snapshot)
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
            loginItemControls
            Divider()
            actions
        }
        .padding(14)
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
                    Text("\(state.inProgressCount) lanes in focus")
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
            if surface == .menu {
                Button(action: openDashboard) {
                    Label("Window", systemImage: "macwindow")
                }
                .tint(BeaconPalette.pink)
                .help("Open Beacon Dashboard")
            }
            Button { Task { await state.scan() } } label: {
                Label("Scan Now", systemImage: "arrow.clockwise")
            }
            .tint(BeaconPalette.cyan)
            .disabled(state.isScanning)
            Button { state.openTopItem() } label: {
                Label("Open Top", systemImage: "arrow.up.forward.app")
            }
            .tint(BeaconPalette.mint)
            .disabled(state.inProgressCount == 0)
            Button { manualTitle = ""; showingManualEditor = true } label: {
                Label("Add Lane", systemImage: "plus.circle")
            }
            .tint(BeaconPalette.pink)
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

    private var loginItemControls: some View {
        HStack(spacing: 8) {
            Toggle(
                "Open Beacon at Login",
                isOn: Binding(
                    get: { loginItem.isEnabled },
                    set: { loginItem.setEnabled($0) }
                )
            )
            .toggleStyle(.switch)
            .controlSize(.small)
            .tint(BeaconPalette.cyan)
            Spacer()
            if loginItem.requiresApproval {
                Button("Approve in Settings") {
                    loginItem.openSystemSettings()
                }
                .buttonStyle(.link)
                .font(.caption)
                .foregroundStyle(BeaconPalette.gold)
            }
        }
        .font(.caption.weight(.medium))
        .foregroundStyle(loginItem.requiresApproval ? BeaconPalette.gold : BeaconPalette.lavender)
    }

    private var enableAgentBanner: some View {
        HStack {
            Label("Background agent unavailable", systemImage: "antenna.radiowaves.left.and.right.slash")
                .font(.caption)
                .foregroundStyle(BeaconPalette.gold)
            Spacer()
            Button("Enable") { Task { await state.enableAgent() } }
                .buttonStyle(.bordered)
                .tint(BeaconPalette.cyan)
        }
        .padding(8)
        .background(BeaconPalette.softGradient(BeaconPalette.gold), in: RoundedRectangle(cornerRadius: 8))
    }

    private func activeDashboard(_ snapshot: BeaconSnapshot) -> some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 14) {
                if !state.loadingProjects.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Label("Loading Projects", systemImage: "antenna.radiowaves.left.and.right")
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(BeaconPalette.borderGradient(BeaconPalette.cyan))
                        ForEach(state.loadingProjects, id: \.projectID) { project in
                            HStack(spacing: 8) {
                                ProgressView()
                                    .controlSize(.mini)
                                    .tint(BeaconPalette.cyan)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(project.name)
                                        .font(.caption.weight(.semibold))
                                    Text(stageLabel(project.stage))
                                        .font(.caption2)
                                        .foregroundStyle(BeaconPalette.lavender)
                                }
                                Spacer()
                                Text(project.projectID)
                                    .font(.caption2)
                                    .foregroundStyle(BeaconPalette.cyan.opacity(0.85))
                            }
                            .padding(8)
                            .background(BeaconPalette.softGradient(BeaconPalette.cyan), in: RoundedRectangle(cornerRadius: 8))
                        }
                    }
                }
                if let working = snapshot.workingSet {
                    laneSection("Active", symbol: "bolt.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: working.active))
                    laneSection("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: working.waiting))
                    laneSection("Recently Active", symbol: "sparkles", accent: BeaconPalette.cyan, lanes: state.lanes(for: working.recent))
                    if !working.parked.isEmpty {
                        Button {
                            showingParkedLanes = true
                        } label: {
                            HStack(spacing: 9) {
                                Image(systemName: "pause.circle.fill")
                                    .foregroundStyle(BeaconPalette.lavender)
                                Text("\(working.parked.count) Parked Lane\(working.parked.count == 1 ? "" : "s")")
                                    .font(.subheadline.weight(.semibold))
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .foregroundStyle(BeaconPalette.cyan)
                            }
                            .padding(10)
                            .background(BeaconPalette.softGradient(BeaconPalette.lavender), in: RoundedRectangle(cornerRadius: 9))
                        }
                        .buttonStyle(.plain)
                    }
                } else {
                    laneSection("Ready for Review", symbol: "checkmark.circle.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: snapshot.groups.ready))
                    laneSection("Needs Action", symbol: "exclamationmark.triangle.fill", accent: BeaconPalette.coral, lanes: state.lanes(for: snapshot.groups.action))
                    laneSection("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: snapshot.groups.waiting))
                }
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

    private func stageLabel(_ stage: String) -> String {
        switch stage {
        case "queued": return "Queued"
        case "local": return "Checking local Git"
        case "github": return "Checking GitHub"
        case "failed": return "Refresh failed — showing previous result"
        case "ready": return "Ready"
        default: return "Cached"
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
                                .contextMenu { laneActions(lane) }
                            }
                        }
                    }
                }
            }
        }
    }

    private func parkedLanes(_ snapshot: BeaconSnapshot) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Button {
                    showingParkedLanes = false
                } label: {
                    Label("Dashboard", systemImage: "chevron.left")
                }
                .buttonStyle(.plain)
                .foregroundStyle(BeaconPalette.cyan)
                Spacer()
                Text("\(snapshot.workingSet?.parked.count ?? 0) parked")
                    .font(.caption.weight(.medium))
                    .foregroundStyle(BeaconPalette.lavender)
            }
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 8) {
                    let parked = state.lanes(for: snapshot.workingSet?.parked ?? [])
                    if parked.isEmpty {
                        ContentUnavailableView("No parked lanes", systemImage: "pause.circle")
                            .foregroundStyle(BeaconPalette.lavender)
                    } else {
                        ForEach(state.projectGroups(for: parked)) { project in
                            projectHeader(project, accent: BeaconPalette.lavender)
                            ForEach(project.lanes) { lane in
                                Button { state.open(lane) } label: {
                                    laneRow(lane, accent: BeaconPalette.lavender)
                                }
                                .buttonStyle(.plain)
                                .contextMenu { laneActions(lane) }
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
                        .contextMenu { laneActions(lane) }
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
            let stage = state.stage(for: project.id)
            if stage != "ready" && stage != "cached" {
                Text(stage.replacingOccurrences(of: "_", with: " ").capitalized)
                    .font(.caption2.weight(.medium))
                    .foregroundStyle(BeaconPalette.gold)
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
            if let attention = lane.attention {
                Text("\(attention.delta) · \(timeSinceActivity(lane.updatedAt))")
                    .font(.caption2)
                    .foregroundStyle(BeaconPalette.cyan)
                if let note = attention.note, !note.isEmpty {
                    Text("Note: \(note)\(attention.noteStale ? " · evidence changed" : "")")
                        .font(.caption2)
                        .foregroundStyle(attention.noteStale ? BeaconPalette.gold : BeaconPalette.lavender)
                        .lineLimit(2)
                }
            }
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

    @ViewBuilder
    private func laneActions(_ lane: WorkLane) -> some View {
        if lane.attention?.state == "parked" {
            Button("Resume") { Task { await state.setLaneAttention(lane, state: "active") } }
        } else {
            Button("Park") { Task { await state.setLaneAttention(lane, state: "parked") } }
        }
        Button(lane.attention?.pinned == true ? "Unpin" : "Pin") {
            Task { await state.setLanePinned(lane, pinned: lane.attention?.pinned != true) }
        }
        Button("Edit Note") {
            noteLane = lane
            noteText = lane.attention?.note ?? ""
            showingNoteEditor = true
        }
        Button("Mark Seen") { Task { await state.markLaneSeen(lane) } }
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

    private func timeSinceActivity(_ value: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let date = formatter.date(from: value) ?? ISO8601DateFormatter().date(from: value)
        guard let date else { return "activity unknown" }
        return RelativeDateTimeFormatter().localizedString(for: date, relativeTo: Date())
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
