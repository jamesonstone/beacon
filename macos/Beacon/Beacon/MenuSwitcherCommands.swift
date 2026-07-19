import AppKit
import SwiftUI

extension MenuView {
    var noteSwitcherCommands: [BeaconCommandItem] {
        var items = [
            BeaconCommandItem.notesAssistant(
                detail: notesAssistantCommandDetail,
                action: { showNotesAssistant(.compact) }
            ),
            BeaconCommandItem(
                id: "note-general", title: "General", detail: "Pinned Note", symbol: "pin.fill", keywords: "signal notes tab",
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
                deletableNote: tab,
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
                id: "note-from-line", title: SignalNotesPresentation.createFromGeneralLabel, detail: state.notesCurrentLine,
                symbol: SignalNotesPresentation.createFromGeneralSymbol, keywords: "signal note",
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

extension BeaconCommandItem {
    static func notesAssistant(
        detail: String,
        action: @escaping @MainActor () -> Void
    ) -> BeaconCommandItem {
        BeaconCommandItem(
            id: "notes-assistant",
            title: NotesAssistantPresentation.quickSwitcherTitle,
            detail: detail,
            symbol: NotesAssistantPresentation.buttonSymbol,
            keywords: NotesAssistantPresentation.quickSwitcherKeywords,
            action: action
        )
    }
}
