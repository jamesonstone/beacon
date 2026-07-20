import SwiftUI

struct FittedFollowingLayout: Equatable {
    let columns: Int
    let rows: Int
    let scale: CGFloat
    let contentSize: CGSize
}

enum FittedFollowingPresentation {
    static let cardSize = CGSize(width: 220, height: 88)
    static let itemSpacing: CGFloat = 6
    static let sectionSpacing: CGFloat = 8
    static let sectionHeaderHeight: CGFloat = 18
    static let sectionHeaderSpacing: CGFloat = 4

    static func layout(
        in availableSize: CGSize,
        sectionItemCounts: [Int]
    ) -> FittedFollowingLayout {
        let counts = sectionItemCounts.filter { $0 > 0 }
        guard availableSize.width > 0,
              availableSize.height > 0,
              let maximumColumns = counts.max() else {
            return FittedFollowingLayout(columns: 1, rows: 0, scale: 1, contentSize: .zero)
        }

        var best: FittedFollowingLayout?
        for columns in 1...maximumColumns {
            let sectionRows = counts.map { Int(ceil(Double($0) / Double(columns))) }
            let rows = sectionRows.reduce(0, +)
            let width = CGFloat(columns) * cardSize.width
                + CGFloat(columns - 1) * itemSpacing
            let gridHeight = sectionRows.reduce(CGFloat.zero) { total, rowCount in
                total + CGFloat(rowCount) * cardSize.height
                    + CGFloat(max(0, rowCount - 1)) * itemSpacing
            }
            let height = gridHeight
                + CGFloat(counts.count) * (sectionHeaderHeight + sectionHeaderSpacing)
                + CGFloat(max(0, counts.count - 1)) * sectionSpacing
            let scale = min(1, availableSize.width / width, availableSize.height / height)
            let candidate = FittedFollowingLayout(
                columns: columns,
                rows: rows,
                scale: max(0, scale),
                contentSize: CGSize(width: width, height: height)
            )
            if let current = best {
                let improvesScale = candidate.scale > current.scale + 0.0001
                let tiesWithFewerRows = abs(candidate.scale - current.scale) <= 0.0001
                    && candidate.rows < current.rows
                if improvesScale || tiesWithFewerRows {
                    best = candidate
                }
            } else {
                best = candidate
            }
        }
        return best ?? FittedFollowingLayout(columns: 1, rows: 0, scale: 1, contentSize: .zero)
    }
}

private struct FittedFollowingSection: Identifiable {
    let id: String
    let title: String
    let symbol: String
    let accent: Color
    let lanes: [WorkLane]
}

extension MenuView {
    func fittedFollowingDashboard(_ snapshot: BeaconSnapshot) -> some View {
        let sections = fittedFollowingSections(snapshot)
        let loadingProjects = state.loadingProjects
        let counts = (loadingProjects.isEmpty ? [] : [loadingProjects.count])
            + sections.map { $0.lanes.count }

        return GeometryReader { geometry in
            let layout = FittedFollowingPresentation.layout(
                in: geometry.size,
                sectionItemCounts: counts
            )
            fittedFollowingContent(
                sections: sections,
                loadingProjects: loadingProjects,
                layout: layout
            )
            .frame(
                width: layout.contentSize.width,
                height: layout.contentSize.height,
                alignment: .topLeading
            )
            .scaleEffect(layout.scale)
            .position(x: geometry.size.width / 2, y: geometry.size.height / 2)
        }
        .clipped()
    }

    private func fittedFollowingContent(
        sections: [FittedFollowingSection],
        loadingProjects: [AgentProjectStatus],
        layout: FittedFollowingLayout
    ) -> some View {
        VStack(alignment: .leading, spacing: FittedFollowingPresentation.sectionSpacing) {
            if !loadingProjects.isEmpty {
                fittedLoadingSection(loadingProjects, columns: layout.columns)
            }
            ForEach(sections) { section in
                VStack(alignment: .leading, spacing: FittedFollowingPresentation.sectionHeaderSpacing) {
                    sectionHeader(
                        section.title,
                        symbol: section.symbol,
                        accent: section.accent,
                        count: section.lanes.count
                    )
                    .frame(height: FittedFollowingPresentation.sectionHeaderHeight)
                    fittedLaneRows(section.lanes, columns: layout.columns)
                }
            }
        }
    }

    private func fittedLaneRows(_ lanes: [WorkLane], columns: Int) -> some View {
        let rowCount = Int(ceil(Double(lanes.count) / Double(columns)))
        return VStack(spacing: FittedFollowingPresentation.itemSpacing) {
            ForEach(0..<rowCount, id: \.self) { row in
                HStack(spacing: FittedFollowingPresentation.itemSpacing) {
                    ForEach(0..<columns, id: \.self) { column in
                        let index = row * columns + column
                        if lanes.indices.contains(index) {
                            laneCard(
                                lanes[index],
                                density: .dense,
                                watermarkProjectName: state.projectGroup(for: lanes[index]).name
                            )
                                .frame(
                                    width: FittedFollowingPresentation.cardSize.width,
                                    height: FittedFollowingPresentation.cardSize.height,
                                    alignment: .top
                                )
                                .clipped()
                        } else {
                            fittedGridSpacer
                        }
                    }
                }
            }
        }
    }

    private func fittedLoadingSection(
        _ projects: [AgentProjectStatus],
        columns: Int
    ) -> some View {
        let rowCount = Int(ceil(Double(projects.count) / Double(columns)))
        return VStack(alignment: .leading, spacing: FittedFollowingPresentation.sectionHeaderSpacing) {
            sectionHeader(
                "Loading Projects",
                symbol: "antenna.radiowaves.left.and.right",
                accent: BeaconThemePreference.current().tokens.info.color,
                count: projects.count
            )
            .frame(height: FittedFollowingPresentation.sectionHeaderHeight)
            VStack(spacing: FittedFollowingPresentation.itemSpacing) {
                ForEach(0..<rowCount, id: \.self) { row in
                    HStack(spacing: FittedFollowingPresentation.itemSpacing) {
                        ForEach(0..<columns, id: \.self) { column in
                            let index = row * columns + column
                            if projects.indices.contains(index) {
                                fittedLoadingCard(projects[index])
                            } else {
                                fittedGridSpacer
                            }
                        }
                    }
                }
            }
        }
    }

    private func fittedLoadingCard(_ project: AgentProjectStatus) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 7) {
                ProgressView().controlSize(.mini)
                Text(project.name).font(BeaconTypography.semibold(10)).lineLimit(1)
                Spacer()
            }
            Text(stageLabel(project.stage))
                .font(BeaconTypography.regular(9))
                .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
            Text(project.projectID)
                .font(BeaconTypography.identifier(9))
                .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                .lineLimit(1)
        }
        .padding(DashboardDensity.dense.cardPadding)
        .frame(
            width: FittedFollowingPresentation.cardSize.width,
            height: FittedFollowingPresentation.cardSize.height,
            alignment: .topLeading
        )
        .background(
            BeaconThemePreference.current().tokens.surface.color,
            in: RoundedRectangle(cornerRadius: 9)
        )
        .overlay {
            RoundedRectangle(cornerRadius: 9).strokeBorder(interfaceBorderColor, lineWidth: 0.8)
        }
    }

    private var fittedGridSpacer: some View {
        Color.clear.frame(
            width: FittedFollowingPresentation.cardSize.width,
            height: FittedFollowingPresentation.cardSize.height
        )
    }

    private func fittedFollowingSections(_ snapshot: BeaconSnapshot) -> [FittedFollowingSection] {
        if let working = snapshot.workingSet {
            return [
                FittedFollowingSection(
                    id: "active", title: "Active", symbol: "bolt.fill",
                    accent: BeaconThemePreference.current().tokens.success.color,
                    lanes: state.lanes(for: working.active)
                ),
                FittedFollowingSection(
                    id: "waiting", title: "Waiting", symbol: "clock.fill",
                    accent: BeaconThemePreference.current().tokens.warning.color,
                    lanes: state.lanes(for: working.waiting)
                ),
                FittedFollowingSection(
                    id: "recent", title: "Recently Active", symbol: "sparkles",
                    accent: BeaconThemePreference.current().tokens.info.color,
                    lanes: state.lanes(for: working.recent)
                ),
            ].filter { !$0.lanes.isEmpty }
        }
        return [
            FittedFollowingSection(
                id: "ready", title: "Ready for Review", symbol: "checkmark.circle.fill",
                accent: BeaconThemePreference.current().tokens.success.color,
                lanes: state.lanes(for: snapshot.groups.ready)
            ),
            FittedFollowingSection(
                id: "action", title: "Needs Action", symbol: "exclamationmark.triangle.fill",
                accent: BeaconThemePreference.current().tokens.danger.color,
                lanes: state.lanes(for: snapshot.groups.action)
            ),
            FittedFollowingSection(
                id: "waiting", title: "Waiting", symbol: "clock.fill",
                accent: BeaconThemePreference.current().tokens.warning.color,
                lanes: state.lanes(for: snapshot.groups.waiting)
            ),
        ].filter { !$0.lanes.isEmpty }
    }
}
