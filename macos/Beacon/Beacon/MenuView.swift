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
    @State var dashboardDestination = DashboardDestination.following
    @State var projectInventoryTab = ProjectInventoryTab.following
    @State var tagLane: WorkLane?
    @State var tagText = ""
    @State var showingTagEditor = false
    @State var manualTitle = ""
    @State var showingManualEditor = false
    @State var notesEditorFocused = false
    @State var switcherScope: BeaconSwitcherScope?
    @State var switcherQuery = ""
    @State var switcherSelection = 0
    @State var notePendingDeletion: AgentNoteTab?
    @AppStorage("beacon.dashboard.view-mode") private var viewModeValue = DashboardViewMode.stacked.rawValue
    @AppStorage("beacon.dismissed-evidence-badges") var dismissedEvidenceBadgesValue = "[]"
    @AppStorage("beacon.signal-notes-expanded") var signalNotesExpanded = SignalNotesPresentation.expandedByDefault
    @AppStorage(BeaconTypography.familyKey) var fontFamilyValue = BeaconTypography.defaultFamily.rawValue
    @AppStorage(BeaconTypography.baseSizeKey) var fontSizeValue = BeaconTypography.defaultBaseSize

    var viewMode: DashboardViewMode {
        get { DashboardViewMode(rawValue: viewModeValue) ?? .stacked }
        nonmutating set { viewModeValue = newValue.rawValue }
    }

    var dismissedEvidenceBadges: Set<String> {
        EvidenceBadgeDismissals.decode(dismissedEvidenceBadgesValue)
    }

    var selectedDashboardTab: DashboardTab {
        guard case let .tab(tab) = dashboardDestination else {
            return .defaultTab
        }
        return tab
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
            Task { await state.refreshIntegrationHealth() }
        }
        .background { keyboardShortcutControls }
        .overlay {
            if let switcherScope {
                ZStack {
                    Color.black.opacity(0.52)
                        .contentShape(Rectangle())
                        .onTapGesture { self.switcherScope = nil }
                    BeaconQuickSwitcher(
                        scope: switcherScope,
                        commands: switcherScope == .all ? allSwitcherCommands : noteSwitcherCommands,
                        query: $switcherQuery,
                        selection: $switcherSelection,
                        onDeleteNote: requestNoteDeletion,
                        dismiss: { self.switcherScope = nil }
                    )
                    .padding(18)
                }
            }
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
        .alert(item: $notePendingDeletion) { tab in
            Alert(
                title: Text("Delete \u{201c}\(tab.title)\u{201d}?"),
                message: Text("This permanently deletes the detail note and cannot be undone."),
                primaryButton: .destructive(Text("Delete Note")) {
                    Task { await state.deleteNote(tab.id) }
                },
                secondaryButton: .cancel()
            )
        }
    }

    private var keyboardShortcutControls: some View {
        ZStack {
            Button("Quick Switcher") { showSwitcher(.all) }
                .keyboardShortcut("k", modifiers: .command)
            Button("Tab Search") { showSwitcher(.notes) }
                .keyboardShortcut("p", modifiers: .command)
            Button("Next Signal Note") { Task { await state.cycleNotes(direction: 1) } }
                .keyboardShortcut(.tab, modifiers: .control)
            Button("Previous Signal Note") { Task { await state.cycleNotes(direction: -1) } }
                .keyboardShortcut(.tab, modifiers: [.control, .shift])
            Button("Next Signal Note Alternate") { Task { await state.cycleNotes(direction: 1) } }
                .keyboardShortcut("]", modifiers: [.command, .shift])
            Button("Previous Signal Note Alternate") { Task { await state.cycleNotes(direction: -1) } }
                .keyboardShortcut("[", modifiers: [.command, .shift])
            ForEach(0..<9, id: \.self) { index in
                Button("Signal Note \(index + 1)") { Task { await state.activateNote(at: index) } }
                    .keyboardShortcut(KeyEquivalent(Character(String(index + 1))), modifiers: .command)
            }
        }
        .frame(width: 0, height: 0)
        .opacity(0)
        .accessibilityHidden(true)
    }

    func requestNoteDeletion(_ tab: AgentNoteTab) {
        guard tab.id != "general", tab.id != "new" else { return }
        switcherScope = nil
        notePendingDeletion = tab
    }

    private var dashboard: some View {
        GeometryReader { geometry in
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
                if dashboardDestination == .repositorySync {
                    RepositorySyncView(
                        state: state,
                        onClose: showFollowing
                    )
                } else if dashboardDestination == .dependencyLimits {
                    DependencyLimitsView(
                        state: state,
                        onClose: showFollowing
                    )
                } else if let snapshot = state.snapshot {
                    if dashboardDestination == .projectInventory {
                        ProjectFollowingView(
                            state: state,
                            selectedTab: $projectInventoryTab,
                            onClose: showFollowing
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
                    .frame(height: signalNotesExpanded ? max(220, geometry.size.height * SignalNotesPresentation.expandedHeightFraction) : nil)
            }
            .padding(12)
        }
        .font(BeaconTypography.regular(12))
        .background(BeaconPalette.panelBackground)
    }

    var viewModeMenu: some View {
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

    func showProjects(_ tab: ProjectInventoryTab) {
        projectInventoryTab = tab
        toggleDashboardDestination(.projectInventory)
    }

    func showDashboardTab(_ tab: DashboardTab) {
        toggleDashboardDestination(.tab(tab))
    }

    @discardableResult
    func toggleDashboardDestination(_ destination: DashboardDestination) -> Bool {
        let isOpening = dashboardDestination != destination
        dashboardDestination = dashboardDestination.toggled(selecting: destination)
        return isOpening
    }

    private func showFollowing() {
        dashboardDestination = .following
    }

    private var enableAgentBanner: some View {
        HStack {
            Label("Background agent unavailable", systemImage: "antenna.radiowaves.left.and.right.slash")
                .font(BeaconTypography.regular(10))
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
