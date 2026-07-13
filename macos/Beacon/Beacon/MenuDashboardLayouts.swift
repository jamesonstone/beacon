import SwiftUI

extension MenuView {
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

    func tileDashboard(_ snapshot: BeaconSnapshot) -> some View {
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

    func kanbanDashboard(_ snapshot: BeaconSnapshot) -> some View {
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
