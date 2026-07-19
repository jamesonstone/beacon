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

enum DashboardViewModePresentation {
    static func notesTransition(
        from previous: DashboardViewMode,
        to next: DashboardViewMode,
        current: SignalNotesSize,
        lastExpanded: SignalNotesSize
    ) -> (current: SignalNotesSize, lastExpanded: SignalNotesSize) {
        let remembered = lastExpanded.isExpanded ? lastExpanded : .half
        if next == .overview {
            let previousExpanded = previous == .fitted
                ? remembered
                : (current.isExpanded ? current : remembered)
            return (.minimized, previousExpanded)
        }
        if next == .fitted {
            let previousExpanded = current.isExpanded ? current : remembered
            return (.half, previousExpanded)
        }
        if previous == .overview || previous == .fitted {
            return (remembered, remembered)
        }
        return (current, lastExpanded)
    }

    static func notesHeight(
        in availableHeight: CGFloat,
        mode: DashboardViewMode,
        size: SignalNotesSize
    ) -> CGFloat? {
        if mode.locksNotesAtHalfHeight {
            return availableHeight * SignalNotesPresentation.expandedHeightFraction
        }
        guard let fraction = size.heightFraction else { return nil }
        return max(220, availableHeight * fraction)
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
