import SwiftUI

enum SignalNotesSize: String, Equatable {
    case half
    case eighty
    case minimized

    var isExpanded: Bool { self != .minimized }

    var heightFraction: Double? {
        switch self {
        case .half: 0.5
        case .eighty: 0.8
        case .minimized: nil
        }
    }

    var nextCycled: SignalNotesSize {
        switch self {
        case .half: .eighty
        case .eighty: .minimized
        case .minimized: .half
        }
    }
}

enum SignalNotesPresentation {
    static let expandedByDefault = true
    static let expandedHeightFraction = 0.5
    static let enlargedHeightFraction = 0.8
    static let sizeStorageKey = "beacon.signal-notes-size"
    static let lastExpandedStorageKey = "beacon.signal-notes-last-expanded-size"
    static let autosaveDelay: Duration = .seconds(3)
    static let createFromGeneralLabel = "Create New Note from Highlighted Text in General"
    static let createFromGeneralSymbol = "doc.badge.plus"

    static func savedLabel(age: String) -> String {
        "Saved \(age)"
    }
}

enum DashboardOverviewPresentation {
    static func notesTransition(
        from previous: DashboardViewMode,
        to next: DashboardViewMode,
        current: SignalNotesSize,
        lastExpanded: SignalNotesSize
    ) -> (current: SignalNotesSize, lastExpanded: SignalNotesSize) {
        if next == .overview {
            return (.minimized, current.isExpanded ? current : lastExpanded)
        }
        if previous == .overview, current == .minimized {
            return (lastExpanded.isExpanded ? lastExpanded : .half, lastExpanded)
        }
        return (current, lastExpanded)
    }
}

@MainActor
final class SignalNotesAutosave: ObservableObject {
    private let delay: Duration
    private var pendingTask: Task<Void, Never>?
    private var generation = 0

    init(delay: Duration = SignalNotesPresentation.autosaveDelay) {
        self.delay = delay
    }

    func schedule(
        content: String,
        save: @escaping @MainActor (String) async -> Void
    ) {
        generation += 1
        let scheduledGeneration = generation
        pendingTask?.cancel()
        let delay = delay
        pendingTask = Task { [weak self] in
            do {
                try await Task.sleep(for: delay)
            } catch {
                return
            }
            guard !Task.isCancelled, self?.generation == scheduledGeneration else { return }
            await save(content)
            if self?.generation == scheduledGeneration {
                self?.pendingTask = nil
            }
        }
    }

    func cancel() {
        generation += 1
        pendingTask?.cancel()
        pendingTask = nil
    }
}
