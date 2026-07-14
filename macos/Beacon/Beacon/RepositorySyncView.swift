import SwiftUI

struct RepositorySyncView: View {
    @ObservedObject var state: AppState
    let onClose: () -> Void
    @State private var selected: Set<String> = []

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Button(action: onClose) {
                    Image(systemName: "chevron.left")
                        .frame(width: 24, height: 24)
                }
                .buttonStyle(.plain)
                .font(BeaconTypography.medium(9))
                .help("Back to Dashboard")
                .accessibilityLabel("Dashboard")
                VStack(alignment: .leading, spacing: 1) {
                    Text("Repository Sync")
                        .font(BeaconTypography.semibold(12))
                        .foregroundStyle(BeaconPalette.borderGradient(BeaconPalette.gold))
                    Text("Git-only · fast-forward-safe")
                        .font(BeaconTypography.regular(8))
                        .foregroundStyle(BeaconPalette.lavender.opacity(0.78))
                        .lineLimit(1)
                        .minimumScaleFactor(0.8)
                }
                .layoutPriority(1)
                Spacer()
                checkButton
            }

            if let error = state.repositorySyncError {
                Label(error, systemImage: "exclamationmark.triangle.fill")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconPalette.coral)
                    .lineLimit(2)
            }

            if let report = state.repositorySyncReport {
                HStack {
                    Label(
                        report.fetchAttempted ? "Remote refs checked" : "Local refs only",
                        systemImage: report.fetchAttempted ? "network" : "internaldrive"
                    )
                    .font(BeaconTypography.regular(8))
                    .foregroundStyle(BeaconPalette.lavender)
                    Spacer()
                    Text("\(state.repositoriesNeedingSync.count) need attention")
                        .font(BeaconTypography.medium(9))
                        .foregroundStyle(state.repositoriesNeedingSync.isEmpty ? BeaconPalette.mint : BeaconPalette.gold)
                }

                ScrollView {
                    LazyVStack(spacing: 7) {
                        ForEach(report.repositories) { repository in
                            repositoryRow(repository)
                        }
                    }
                    .padding(.vertical, 2)
                }

                HStack {
                    Button("Update All Safe", systemImage: "arrow.up.circle.fill") {
                        Task { await state.syncRepositories(state.safeRepositoryUpdates.map(\.projectID)) }
                    }
                    .buttonStyle(.bordered)
                    .disabled(state.safeRepositoryUpdates.isEmpty || state.isApplyingRepositorySync)
                    Spacer()
                    Button("Update Selected", systemImage: "checkmark.circle.fill") {
                        Task { await state.syncRepositories(selected.sorted()) }
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(BeaconPalette.gold.opacity(0.78))
                    .disabled(selected.isEmpty || state.isApplyingRepositorySync)
                }
                .font(BeaconTypography.medium(9))
            } else if state.isCheckingRepositorySync {
                ProgressView("Inspecting local Git refs…")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                ContentUnavailableView(
                    "No repository check yet",
                    systemImage: "arrow.triangle.2.circlepath",
                    description: Text("Open this panel to inspect local refs, then click Check for Updates to fetch explicitly.")
                )
            }
        }
        .onAppear { synchronizeSelection() }
        .onChange(of: state.repositorySyncReport) { _, _ in synchronizeSelection() }
    }

    private var checkButton: some View {
        Button {
            Task { await state.checkRepositorySync(refresh: true) }
        } label: {
            if state.isCheckingRepositorySync {
                ProgressView().controlSize(.small)
            } else {
                Label("Check for Updates", systemImage: "arrow.clockwise")
            }
        }
        .buttonStyle(.bordered)
        .font(BeaconTypography.medium(9))
        .disabled(state.isCheckingRepositorySync || state.isApplyingRepositorySync)
        .help("Run git fetch --prune --no-tags for configured default branches")
    }

    private func repositoryRow(_ repository: RepositorySyncItem) -> some View {
        HStack(alignment: .top, spacing: 8) {
            Toggle(
                isOn: Binding(
                    get: { selected.contains(repository.projectID) },
                    set: { enabled in
                        if enabled { selected.insert(repository.projectID) } else { selected.remove(repository.projectID) }
                    }
                )
            ) {
                EmptyView()
            }
            .toggleStyle(.checkbox)
            .labelsHidden()
            .disabled(!repository.canUpdate || state.isApplyingRepositorySync)

            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 6) {
                    Text(repository.name)
                        .font(BeaconTypography.semibold(10))
                    Text("\(repository.currentBranch ?? "detached") → \(repository.base)")
                        .font(BeaconTypography.medium(8))
                        .foregroundStyle(BeaconPalette.cyan)
                    Spacer()
                    Label(stateLabel(repository), systemImage: stateSymbol(repository))
                        .font(BeaconTypography.medium(8))
                        .foregroundStyle(stateAccent(repository))
                }
                Text(repository.reason)
                    .font(BeaconTypography.regular(8))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.82))
                    .lineLimit(2)
            }

            if repository.canUpdate {
                Button("Update") {
                    Task { await state.syncRepositories([repository.projectID]) }
                }
                .buttonStyle(.bordered)
                .font(BeaconTypography.medium(8))
                .disabled(state.isApplyingRepositorySync)
            }
        }
        .padding(8)
        .background(BeaconPalette.softGradient(stateAccent(repository)), in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9)
                .strokeBorder(stateAccent(repository).opacity(0.34), lineWidth: 0.7)
        }
    }

    private func synchronizeSelection() {
        let available = Set(state.safeRepositoryUpdates.map(\.projectID))
        selected.formIntersection(available)
        if selected.isEmpty {
            selected = available
        }
    }

    private func stateLabel(_ repository: RepositorySyncItem) -> String {
        if repository.updated { return "Updated" }
        if repository.canUpdate { return "Ready" }
        switch repository.state {
        case "current": return "Current"
        case "ahead": return "Ahead"
        default: return "Manual"
        }
    }

    private func stateSymbol(_ repository: RepositorySyncItem) -> String {
        if repository.updated || repository.state == "current" { return "checkmark.circle.fill" }
        if repository.canUpdate { return "arrow.up.circle.fill" }
        if repository.state == "ahead" { return "arrow.up.right.circle.fill" }
        return "hand.raised.fill"
    }

    private func stateAccent(_ repository: RepositorySyncItem) -> Color {
        if repository.updated || repository.state == "current" { return BeaconPalette.mint }
        if repository.canUpdate { return BeaconPalette.gold }
        if repository.state == "ahead" { return BeaconPalette.cyan }
        return BeaconPalette.coral
    }
}
