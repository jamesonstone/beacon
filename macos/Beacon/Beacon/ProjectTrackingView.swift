import SwiftUI

enum ProjectTrackingTab: String, CaseIterable, Identifiable {
    case tracked = "Tracked"
    case untracked = "Untracked"

    var id: String { rawValue }
}

struct ProjectTrackingView: View {
    @ObservedObject var state: AppState
    @Binding var selectedTab: ProjectTrackingTab
    let onClose: () -> Void
    var showsTabPicker = true
    @State private var search = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Button(action: onClose) {
                    Label("Dashboard", systemImage: "chevron.left")
                }
                .buttonStyle(.plain)
                .foregroundStyle(BeaconPalette.cyan)
                Spacer()
                Text("\(projects.count) \(selectedTab.rawValue.lowercased())")
                    .font(.caption.weight(.medium))
                    .foregroundStyle(BeaconPalette.lavender)
                if state.queuedTrackingCount > 0 {
                    Text("\(state.queuedTrackingCount) queued")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(BeaconPalette.gold)
                }
            }
            if showsTabPicker {
                Picker("Project tracking", selection: $selectedTab) {
                    ForEach(ProjectTrackingTab.allCases) { tab in
                        Text(tab.rawValue).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                .onChange(of: selectedTab) { _, _ in search = "" }
            }

            TextField("Search \(selectedTab.rawValue.lowercased()) projects", text: $search)
                .textFieldStyle(.roundedBorder)

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 8) {
                    if filteredProjects.isEmpty {
                        ContentUnavailableView.search(text: search)
                            .foregroundStyle(BeaconPalette.lavender)
                    } else {
                        ForEach(filteredProjects) { project in
                            projectRow(project)
                        }
                    }
                }
            }
        }
    }

    private var projects: [BeaconProject] {
        selectedTab == .tracked ? state.trackedProjects : state.untrackedProjects
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
        let tracking = selectedTab == .untracked
        let accent = tracking ? BeaconPalette.mint : BeaconPalette.lavender
        return HStack(spacing: 10) {
            VStack(alignment: .leading, spacing: 3) {
                Text(project.name)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(BeaconPalette.borderGradient(accent))
                Text(project.github)
                    .font(.caption)
                    .foregroundStyle(BeaconPalette.cyan.opacity(0.9))
                Text(project.path)
                    .font(.caption2)
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.78))
                    .lineLimit(1)
                let status = state.projectStatuses[project.github]
                HStack(spacing: 6) {
                    Text(stageLabel(state.stage(for: project.github)))
                    if let mutedAt = status?.mutedAt, selectedTab == .untracked {
                        Text("Muted \(mutedAt)")
                    }
                    if let probe = status?.lastProbeAt, selectedTab == .untracked {
                        Text("Probed \(probe)")
                    }
                }
                .font(.caption2)
                .foregroundStyle(accent.opacity(0.9))
            }
            Spacer()
            if state.isMutating(project) {
                ProgressView()
                    .controlSize(.small)
                    .tint(accent)
            } else {
                Button {
                    state.setProjectTracked(project, tracked: tracking)
                } label: {
                    Label(tracking ? "Track" : "Untrack", systemImage: tracking ? "eye.fill" : "eye.slash.fill")
                }
                .buttonStyle(.bordered)
                .tint(accent)
            }
        }
        .padding(9)
        .background(BeaconPalette.softGradient(accent), in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9)
                .strokeBorder(BeaconPalette.borderGradient(accent), lineWidth: 0.8)
        }
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
