import AppKit
import SwiftUI

enum DashboardSurface {
    case menu
    case window
}

struct MenuView: View {
    @Environment(\.accessibilityDifferentiateWithoutColor) var differentiateWithoutColor
    @Environment(\.accessibilityReduceMotion) var reduceMotion
    @Environment(\.accessibilityReduceTransparency) var reduceTransparency
    @Environment(\.colorSchemeContrast) var colorSchemeContrast
    @ObservedObject var state: AppState
    @ObservedObject var loginItem: LoginItemController
    @ObservedObject var terminal: DropDownTerminalController
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
    @State var evidenceHoverLaneID: String?
    @AppStorage("beacon.dashboard.view-mode") private var viewModeValue = DashboardViewMode.stacked.rawValue
    @AppStorage(DashboardDensity.storageKey) var densityValue = DashboardDensity.defaultDensity.rawValue
    @AppStorage("beacon.dismissed-evidence-badges") var dismissedEvidenceBadgesValue = "[]"
    @AppStorage(SignalNotesPresentation.sizeStorageKey) var signalNotesSizeValue = SignalNotesSize.half.rawValue
    @AppStorage(SignalNotesPresentation.lastExpandedStorageKey) var signalNotesLastExpandedSizeValue = SignalNotesSize.half.rawValue
    @AppStorage(BeaconTypography.familyKey) var fontFamilyValue = BeaconTypography.defaultFamily
    @AppStorage(BeaconTypography.baseSizeKey) var fontSizeValue = BeaconTypography.defaultBaseSize
    @AppStorage(BeaconThemePreference.storageKey) var themeIDValue = BeaconThemePreference.defaultID.rawValue

    var theme: BeaconTheme {
        BeaconThemeCatalog.theme(forStoredID: themeIDValue)
    }

    var interfaceBorderColor: Color {
        colorSchemeContrast == .increased
            ? theme.tokens.borderStrong.color
            : theme.tokens.border.color
    }

    var viewMode: DashboardViewMode {
        get { DashboardViewMode(rawValue: viewModeValue) ?? .stacked }
        nonmutating set { setViewMode(newValue) }
    }

    var density: DashboardDensity {
        DashboardDensity(rawValue: densityValue) ?? .comfortable
    }

    func setViewMode(_ mode: DashboardViewMode) {
        let previous = DashboardViewMode(rawValue: viewModeValue) ?? .stacked
        guard previous != mode else { return }
        let transition = DashboardOverviewPresentation.notesTransition(
            from: previous,
            to: mode,
            current: SignalNotesSize(rawValue: signalNotesSizeValue) ?? .half,
            lastExpanded: SignalNotesSize(rawValue: signalNotesLastExpandedSizeValue) ?? .half
        )
        signalNotesSizeValue = transition.current.rawValue
        signalNotesLastExpandedSizeValue = transition.lastExpanded.rawValue
        viewModeValue = mode.rawValue
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
            Task { await state.refreshOllamaModels() }
        }
        .background { keyboardShortcutControls }
        .overlay {
            GeometryReader { proxy in
                notesAssistantConversationOverlay(in: proxy.size)
            }
        }
        .overlay {
            if let switcherScope {
                ZStack {
                    (reduceTransparency ? theme.tokens.canvas.color : Color.black.opacity(0.42))
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
        .environment(\.beaconTheme, theme)
        .font(BeaconTypography.regular(10))
        .preferredColorScheme(theme.appearance.colorScheme)
        .tint(theme.tokens.accent.color)
        .foregroundStyle(theme.tokens.textPrimary.color)
        .symbolRenderingMode(differentiateWithoutColor ? .monochrome : .hierarchical)
        .overlay {
            if colorSchemeContrast == .increased {
                Rectangle()
                    .strokeBorder(theme.tokens.borderStrong.color, lineWidth: 1)
                    .allowsHitTesting(false)
                    .accessibilityHidden(true)
            }
        }
        .transaction { transaction in
            if reduceMotion { transaction.animation = nil }
        }
    }

    private var keyboardShortcutControls: some View {
        ZStack {
            Button("Quick Switcher") { showSwitcher(.all) }
                .keyboardShortcut("k", modifiers: .command)
            Button("Tab Search") { showSwitcher(.notes) }
                .keyboardShortcut("p", modifiers: .command)
            Button("Open AI Conversation") { showNotesAssistant(.conversation) }
                .keyboardShortcut("i", modifiers: .command)
            Button("Open Compact Notes AI") { showNotesAssistant(.compact) }
                .keyboardShortcut("i", modifiers: [.command, .shift])
            Button("Next Note") { Task { await state.cycleNotes(direction: 1) } }
                .keyboardShortcut(.tab, modifiers: .control)
            Button("Previous Note") { Task { await state.cycleNotes(direction: -1) } }
                .keyboardShortcut(.tab, modifiers: [.control, .shift])
            Button("Next Note Alternate") { Task { await state.cycleNotes(direction: 1) } }
                .keyboardShortcut("]", modifiers: [.command, .shift])
            Button("Previous Note Alternate") { Task { await state.cycleNotes(direction: -1) } }
                .keyboardShortcut("[", modifiers: [.command, .shift])
            ForEach(0..<9, id: \.self) { index in
                Button("Note \(index + 1)") { Task { await state.activateNote(at: index) } }
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
                        .tint(BeaconThemePreference.current().tokens.info.color)
                        .frame(maxWidth: .infinity, minHeight: 180)
                } else {
                    ContentUnavailableView("No scan available", systemImage: "dot.radiowaves.left.and.right")
                        .symbolRenderingMode(.palette)
                        .foregroundStyle(BeaconThemePreference.current().tokens.info.color, BeaconThemePreference.current().tokens.textSecondary.color)
                }
                signalNotesPanel
                    .frame(height: signalNotesHeight(in: geometry.size.height))
            }
            .padding(12)
        }
        .font(BeaconTypography.regular(12))
        .background(theme.tokens.canvas.color)
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
                .foregroundStyle(BeaconThemePreference.current().tokens.warning.color)
            Spacer()
            Button("Enable") { Task { await state.enableAgent() } }
                .buttonStyle(.bordered)
                .tint(BeaconThemePreference.current().tokens.info.color)
        }
        .padding(8)
        .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 8))
    }
}
