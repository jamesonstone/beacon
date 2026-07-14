import AppKit
import SwiftUI

enum BeaconSwitcherScope {
    case all
    case notes

    var title: String { self == .all ? "Quick Switcher" : "Tab Search" }
    var prompt: String { self == .all ? "Search commands, tabs, projects, and lanes" : "Search Signal Notes" }
}

struct BeaconCommandItem: Identifiable {
    let id: String
    let title: String
    let detail: String
    let symbol: String
    let keywords: String
    let action: @MainActor () -> Void

    func matches(_ query: String) -> Bool {
        let needle = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !needle.isEmpty else { return true }
        return "\(title) \(detail) \(keywords)".lowercased().contains(needle)
    }
}

struct BeaconQuickSwitcher: View {
    let scope: BeaconSwitcherScope
    let commands: [BeaconCommandItem]
    @Binding var query: String
    @Binding var selection: Int
    let dismiss: () -> Void
    @FocusState private var searchFocused: Bool

    private var results: [BeaconCommandItem] {
        commands.filter { $0.matches(query) }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: scope == .all ? "command" : "doc.text.magnifyingglass")
                    .foregroundStyle(BeaconPalette.cyan)
                Text(scope.title)
                    .font(BeaconTypography.semibold(11))
                    .foregroundStyle(BeaconPalette.mint)
                Spacer()
                Text(scope == .all ? "⌘K" : "⌘P")
                    .font(BeaconTypography.regular(8))
                    .foregroundStyle(BeaconPalette.lavender.opacity(0.72))
            }

            TextField(scope.prompt, text: $query)
                .textFieldStyle(.roundedBorder)
                .focused($searchFocused)
                .onSubmit(performSelection)
                .onChange(of: query) { _, _ in selection = 0 }

            if results.isEmpty {
                ContentUnavailableView("No matches", systemImage: "magnifyingglass")
                    .frame(maxWidth: .infinity, minHeight: 120)
            } else {
                ScrollViewReader { proxy in
                    ScrollView {
                        LazyVStack(spacing: 3) {
                            ForEach(Array(results.enumerated()), id: \.element.id) { index, command in
                                Button {
                                    selection = index
                                    performSelection()
                                } label: {
                                    HStack(spacing: 8) {
                                        Image(systemName: command.symbol)
                                            .frame(width: 15)
                                            .foregroundStyle(index == selection ? BeaconPalette.mint : BeaconPalette.cyan)
                                        VStack(alignment: .leading, spacing: 1) {
                                            Text(command.title)
                                                .font(BeaconTypography.medium(9))
                                                .foregroundStyle(BeaconPalette.mint)
                                                .lineLimit(1)
                                            if !command.detail.isEmpty {
                                                Text(command.detail)
                                                    .font(BeaconTypography.regular(7))
                                                    .foregroundStyle(BeaconPalette.lavender.opacity(0.72))
                                                    .lineLimit(1)
                                            }
                                        }
                                        Spacer()
                                        if index == selection {
                                            Image(systemName: "return")
                                                .font(.system(size: 8, weight: .semibold))
                                                .foregroundStyle(BeaconPalette.lavender)
                                        }
                                    }
                                    .padding(.horizontal, 8)
                                    .frame(height: 34)
                                    .contentShape(Rectangle())
                                    .background(
                                        index == selection ? BeaconPalette.softGradient(BeaconPalette.cyan) : BeaconPalette.softGradient(.clear),
                                        in: RoundedRectangle(cornerRadius: 7)
                                    )
                                }
                                .buttonStyle(.plain)
                                .id(command.id)
                            }
                        }
                    }
                    .onChange(of: selection) { _, latest in
                        guard results.indices.contains(latest) else { return }
                        withAnimation(.easeOut(duration: 0.08)) { proxy.scrollTo(results[latest].id) }
                    }
                }
            }

            Text("↑↓ navigate · Return open · Esc dismiss")
                .font(BeaconTypography.regular(7))
                .foregroundStyle(BeaconPalette.lavender.opacity(0.65))
        }
        .padding(12)
        .frame(maxWidth: 390, maxHeight: 390)
        .background(BeaconPalette.panelBackground, in: RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12)
                .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.cyan), lineWidth: 1)
        }
        .shadow(color: Color.black.opacity(0.45), radius: 18, y: 8)
        .task {
            await Task.yield()
            searchFocused = true
        }
        .onKeyPress(.upArrow) { moveSelection(-1); return .handled }
        .onKeyPress(.downArrow) { moveSelection(1); return .handled }
        .onKeyPress(.escape) { dismiss(); return .handled }
    }

    private func moveSelection(_ delta: Int) {
        guard !results.isEmpty else { return }
        selection = (selection + delta + results.count) % results.count
    }

    private func performSelection() {
        guard results.indices.contains(selection) else { return }
        let command = results[selection]
        dismiss()
        command.action()
    }
}

extension MenuView {
    var noteSwitcherCommands: [BeaconCommandItem] {
        var items = [
            BeaconCommandItem(
                id: "note-general", title: "General", detail: "Pinned Signal Note", symbol: "pin.fill", keywords: "notes tab",
                action: { Task { await state.activateNote("general") } }
            ),
            BeaconCommandItem(
                id: "note-new", title: "New Tab", detail: "Create or reopen a detail note", symbol: "plus.square", keywords: "notes create",
                action: { Task { await state.showNewNotePicker() } }
            ),
        ]
        items += state.noteHistory.map { tab in
            BeaconCommandItem(
                id: "note-\(tab.id)", title: tab.id == state.activeNoteID ? state.activeNoteTitle : tab.title,
                detail: tab.isOpen ? "Open · \(tab.id)" : "Closed · \(tab.id)",
                symbol: tab.isOpen ? "rectangle.on.rectangle" : "doc.text", keywords: "signal note tab",
                action: { Task { await state.activateNote(tab.id) } }
            )
        }
        return items
    }

    var allSwitcherCommands: [BeaconCommandItem] {
        var items = noteSwitcherCommands
        items += DashboardTab.allCases.map { tab in
            BeaconCommandItem(
                id: "dashboard-\(tab.rawValue)", title: tab.title, detail: "Dashboard destination",
                symbol: tab.symbol, keywords: "dashboard lane view",
                action: { showDashboardTab(tab) }
            )
        }
        items += [
            BeaconCommandItem(
                id: "scan", title: "Scan Now", detail: "Refresh Git and GitHub evidence", symbol: "arrow.clockwise", keywords: "refresh",
                action: { Task { await state.scan() } }
            ),
            BeaconCommandItem(
                id: "open-top", title: "Open Top Item", detail: "Open the first actionable lane", symbol: "arrow.up.forward.app", keywords: "lane",
                action: { state.openTopItem() }
            ),
            BeaconCommandItem(
                id: "manual-lane", title: "Add Manual Lane", detail: "Create a planning or research lane", symbol: "plus.circle", keywords: "project action",
                action: { manualTitle = ""; showingManualEditor = true }
            ),
            BeaconCommandItem(
                id: "projects", title: "Manage Following", detail: "Choose tracked projects", symbol: "star", keywords: "projects settings",
                action: { showProjects(.following) }
            ),
            BeaconCommandItem(
                id: "repository-sync", title: "Repository Sync", detail: "Inspect local default branches", symbol: "arrow.triangle.2.circlepath", keywords: "git projects",
                action: { showRepositorySync() }
            ),
            BeaconCommandItem(
                id: "dependency-limits", title: "Dependency Limits", detail: "Inspect current service allowance", symbol: "gauge.with.dots.needle.50percent", keywords: "github gh usage",
                action: { showDependencyLimits() }
            ),
            BeaconCommandItem(
                id: "config", title: "Open Config", detail: "Open Beacon configuration", symbol: "slider.horizontal.3", keywords: "settings yaml",
                action: { state.openConfig() }
            ),
            BeaconCommandItem(
                id: "dashboard-window", title: "Open Dashboard", detail: "Show the detachable Beacon window", symbol: "macwindow", keywords: "window",
                action: openDashboard
            ),
            BeaconCommandItem(
                id: "quit", title: "Quit Beacon", detail: "Close the application", symbol: "power", keywords: "app",
                action: { NSApplication.shared.terminate(nil) }
            ),
        ]
        if state.activeNoteID == "general", !state.notesCurrentLine.isEmpty {
            items.append(BeaconCommandItem(
                id: "note-from-line", title: "Create Detail From Current Line", detail: state.notesCurrentLine,
                symbol: "text.line.first.and.arrowtriangle.forward", keywords: "signal note",
                action: { Task { await state.createNoteFromCurrentLine() } }
            ))
        }
        if state.notesAreDirty {
            items += [
                BeaconCommandItem(
                    id: "note-save", title: "Save Current Note", detail: state.activeNoteTitle, symbol: "square.and.arrow.down", keywords: "signal note",
                    action: { Task { await state.saveNotes(state.notesDraft) } }
                ),
                BeaconCommandItem(
                    id: "note-revert", title: "Revert Current Note", detail: state.activeNoteTitle, symbol: "arrow.uturn.backward", keywords: "signal note",
                    action: { state.revertNotes() }
                ),
            ]
        }
        items += DashboardViewMode.allCases.map { mode in
            BeaconCommandItem(
                id: "view-\(mode.rawValue)", title: "View: \(mode.title)", detail: "Change dashboard layout",
                symbol: mode.symbol, keywords: "settings display",
                action: { viewMode = mode }
            )
        }
        let lanes: [WorkLane] = state.snapshot?.lanes ?? []
        items.append(contentsOf: lanes.compactMap { lane -> BeaconCommandItem? in
            guard AppState.openTarget(for: lane) != nil else { return nil }
            let title = lane.attention?.title ?? lane.pullRequest?.title ?? lane.issue?.title ?? lane.branch
            return BeaconCommandItem(
                id: "lane-\(lane.id)", title: title, detail: lane.repository,
                symbol: "arrow.up.forward.app", keywords: "lane project \(lane.github)",
                action: { state.open(lane) }
            )
        })
        return items
    }

    func showSwitcher(_ scope: BeaconSwitcherScope) {
        switcherQuery = ""
        switcherSelection = 0
        switcherScope = scope
    }

    func showRepositorySync() {
        dashboardDestination = .repositorySync
        if state.repositorySyncReport == nil, !state.isCheckingRepositorySync {
            Task { await state.checkRepositorySync(refresh: false) }
        }
    }

    func showDependencyLimits() {
        dashboardDestination = .dependencyLimits
        if state.dependencyLimitsReport == nil, !state.isCheckingDependencyLimits {
            Task { await state.checkDependencyLimits() }
        }
    }
}
