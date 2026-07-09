import SwiftUI

@main
struct BeaconApp: App {
    @StateObject private var state = AppState()

    var body: some Scene {
        MenuBarExtra {
            MenuView(state: state)
        } label: {
            Label("Beacon \(state.readyCount)", systemImage: "dot.radiowaves.left.and.right")
                .task { state.start() }
        }
        .menuBarExtraStyle(.window)
    }
}
