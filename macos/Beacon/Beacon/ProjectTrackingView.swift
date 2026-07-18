import SwiftUI

enum ProjectInventoryTab: String, CaseIterable, Identifiable {
    case following
    case recent
    case quiet

    var id: String { rawValue }

    var title: String {
        switch self {
        case .following: "Following"
        case .recent: "Recently Updated"
        case .quiet: "Quiet"
        }
    }
}

struct ProjectFollowingView: View {
    @ObservedObject var state: AppState
    @Binding var selectedTab: ProjectInventoryTab
    let onClose: () -> Void
    var showsNavigation = true
    var showsTabPicker = true
    @State private var search = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                if showsNavigation {
                    Button(action: onClose) {
                        Label("Dashboard", systemImage: "chevron.left")
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                }
                Spacer()
                Text("\(projects.count) \(selectedTab.title.lowercased())")
                    .font(BeaconTypography.medium(9))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
                if state.queuedTrackingCount > 0 {
                    Text("\(state.queuedTrackingCount) queued")
                        .font(BeaconTypography.semibold(10))
                        .foregroundStyle(BeaconThemePreference.current().tokens.warning.color)
                }
            }
            if showsTabPicker {
                Picker("Project following", selection: $selectedTab) {
                    ForEach(ProjectInventoryTab.allCases) { tab in
                        Text(tab.title).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                .onChange(of: selectedTab) { _, _ in search = "" }
            }

            TextField("Search \(selectedTab.title.lowercased()) projects", text: $search)
                .textFieldStyle(.roundedBorder)

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 8) {
                    if filteredProjects.isEmpty {
                        ContentUnavailableView.search(text: search)
                            .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
                    } else {
                        ForEach(filteredProjects) { project in
                            projectRow(project)
                        }
                    }
                }
            }
        }
        .font(BeaconTypography.regular(11))
    }

    private var projects: [BeaconProject] {
        state.projects(in: selectedTab.rawValue)
    }

    private var filteredProjects: [BeaconProject] {
        let query = search.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !query.isEmpty else { return projects }
        return projects.filter {
            $0.name.lowercased().contains(query)
                || $0.github.lowercased().contains(query)
                || $0.path.lowercased().contains(query)
        }
    }

    private func projectRow(_ project: BeaconProject) -> some View {
        let willFollow = selectedTab != .following
        let accent = inventoryAccent
        return HStack(spacing: 10) {
            VStack(alignment: .leading, spacing: 3) {
                Text(project.name)
                    .font(BeaconTypography.semibold(11))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textPrimary.color)
                Text(project.github)
                    .font(BeaconTypography.regular(10))
                    .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                Text(project.path)
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                    .lineLimit(1)
                let status = state.projectStatuses[project.github]
                HStack(spacing: 6) {
                    Text(stageLabel(state.stage(for: project.github)))
                    if let activityReason = project.activityReason, selectedTab == .recent {
                        Text(activityReason)
                    }
                    if let activityAt = project.lastActivityAt, selectedTab == .recent {
                        Text(relativeTime(activityAt))
                    } else if let probe = status?.lastProbeAt, selectedTab != .following {
                        Text("Checked \(relativeTime(probe))")
                    }
                }
                .font(BeaconTypography.regular(9))
                .foregroundStyle(accent)
            }
            Spacer()
            if state.isMutating(project) {
                ProgressView()
                    .controlSize(.small)
                    .tint(accent)
            } else {
                Button {
                    state.setProjectFollowed(project, followed: willFollow)
                } label: {
                    Label(
                        willFollow ? "Follow" : "Stop Following",
                        systemImage: willFollow ? "star.fill" : "star.slash.fill"
                    )
                }
                .buttonStyle(.bordered)
                .tint(accent)
            }
        }
        .padding(9)
        .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9)
                .strokeBorder(BeaconThemePreference.current().tokens.borderStrong.color, lineWidth: 0.8)
        }
    }

    private var inventoryAccent: Color {
        switch selectedTab {
        case .following: BeaconThemePreference.current().tokens.textSecondary.color
        case .recent: BeaconThemePreference.current().tokens.accent.color
        case .quiet: BeaconThemePreference.current().tokens.info.color
        }
    }

    private func relativeTime(_ value: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let date = formatter.date(from: value) ?? ISO8601DateFormatter().date(from: value)
        guard let date else { return value }
        return RelativeDateTimeFormatter().localizedString(for: date, relativeTo: Date())
    }

    private func stageLabel(_ stage: String) -> String {
        switch stage {
        case "queued": "Queued"
        case "local": "Checking local Git"
        case "github": "Checking GitHub"
        case "failed": "Refresh failed — showing previous result"
        case "ready": "Ready"
        default: "Cached"
        }
    }
}
