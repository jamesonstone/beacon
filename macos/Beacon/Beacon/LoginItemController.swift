import ServiceManagement

enum LoginItemStatus: Equatable {
    case disabled
    case enabled
    case requiresApproval
    case notFound
}

protocol LoginItemServiceProtocol {
    var status: LoginItemStatus { get }
    func register() throws
    func unregister() throws
    func openSystemSettings()
}

struct LoginItemService: LoginItemServiceProtocol {
    static let helperIdentifier = "com.jamesonstone.beacon.login-item"

    private var service: SMAppService {
        SMAppService.loginItem(identifier: Self.helperIdentifier)
    }

    var status: LoginItemStatus {
        switch service.status {
        case .enabled: .enabled
        case .requiresApproval: .requiresApproval
        case .notFound: .notFound
        case .notRegistered: .disabled
        @unknown default: .notFound
        }
    }

    func register() throws {
        try service.register()
    }

    func unregister() throws {
        try service.unregister()
    }

    func openSystemSettings() {
        SMAppService.openSystemSettingsLoginItems()
    }
}

@MainActor
final class LoginItemController: ObservableObject {
    @Published private(set) var status: LoginItemStatus
    @Published private(set) var errorMessage: String?

    private let service: LoginItemServiceProtocol

    init(service: LoginItemServiceProtocol = LoginItemService()) {
        self.service = service
        status = service.status
    }

    var isEnabled: Bool {
        status == .enabled
    }

    var requiresApproval: Bool {
        status == .requiresApproval
    }

    func refresh() {
        status = service.status
    }

    func setEnabled(_ enabled: Bool) {
        errorMessage = nil
        do {
            if enabled {
                try service.register()
            } else {
                try service.unregister()
            }
            status = service.status
        } catch {
            status = service.status
            errorMessage = error.localizedDescription
        }
    }

    func openSystemSettings() {
        service.openSystemSettings()
    }
}
