import Carbon
import Foundation

protocol GlobalHotKeyRegistering: AnyObject {
    func register(handler: @escaping () -> Void) throws
    func unregister()
}

struct GlobalHotKeyRegistrationError: LocalizedError, Equatable {
    let status: OSStatus

    var errorDescription: String? {
        if status == eventHotKeyExistsErr {
            return "Command-J is already registered by another application. Change that shortcut, then relaunch Beacon."
        }
        return "Beacon could not register Command-J (macOS error \(status))."
    }
}

private final class GlobalHotKeyHandlerBox {
    let handler: () -> Void

    init(handler: @escaping () -> Void) {
        self.handler = handler
    }
}

final class CarbonGlobalHotKeyRegistrar: GlobalHotKeyRegistering {
    private static let signature: OSType = 0x42434E54 // BCNT
    private var hotKey: EventHotKeyRef?
    private var eventHandler: EventHandlerRef?
    private var handlerBox: GlobalHotKeyHandlerBox?

    func register(handler: @escaping () -> Void) throws {
        guard hotKey == nil else { return }

        let box = GlobalHotKeyHandlerBox(handler: handler)
        var eventType = EventTypeSpec(
            eventClass: OSType(kEventClassKeyboard),
            eventKind: UInt32(kEventHotKeyPressed)
        )
        let installStatus = InstallEventHandler(
            GetApplicationEventTarget(),
            { _, _, userData in
                guard let userData else { return OSStatus(eventNotHandledErr) }
                let box = Unmanaged<GlobalHotKeyHandlerBox>.fromOpaque(userData).takeUnretainedValue()
                box.handler()
                return noErr
            },
            1,
            &eventType,
            Unmanaged.passUnretained(box).toOpaque(),
            &eventHandler
        )
        guard installStatus == noErr else {
            throw GlobalHotKeyRegistrationError(status: installStatus)
        }
        handlerBox = box

        var reference: EventHotKeyRef?
        let hotKeyStatus = RegisterEventHotKey(
            UInt32(kVK_ANSI_J),
            UInt32(cmdKey),
            EventHotKeyID(signature: Self.signature, id: 1),
            GetApplicationEventTarget(),
            0,
            &reference
        )
        guard hotKeyStatus == noErr else {
            if let eventHandler {
                RemoveEventHandler(eventHandler)
            }
            self.eventHandler = nil
            handlerBox = nil
            throw GlobalHotKeyRegistrationError(status: hotKeyStatus)
        }
        hotKey = reference
    }

    func unregister() {
        if let hotKey {
            UnregisterEventHotKey(hotKey)
        }
        if let eventHandler {
            RemoveEventHandler(eventHandler)
        }
        hotKey = nil
        eventHandler = nil
        handlerBox = nil
    }

    deinit {
        unregister()
    }
}
