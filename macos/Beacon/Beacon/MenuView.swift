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
    @State var showingQuietProjects = false
    @State var showingParkedLanes = false
    @State var showingProjectTracking = false
    @State var projectTrackingTab = ProjectTrackingTab.tracked
    @State var quietSearch = ""
    @State var tagLane: WorkLane?
    @State var tagText = ""
    @State var showingTagEditor = false
    @State var manualTitle = ""
    @State var showingManualEditor = false
    @AppStorage("beacon.dashboard.view-mode") private var viewModeValue = DashboardViewMode.stacked.rawValue
    @AppStorage("beacon.dismissed-evidence-badges") private var dismissedEvidenceBadgesValue = "[]"

    var viewMode: DashboardViewMode {
        get { DashboardViewMode(rawValue: viewModeValue) ?? .stacked }
        nonmutating set { viewModeValue = newValue.rawValue }
    }

    var dismissedEvidenceBadges: Set<String> {
        EvidenceBadgeDismissals.decode(dismissedEvidenceBadgesValue)
    }

    func dismissEvidenceBadge(_ key: String) {
        var hidden = dismissedEvidenceBadges
        hidden.insert(key)
        dismissedEvidenceBadgesValue = EvidenceBadgeDismissals.encode(hidden)
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
}
