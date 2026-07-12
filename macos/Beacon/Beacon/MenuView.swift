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
    @State private var tagLane: WorkLane?
    @State private var tagText = ""
    @State private var showingTagEditor = false
    @State private var manualTitle = ""
    @State private var showingManualEditor = false
    @AppStorage("beacon.dashboard.view-mode") private var viewModeValue = DashboardViewMode.stacked.rawValue
    @AppStorage("beacon.dismissed-evidence-badges") private var dismissedEvidenceBadgesValue = "[]"

    private var viewMode: DashboardViewMode {
        get { DashboardViewMode(rawValue: viewModeValue) ?? .stacked }
        nonmutating set { viewModeValue = newValue.rawValue }
    }

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
        .alert("Add lane tag", isPresented: $showingTagEditor) {
            TextField("Short tag", text: $tagText)
            Button("Add") { if let lane = tagLane { Task { await state.addLaneTag(lane, tag: tagText) } } }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("Tags are local context and never change Beacon's status inference.")
        }
        .alert("Add manual lane", isPresented: $showingManualEditor) {
            TextField("Planning or research lane", text: $manualTitle)
            Button("Add") { Task { await state.addManualLane(manualTitle) } }
            Button("Cancel", role: .cancel) {}
        }
    }

    private var dashboard: some View {
        VStack(alignment: .leading, spacing: 8) {
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
                if showingProjectTracking {
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
            } else if state.isScanning {
                ProgressView("Scanning repositories…")
                    .tint(BeaconPalette.cyan)
                    .frame(maxWidth: .infinity, minHeight: 180)
            } else {
                ContentUnavailableView("No scan available", systemImage: "dot.radiowaves.left.and.right")
                    .symbolRenderingMode(.palette)
                    .foregroundStyle(BeaconPalette.cyan, BeaconPalette.lavender)
            }
        }
        .padding(12)
        .font(BeaconTypography.regular(12))
        .background(BeaconPalette.panelBackground)
    }

    private var header: some View {
        HStack(alignment: .top, spacing: 10) {
            HStack(spacing: 8) {
                Image(systemName: "sparkles")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(BeaconPalette.neonGradient)
                    .shadow(color: BeaconPalette.cyan.opacity(0.55), radius: 2)
                VStack(alignment: .leading, spacing: 3) {
                    Text("Beacon")
                        .font(BeaconTypography.bold(17))
                        .foregroundStyle(BeaconPalette.neonGradient)
                    Text("\(state.inProgressCount) lanes in focus")
                        .font(BeaconTypography.medium(11))
                        .foregroundStyle(BeaconPalette.mint)
                    if let generatedAt = state.snapshot?.generatedAt {
                        Text("Updated \(timeSinceActivity(generatedAt))")
                            .font(BeaconTypography.regular(9))
                            .foregroundStyle(BeaconPalette.lavender.opacity(0.82))
                    }
                }
            }
            Spacer()
            if state.isScanning {
                ProgressView()
                    .controlSize(.small)
                    .tint(BeaconPalette.cyan)
            }
            viewModeMenu
            settingsMenu
        }
    }

    private var viewModeMenu: some View {
        Menu {
            Picker("View mode", selection: Binding(get: { viewMode }, set: { viewMode = $0 })) {
                ForEach(DashboardViewMode.allCases) { mode in
                    Label(mode.title, systemImage: mode.symbol).tag(mode)
                }
            }
        } label: {
            Image(systemName: viewMode.symbol)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(BeaconPalette.cyan)
                .frame(width: 28, height: 28)
                .background(BeaconPalette.softGradient(BeaconPalette.cyan), in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(BeaconPalette.cyan.opacity(0.35), lineWidth: 0.7)
                }
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .fixedSize()
        .help("View mode: \(viewMode.title)")
    }

    private var settingsMenu: some View {
        Menu {
            if surface == .menu {
                Button(action: openDashboard) { Label("Open Dashboard", systemImage: "macwindow") }
            }
            Button { Task { await state.scan() } } label: { Label("Scan Now", systemImage: "arrow.clockwise") }
                .disabled(state.isScanning)
            Button { state.openTopItem() } label: { Label("Open Top Item", systemImage: "arrow.up.forward.app") }
                .disabled(state.inProgressCount == 0)
            Button { manualTitle = ""; showingManualEditor = true } label: { Label("Add Manual Lane", systemImage: "plus.circle") }
            Divider()
            Button { showProjects(.tracked) } label: { Label("Tracked Projects", systemImage: "checkmark.circle") }
            Button { showProjects(.untracked) } label: { Label("Untracked Projects", systemImage: "eye.slash") }
            Button { showingParkedLanes = true; showingProjectTracking = false; showingQuietProjects = false } label: {
                Label("Parked Lanes", systemImage: "pause.circle")
            }
            Button { quietSearch = ""; showingQuietProjects = true; showingProjectTracking = false; showingParkedLanes = false } label: {
                Label("Quiet Projects", systemImage: "moon.stars")
            }
            Button { state.openConfig() } label: { Label("Open Config", systemImage: "slider.horizontal.3") }
            Button {
                dismissedEvidenceBadgesValue = "[]"
            } label: {
                Label("Restore Hidden Badges", systemImage: "eye")
            }
            .disabled(dismissedEvidenceBadges.isEmpty)
            Divider()
            Toggle(
                "Open Beacon at Login",
                isOn: Binding(
                    get: { loginItem.isEnabled },
                    set: { loginItem.setEnabled($0) }
                )
            )
            if loginItem.requiresApproval {
                Button("Approve in Settings") {
                    loginItem.openSystemSettings()
                }
            }
            if !state.agentAvailable {
                Button { Task { await state.enableAgent() } } label: {
                    Label("Enable Background Agent", systemImage: "antenna.radiowaves.left.and.right")
                }
            }
            Divider()
            Button("Quit Beacon", systemImage: "power") { NSApplication.shared.terminate(nil) }
        } label: {
            Image(systemName: "gearshape.fill")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(BeaconPalette.neonGradient)
                .frame(width: 28, height: 28)
                .background(BeaconPalette.softGradient(BeaconPalette.lavender), in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.lavender), lineWidth: 0.7)
                }
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .fixedSize()
        .help("Settings")
    }

    private func showProjects(_ tab: ProjectTrackingTab) {
        projectTrackingTab = tab
        showingQuietProjects = false
        showingParkedLanes = false
        showingProjectTracking = true
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

    @ViewBuilder
    private func activeDashboard(_ snapshot: BeaconSnapshot) -> some View {
        switch viewMode {
        case .stacked:
            stackedDashboard(snapshot)
        case .tiles:
            tileDashboard(snapshot)
        case .kanban:
            kanbanDashboard(snapshot)
        }
    }

    private func stackedDashboard(_ snapshot: BeaconSnapshot) -> some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 10) {
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

    private func tileDashboard(_ snapshot: BeaconSnapshot) -> some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 12) {
                loadingProjectStrip
                if let working = snapshot.workingSet {
                    tileSection("Active", symbol: "bolt.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: working.active))
                    tileSection("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: working.waiting))
                    tileSection("Recently Active", symbol: "sparkles", accent: BeaconPalette.cyan, lanes: state.lanes(for: working.recent))
                    tileSection("Parked", symbol: "pause.circle.fill", accent: BeaconPalette.lavender, lanes: state.lanes(for: working.parked))
                } else {
                    tileSection("Ready for Review", symbol: "checkmark.circle.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: snapshot.groups.ready))
                    tileSection("Needs Action", symbol: "exclamationmark.triangle.fill", accent: BeaconPalette.coral, lanes: state.lanes(for: snapshot.groups.action))
                    tileSection("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: snapshot.groups.waiting))
                }
            }
        }
    }

    private func kanbanDashboard(_ snapshot: BeaconSnapshot) -> some View {
        GeometryReader { geometry in
            ScrollView(.horizontal) {
                HStack(alignment: .top, spacing: 10) {
                    if let working = snapshot.workingSet {
                        kanbanColumn("Active", symbol: "bolt.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: working.active), height: geometry.size.height)
                        kanbanColumn("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: working.waiting), height: geometry.size.height)
                        kanbanColumn("Recent", symbol: "sparkles", accent: BeaconPalette.cyan, lanes: state.lanes(for: working.recent), height: geometry.size.height)
                        kanbanColumn("Parked", symbol: "pause.circle.fill", accent: BeaconPalette.lavender, lanes: state.lanes(for: working.parked), height: geometry.size.height)
                    } else {
                        kanbanColumn("Ready", symbol: "checkmark.circle.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: snapshot.groups.ready), height: geometry.size.height)
                        kanbanColumn("Action", symbol: "exclamationmark.triangle.fill", accent: BeaconPalette.coral, lanes: state.lanes(for: snapshot.groups.action), height: geometry.size.height)
                        kanbanColumn("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: snapshot.groups.waiting), height: geometry.size.height)
                    }
                }
                .padding(.bottom, 4)
            }
        }
    }

    @ViewBuilder
    private var loadingProjectStrip: some View {
        if !state.loadingProjects.isEmpty {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(state.loadingProjects, id: \.projectID) { project in
                        HStack(spacing: 6) {
                            ProgressView().controlSize(.mini).tint(BeaconPalette.cyan)
                            Text(project.name).font(BeaconTypography.medium(10))
                            Text(stageLabel(project.stage))
                                .font(BeaconTypography.regular(9))
                                .foregroundStyle(BeaconPalette.lavender)
                        }
                        .padding(.horizontal, 9)
                        .padding(.vertical, 6)
                        .background(BeaconPalette.softGradient(BeaconPalette.cyan), in: Capsule())
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func tileSection(_ title: String, symbol: String, accent: Color, lanes: [WorkLane]) -> some View {
        if !lanes.isEmpty {
            VStack(alignment: .leading, spacing: 6) {
                sectionHeader(title, symbol: symbol, accent: accent, count: lanes.count)
                ScrollView(.horizontal, showsIndicators: false) {
                    LazyHStack(alignment: .top, spacing: 9) {
                        ForEach(lanes) { lane in
                            laneCard(lane, accent: accent, compact: true)
                                .frame(width: 248)
                        }
                    }
                    .padding(.vertical, 2)
                }
            }
        }
    }

    private func kanbanColumn(_ title: String, symbol: String, accent: Color, lanes: [WorkLane], height: CGFloat) -> some View {
        VStack(alignment: .leading, spacing: 7) {
            sectionHeader(title, symbol: symbol, accent: accent, count: lanes.count)
            ScrollView {
                LazyVStack(spacing: 8) {
                    ForEach(lanes) { lane in
                        laneCard(lane, accent: accent, compact: true)
                    }
                    if lanes.isEmpty {
                        Text("No lanes")
                            .font(BeaconTypography.regular(10))
                            .foregroundStyle(BeaconPalette.lavender.opacity(0.65))
                            .frame(maxWidth: .infinity, minHeight: 70)
                    }
                }
            }
        }
        .padding(9)
        .frame(width: 238, height: height, alignment: .top)
        .background(BeaconPalette.softGradient(accent), in: RoundedRectangle(cornerRadius: 10))
        .overlay {
            RoundedRectangle(cornerRadius: 10)
                .strokeBorder(accent.opacity(0.28), lineWidth: 0.7)
        }
    }

    private func sectionHeader(_ title: String, symbol: String, accent: Color, count: Int) -> some View {
        HStack(spacing: 6) {
            Image(systemName: symbol)
                .foregroundStyle(accent)
                .shadow(color: accent.opacity(0.45), radius: 2)
            Text(title).foregroundStyle(BeaconPalette.borderGradient(accent))
            Spacer()
            Text("\(count)")
                .font(BeaconTypography.medium(9))
                .foregroundStyle(accent)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(accent.opacity(0.12), in: Capsule())
        }
        .font(BeaconTypography.semibold(12))
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
                                laneCard(lane, accent: BeaconPalette.lavender)
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
                                laneCard(lane, accent: BeaconPalette.lavender)
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
                sectionHeader(title, symbol: symbol, accent: accent, count: lanes.count)
                ForEach(state.projectGroups(for: lanes)) { project in
                    projectHeader(project, accent: accent)
                    ForEach(project.lanes) { lane in
                        laneCard(lane, accent: accent)
                    }
                }
            }
        }
    }

    private func projectHeader(_ project: ProjectLaneGroup, accent: Color) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(project.name)
                .font(BeaconTypography.semibold(10))
                .foregroundStyle(BeaconPalette.borderGradient(accent))
            if let progress = project.progress {
                Text("\(progress.feature) · \(actionLabel(progress.phase))")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.85))
                    .lineLimit(1)
            }
            let stage = state.stage(for: project.id)
            if stage != "ready" && stage != "cached" {
                Text(stage.replacingOccurrences(of: "_", with: " ").capitalized)
                    .font(BeaconTypography.medium(9))
                    .foregroundStyle(BeaconPalette.gold)
            }
            Spacer()
        }
        .padding(.top, 2)
    }

    private func laneCard(_ lane: WorkLane, accent: Color, compact: Bool = false) -> some View {
        laneRow(lane, accent: accent, compact: compact)
            .contentShape(RoundedRectangle(cornerRadius: 9))
            .onTapGesture { state.open(lane) }
            .contextMenu { laneActions(lane) }
    }

    private func laneRow(_ lane: WorkLane, accent: Color, compact: Bool = false) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            HStack {
                if compact {
                    projectGlyph(lane, accent: accent)
                }
                Text(workItemTitle(lane))
                    .font(BeaconTypography.semibold(compact ? 11 : 13))
                    .lineLimit(compact ? 2 : 1)
                Spacer()
                if let pullRequest = lane.pullRequest {
                    Text("PR #\(pullRequest.number)").font(BeaconTypography.medium(10)).foregroundStyle(accent)
                } else if let issue = lane.issue {
                    Text("Issue #\(issue.number)").font(BeaconTypography.medium(10)).foregroundStyle(accent)
                } else if !lane.branch.isEmpty {
                    Text(lane.branch).font(BeaconTypography.medium(9)).foregroundStyle(accent).lineLimit(1)
                }
            }
            Text(actionLabel(lane.nextAction))
                .font(BeaconTypography.medium(10))
                .foregroundStyle(accent)
            if let attention = lane.attention {
                Text("\(attention.delta) · \(timeSinceActivity(lane.updatedAt))")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconPalette.cyan)
                    .lineLimit(1)
                tagChips(lane, tags: attention.tags ?? [], accent: accent)
                if let note = attention.note, !note.isEmpty {
                    Label("\(note)\(attention.noteStale ? " · evidence changed" : "")", systemImage: "note.text")
                        .font(BeaconTypography.regular(9))
                        .foregroundStyle(attention.noteStale ? BeaconPalette.gold : BeaconPalette.lavender)
                        .lineLimit(2)
                }
            }
            if !compact, let progress = lane.progress {
                Text("Kit \(actionLabel(progress.phase)) · \(progress.summary)")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.85))
                    .lineLimit(1)
            }
            evidenceBadges(lane)
        }
        .padding(compact ? 9 : 10)
        .background(BeaconPalette.softGradient(accent), in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9)
                .strokeBorder(BeaconPalette.borderGradient(accent), lineWidth: 0.8)
        }
        .shadow(color: accent.opacity(0.09), radius: 4, y: 2)
    }

    private func projectGlyph(_ lane: WorkLane, accent: Color) -> some View {
        Text(String((lane.repository.isEmpty ? "?" : lane.repository).prefix(1)).uppercased())
            .font(BeaconTypography.bold(10))
            .foregroundStyle(accent)
            .frame(width: 24, height: 24)
            .background(BeaconPalette.softGradient(accent), in: RoundedRectangle(cornerRadius: 7))
            .overlay {
                RoundedRectangle(cornerRadius: 7).strokeBorder(accent.opacity(0.45), lineWidth: 0.7)
            }
    }

    @ViewBuilder
    private func tagChips(_ lane: WorkLane, tags: [String], accent: Color) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 5) {
                ForEach(tags, id: \.self) { tag in
                    HStack(spacing: 3) {
                        Text("#\(tag)")
                            .font(BeaconTypography.medium(9))
                        Button {
                            Task { await state.removeLaneTag(lane, tag: tag) }
                        } label: {
                            Image(systemName: "xmark")
                                .font(.system(size: 7, weight: .bold))
                        }
                        .buttonStyle(.plain)
                        .help("Remove \(tag)")
                    }
                    .foregroundStyle(BeaconPalette.lavender)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 3)
                    .background(BeaconPalette.softGradient(BeaconPalette.lavender), in: Capsule())
                    .overlay { Capsule().strokeBorder(BeaconPalette.lavender.opacity(0.38), lineWidth: 0.6) }
                }
                Button {
                    beginAddingTag(to: lane)
                } label: {
                    Label("Tag", systemImage: "plus")
                        .font(BeaconTypography.medium(9))
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3)
                }
                .buttonStyle(.plain)
                .foregroundStyle(accent)
                .background(BeaconPalette.softGradient(accent), in: Capsule())
                .overlay { Capsule().strokeBorder(accent.opacity(0.35), lineWidth: 0.6) }
            }
        }
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
        Button("Add Tag") { beginAddingTag(to: lane) }
        if let tags = lane.attention?.tags, !tags.isEmpty {
            Menu("Remove Tag") {
                ForEach(tags, id: \.self) { tag in
                    Button(tag) { Task { await state.removeLaneTag(lane, tag: tag) } }
                }
            }
        }
        if lane.attention?.note?.isEmpty == false {
            Button("Clear Legacy Note") { Task { await state.setLaneNote(lane, note: "") } }
        }
        Button("Mark Seen") { Task { await state.markLaneSeen(lane) } }
    }

    private func beginAddingTag(to lane: WorkLane) {
        tagLane = lane
        tagText = ""
        showingTagEditor = true
    }

    private func evidenceBadges(_ lane: WorkLane) -> some View {
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

    private var dismissedEvidenceBadges: Set<String> {
        EvidenceBadgeDismissals.decode(dismissedEvidenceBadgesValue)
    }

    @ViewBuilder
    private func dismissibleBadge(
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
                var hidden = dismissedEvidenceBadges
                hidden.insert(key)
                dismissedEvidenceBadgesValue = EvidenceBadgeDismissals.encode(hidden)
            }
        }
    }

    private func timeSinceActivity(_ value: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let date = formatter.date(from: value) ?? ISO8601DateFormatter().date(from: value)
        guard let date else { return "activity unknown" }
        return RelativeDateTimeFormatter().localizedString(for: date, relativeTo: Date())
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
