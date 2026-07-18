import AppKit
import Combine

enum TerminalHotKeyStatus: Equatable {
    case inactive
    case registered
    case failed(String)
}

@MainActor
protocol DropDownTerminalWindowControlling: AnyObject {
    var isVisible: Bool { get }
    func toggle(edge: TerminalEdge, height: TerminalHeight)
    func update(edge: TerminalEdge, height: TerminalHeight)
    func terminate()
}

@MainActor
final class DropDownTerminalController: ObservableObject {
    @Published private(set) var hotKeyStatus = TerminalHotKeyStatus.inactive
    @Published private(set) var isVisible = false
    @Published var edge: TerminalEdge {
        didSet {
            defaults.set(edge.rawValue, forKey: TerminalEdge.storageKey)
            windowController?.update(edge: edge, height: height)
        }
    }
    @Published var height: TerminalHeight {
        didSet {
            defaults.set(height.rawValue, forKey: TerminalHeight.storageKey)
            windowController?.update(edge: edge, height: height)
        }
    }

    private let defaults: UserDefaults
    private let registrar: GlobalHotKeyRegistering
    private let makeWindowController: () -> DropDownTerminalWindowControlling
    private var windowController: DropDownTerminalWindowControlling?
    private var started = false

    init(
        defaults: UserDefaults = .standard,
        registrar: GlobalHotKeyRegistering = CarbonGlobalHotKeyRegistrar(),
        makeWindowController: (() -> DropDownTerminalWindowControlling)? = nil
    ) {
        self.defaults = defaults
        self.registrar = registrar
        self.makeWindowController = makeWindowController ?? {
            DropDownTerminalWindowController()
        }
        edge = TerminalEdge(rawValue: defaults.string(forKey: TerminalEdge.storageKey) ?? "")
            ?? .defaultEdge
        height = TerminalHeight(rawValue: defaults.string(forKey: TerminalHeight.storageKey) ?? "")
            ?? .defaultHeight
    }

    func start() {
        guard !started else { return }
        started = true
        do {
            try registrar.register { [weak self] in
                Task { @MainActor in
                    self?.toggle()
                }
            }
            hotKeyStatus = .registered
        } catch {
            hotKeyStatus = .failed(error.localizedDescription)
        }
    }

    func stop() {
        guard started || windowController != nil else { return }
        registrar.unregister()
        windowController?.terminate()
        windowController = nil
        isVisible = false
        started = false
        hotKeyStatus = .inactive
    }

    func toggle() {
        if windowController == nil {
            windowController = makeWindowController()
        }
        isVisible = !(windowController?.isVisible == true)
        windowController?.toggle(edge: edge, height: height)
    }
}
