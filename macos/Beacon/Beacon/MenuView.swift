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
    @State var selectedDashboardTab = DashboardTab.defaultTab
    @State var showingProjectInventory = false
    @State var projectInventoryTab = ProjectInventoryTab.following
    @State var tagLane: WorkLane?
    @State var tagText = ""
    @State var showingTagEditor = false
    @State var manualTitle = ""
    @State var showingManualEditor = false
    @State var notesDraft = ""
    @StateObject var notesAutosave = SignalNotesAutosave()
    @FocusState var notesEditorFocused: Bool
    @AppStorage("beacon.dashboard.view-mode") private var viewModeValue = DashboardViewMode.stacked.rawValue
    @AppStorage("beacon.dismissed-evidence-badges") private var dismissedEvidenceBadgesValue = "[]"
    @AppStorage("beacon.signal-notes-expanded") var signalNotesExpanded = SignalNotesPresentation.expandedByDefault

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
        .onAppear {
            loginItem.refresh()
            notesDraft = state.notesContent
        }
        .onChange(of: state.notesContent) { previous, latest in
            if !notesEditorFocused || notesDraft == previous {
                notesDraft = latest
            }
        }
        .onChange(of: notesDraft) { _, latest in
            scheduleSignalNotesAutosave(latest)
        }
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
            if let snapshot = state.snapshot {
                if showingProjectInventory {
                    ProjectFollowingView(
                        state: state,
                        selectedTab: $projectInventoryTab,
                        onClose: { showingProjectInventory = false }
                    )
                } else {
                    dashboardTabs()
                    dashboardContent(snapshot)
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
            signalNotesPanel
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
                    NeonWaveWordmark("Beacon")
                        .font(BeaconTypography.bold(17))
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
            refreshButton
            viewModeMenu
            settingsMenu
        }
    }

    private var refreshButton: some View {
        Button {
            Task { await state.scan() }
        } label: {
            Group {
                if state.isScanning {
                    ProgressView()
                        .controlSize(.small)
                        .tint(BeaconPalette.mint)
                } else {
                    Image(systemName: "arrow.clockwise")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(BeaconPalette.mint)
                }
            }
            .frame(width: 28, height: 28)
            .background(BeaconPalette.softGradient(BeaconPalette.mint), in: RoundedRectangle(cornerRadius: 8))
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(BeaconPalette.mint.opacity(0.42), lineWidth: 0.7)
            }
        }
        .buttonStyle(.plain)
        .disabled(state.isScanning)
        .help(state.isScanning ? "Scanning Git and GitHub evidence" : "Scan Now — refresh Git and GitHub evidence")
        .accessibilityLabel(state.isScanning ? "Scan in progress" : "Scan Now")
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
            Button { state.openTopItem() } label: { Label("Open Top Item", systemImage: "arrow.up.forward.app") }
                .disabled(state.inProgressCount == 0)
            Button { manualTitle = ""; showingManualEditor = true } label: { Label("Add Manual Lane", systemImage: "plus.circle") }
            Divider()
            Button { showProjects(.following) } label: { Label("Manage Following", systemImage: "star") }
            Button { showDashboardTab(.recent) } label: { Label("Recently Updated", systemImage: "sparkles") }
            Button { showDashboardTab(.quiet) } label: {
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

    private func showProjects(_ tab: ProjectInventoryTab) {
        projectInventoryTab = tab
        showingProjectInventory = true
    }

    private func showDashboardTab(_ tab: DashboardTab) {
        showingProjectInventory = false
        selectedDashboardTab = tab
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
