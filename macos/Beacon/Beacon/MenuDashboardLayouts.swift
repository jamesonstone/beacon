import SwiftUI

enum DashboardTab: String, CaseIterable, Identifiable {
    case active
    case parkingLot = "parking_lot"
    case quiet
    case untracked

    static let defaultTab = DashboardTab.active

    var id: String { rawValue }

    var title: String {
        switch self {
        case .active: "Active"
        case .parkingLot: "Parking Lot"
        case .quiet: "Quiet"
        case .untracked: "Untracked"
        }
    }

    var symbol: String {
        switch self {
        case .active: "bolt.fill"
        case .parkingLot: "pause.circle.fill"
        case .quiet: "moon.stars.fill"
        case .untracked: "eye.slash.fill"
        }
    }
}

extension MenuView {
    func dashboardTabs(_ snapshot: BeaconSnapshot) -> some View {
        HStack(spacing: 5) {
            ForEach(DashboardTab.allCases, id: \.self) { tab in
                let accent = dashboardTabAccent(tab)
                let selected = selectedDashboardTab == tab
                Button {
                    if tab == .quiet, selectedDashboardTab != .quiet {
                        quietSearch = ""
                    }
                    selectedDashboardTab = tab
                } label: {
                    VStack(spacing: 2) {
                        Label(tab.title, systemImage: tab.symbol)
                            .font(BeaconTypography.semibold(9))
                            .lineLimit(1)
                        Text("\(dashboardTabCount(tab, snapshot: snapshot))")
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
                .help("Show \(tab.title)")
            }
        }
    }

    @ViewBuilder
    func dashboardContent(_ snapshot: BeaconSnapshot) -> some View {
        switch selectedDashboardTab {
        case .active:
            activeDashboard(snapshot)
        case .parkingLot:
            parkedLanes(snapshot)
        case .quiet:
            quietProjects
        case .untracked:
            ProjectTrackingView(
                state: state,
                selectedTab: .constant(.untracked),
                onClose: {},
                showsNavigation: false,
                showsTabPicker: false
            )
        }
    }

    func dashboardTabCount(_ tab: DashboardTab, snapshot: BeaconSnapshot) -> Int {
        switch tab {
        case .active:
            state.inProgressCount
        case .parkingLot:
            snapshot.workingSet?.parked.count ?? 0
        case .quiet:
            state.quietProjectCount
        case .untracked:
            state.untrackedProjectCount
        }
    }

    func dashboardTabAccent(_ tab: DashboardTab) -> Color {
        switch tab {
        case .active: BeaconPalette.mint
        case .parkingLot: BeaconPalette.lavender
        case .quiet: BeaconPalette.cyan
        case .untracked: BeaconPalette.pink
        }
    }

    @ViewBuilder
    func activeDashboard(_ snapshot: BeaconSnapshot) -> some View {
        switch viewMode {
        case .stacked:
            stackedDashboard(snapshot)
        case .tiles:
            tileDashboard(snapshot)
        case .kanban:
            kanbanDashboard(snapshot)
        }
    }

    func stackedDashboard(_ snapshot: BeaconSnapshot) -> some View {
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
                } else {
                    laneSection("Ready for Review", symbol: "checkmark.circle.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: snapshot.groups.ready))
                    laneSection("Needs Action", symbol: "exclamationmark.triangle.fill", accent: BeaconPalette.coral, lanes: state.lanes(for: snapshot.groups.action))
                    laneSection("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: snapshot.groups.waiting))
                }
            }
        }
    }

    func tileDashboard(_ snapshot: BeaconSnapshot) -> some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 12) {
                loadingProjectStrip
                if let working = snapshot.workingSet {
                    tileSection("Active", symbol: "bolt.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: working.active))
                    tileSection("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: working.waiting))
                    tileSection("Recently Active", symbol: "sparkles", accent: BeaconPalette.cyan, lanes: state.lanes(for: working.recent))
                } else {
                    tileSection("Ready for Review", symbol: "checkmark.circle.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: snapshot.groups.ready))
                    tileSection("Needs Action", symbol: "exclamationmark.triangle.fill", accent: BeaconPalette.coral, lanes: state.lanes(for: snapshot.groups.action))
                    tileSection("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: snapshot.groups.waiting))
                }
            }
        }
    }

    func kanbanDashboard(_ snapshot: BeaconSnapshot) -> some View {
        GeometryReader { geometry in
            ScrollView(.horizontal) {
                HStack(alignment: .top, spacing: 10) {
                    if let working = snapshot.workingSet {
                        kanbanColumn("Active", symbol: "bolt.fill", accent: BeaconPalette.mint, lanes: state.lanes(for: working.active), height: geometry.size.height)
                        kanbanColumn("Waiting", symbol: "clock.fill", accent: BeaconPalette.gold, lanes: state.lanes(for: working.waiting), height: geometry.size.height)
                        kanbanColumn("Recent", symbol: "sparkles", accent: BeaconPalette.cyan, lanes: state.lanes(for: working.recent), height: geometry.size.height)
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
    var loadingProjectStrip: some View {
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
    func tileSection(_ title: String, symbol: String, accent: Color, lanes: [WorkLane]) -> some View {
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

    func kanbanColumn(_ title: String, symbol: String, accent: Color, lanes: [WorkLane], height: CGFloat) -> some View {
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

    func sectionHeader(_ title: String, symbol: String, accent: Color, count: Int) -> some View {
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

    func stageLabel(_ stage: String) -> String {
        switch stage {
        case "queued": return "Queued"
        case "local": return "Checking local Git"
        case "github": return "Checking GitHub"
        case "failed": return "Refresh failed — showing previous result"
        case "ready": return "Ready"
        default: return "Cached"
        }
    }
}
