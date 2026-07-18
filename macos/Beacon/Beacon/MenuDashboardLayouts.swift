import SwiftUI

enum DashboardTab: String, CaseIterable, Identifiable {
    case following
    case parking
    case recent
    case quiet

    static let defaultTab = DashboardTab.following

    var id: String { rawValue }

    var title: String {
        switch self {
        case .following: "Following"
        case .parking: "Parking Lot"
        case .recent: "Recently Updated"
        case .quiet: "Quiet"
        }
    }

    var symbol: String {
        switch self {
        case .following: "star.fill"
        case .parking: "pause.circle.fill"
        case .recent: "sparkles"
        case .quiet: "moon.stars.fill"
        }
    }
}
enum DashboardDestination: Equatable {
    case tab(DashboardTab)
    case projectInventory
    case repositorySync
    case dependencyLimits

    static let following = DashboardDestination.tab(.following)

    func toggled(selecting destination: DashboardDestination) -> DashboardDestination {
        self == destination ? .following : destination
    }
}

extension MenuView {
    func dashboardTabs() -> some View {
        HStack(spacing: 5) {
            ForEach(DashboardTab.allCases, id: \.self) { tab in
                let accent = dashboardTabAccent(tab)
                let selected = selectedDashboardTab == tab
                Button {
                    showDashboardTab(tab)
                } label: {
                    VStack(spacing: 2) {
                        Label(tab.title, systemImage: tab.symbol)
                            .font(BeaconTypography.semibold(9))
                            .lineLimit(1)
                        Text("\(dashboardTabCount(tab))")
                            .font(BeaconTypography.medium(8))
                            .foregroundStyle(selected ? accent : BeaconPalette.lavender.opacity(0.72))
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 5)
                    .foregroundStyle(selected ? accent : BeaconPalette.lavender)
                    .background(
                        BeaconPalette.softGradient(selected ? accent : BeaconPalette.lavender)
                            .opacity(selected ? 1 : 0.35),
                        in: RoundedRectangle(cornerRadius: 8)
                    )
                    .overlay {
                        RoundedRectangle(cornerRadius: 8)
                            .strokeBorder(selected ? accent : BeaconPalette.lavender.opacity(0.18), lineWidth: selected ? 0.9 : 0.6)
                    }
                }
                .buttonStyle(.plain)
                .help(selected && tab != .following ? "Return to Following" : "Show \(tab.title)")
                .dropDestination(for: String.self) { laneIDs, _ in
                    guard (tab == .following || tab == .parking), let laneID = laneIDs.first else {
                        return false
                    }
                    Task { await state.moveLane(laneID, to: tab) }
                    return true
                }
                .accessibilityHint(tab == .following || tab == .parking
                    ? "Accepts dropped lanes"
                    : "Shows \(tab.title)")
            }
        }
    }

    @ViewBuilder
    func dashboardContent(_ snapshot: BeaconSnapshot) -> some View {
        switch selectedDashboardTab {
        case .following:
            activeDashboard(snapshot)
        case .parking:
            parkedDashboard(snapshot)
        case .recent:
            ProjectFollowingView(
                state: state,
                selectedTab: .constant(.recent),
                onClose: {},
                showsNavigation: false,
                showsTabPicker: false
            )
        case .quiet:
            ProjectFollowingView(
                state: state,
                selectedTab: .constant(.quiet),
                onClose: {},
                showsNavigation: false,
                showsTabPicker: false
            )
        }
    }

    func dashboardTabCount(_ tab: DashboardTab) -> Int {
        switch tab {
        case .following:
            state.followedProjectCount
        case .parking:
            snapshotParkedCount
        case .recent:
            state.recentProjectCount
        case .quiet:
            state.quietProjectCount
        }
    }

    func dashboardTabAccent(_ tab: DashboardTab) -> Color {
        switch tab {
        case .following: BeaconPalette.mint
        case .parking: BeaconPalette.lavender
        case .recent: BeaconPalette.pink
        case .quiet: BeaconPalette.cyan
        }
    }

    private var snapshotParkedCount: Int {
        guard let snapshot = state.snapshot, let workingSet = snapshot.workingSet else { return 0 }
        return workingSet.parked.count
    }

    @ViewBuilder
    func activeDashboard(_ snapshot: BeaconSnapshot) -> some View {
        if UpToDatePresentation.shouldShow(
            inProgressCount: state.inProgressCount,
            loadingProjectCount: state.loadingProjects.count
        ) {
            UpToDateBacksplash(surface: surface)
        } else {
            switch viewMode {
            case .stacked:
                stackedDashboard(snapshot)
            case .tiles:
                tileDashboard(snapshot)
            case .kanban:
                kanbanDashboard(snapshot)
            case .overview:
                overviewDashboard(snapshot)
            }
        }
    }

    @ViewBuilder
    func parkedDashboard(_ snapshot: BeaconSnapshot) -> some View {
        let lanes = state.lanes(for: snapshot.workingSet?.parked ?? [])
        if lanes.isEmpty {
            ContentUnavailableView("Parking Lot is empty", systemImage: "pause.circle")
                .foregroundStyle(BeaconPalette.lavender)
        } else {
            switch viewMode {
            case .stacked:
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 10) {
                        laneSection("Parking Lot", symbol: "pause.circle.fill", accent: BeaconPalette.lavender, lanes: lanes)
                    }
                }
            case .tiles:
                ScrollView {
                    tileSection("Parking Lot", symbol: "pause.circle.fill", accent: BeaconPalette.lavender, lanes: lanes)
                }
            case .kanban:
                GeometryReader { geometry in
                    kanbanColumn("Parking Lot", symbol: "pause.circle.fill", accent: BeaconPalette.lavender, lanes: lanes, height: geometry.size.height)
                }
            case .overview:
                overviewSection(
                    "Parking Lot",
                    symbol: "pause.circle.fill",
                    accent: BeaconPalette.lavender,
                    lanes: lanes
                )
            }
        }
    }

    func stackedDashboard(_ snapshot: BeaconSnapshot) -> some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 10) {
                if !state.loadingProjects.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Label("Loading Projects", systemImage: "antenna.radiowaves.left.and.right")
                            .font(BeaconTypography.semibold(11))
                            .foregroundStyle(BeaconPalette.borderGradient(BeaconPalette.cyan))
                        ForEach(state.loadingProjects, id: \.projectID) { project in
                            HStack(spacing: 8) {
                                ProgressView()
                                    .controlSize(.mini)
                                    .tint(BeaconPalette.cyan)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(project.name)
                                        .font(BeaconTypography.semibold(10))
                                    Text(stageLabel(project.stage))
                                        .font(BeaconTypography.regular(9))
                                        .foregroundStyle(BeaconPalette.lavender)
                                }
                                Spacer()
                                Text(project.projectID)
                                    .font(BeaconTypography.regular(9))
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
                } else {
                    laneSection("Ready for Review", symbol: "checkmark.circle.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: snapshot.groups.ready))
                    laneSection("Needs Action", symbol: "exclamationmark.triangle.fill", accent: BeaconPalette.coral, lanes: state.lanes(for: snapshot.groups.action))
                    laneSection("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: snapshot.groups.waiting))
                }
            }
        }
    }
}
