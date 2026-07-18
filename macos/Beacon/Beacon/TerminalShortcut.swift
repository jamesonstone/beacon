import AppKit
import Foundation

protocol TerminalShortcutRegistering: AnyObject {
    func register(handler: @escaping () -> Void) throws
    func unregister()
}

enum TerminalShortcutRegistrationError: LocalizedError {
    case unavailable

    var errorDescription: String? {
        "Beacon could not listen for Command-J while the application is active."
    }
}

enum TerminalShortcut {
    static func matches(_ event: NSEvent) -> Bool {
        guard event.type == .keyDown,
              event.charactersIgnoringModifiers?.lowercased() == "j" else {
            return false
        }
        let modifiers = event.modifierFlags.intersection(.deviceIndependentFlagsMask)
        let disallowedModifiers: NSEvent.ModifierFlags = [.control, .option, .shift]
        return modifiers.contains(.command)
            && modifiers.intersection(disallowedModifiers).isEmpty
    }
}

final class AppLocalTerminalShortcutRegistrar: TerminalShortcutRegistering {
    private var monitor: Any?

    func register(handler: @escaping () -> Void) throws {
        guard monitor == nil else { return }
        guard let monitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown, handler: { event in
            guard TerminalShortcut.matches(event) else { return event }
            if !event.isARepeat {
                handler()
            }
            return nil
        }) else {
            throw TerminalShortcutRegistrationError.unavailable
        }
        self.monitor = monitor
    }

    func unregister() {
        if let monitor {
            NSEvent.removeMonitor(monitor)
        }
        monitor = nil
    }

    deinit {
        unregister()
    }
}
