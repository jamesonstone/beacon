import SwiftUI

extension MenuView {
    var terminalSettingsMenu: some View {
        Menu {
            Button { terminal.toggle() } label: {
                Label(terminal.isVisible ? "Hide Terminal" : "Show Terminal", systemImage: "terminal")
            }

            Menu {
                Picker("Position", selection: $terminal.edge) {
                    ForEach(TerminalEdge.allCases) { edge in
                        Label(edge.title, systemImage: edge.symbol).tag(edge)
                    }
                }
            } label: {
                Label("Position: \(terminal.edge.title)", systemImage: terminal.edge.symbol)
            }

            Menu {
                Picker("Height", selection: $terminal.height) {
                    ForEach(TerminalHeight.allCases) { height in
                        Text("\(height.title) · \(Int(height.fraction * 100))%").tag(height)
                    }
                }
            } label: {
                Label("Height: \(terminal.height.title)", systemImage: "arrow.up.and.down")
            }

            Divider()
            hotKeyStatusRow
            Divider()

            if WarpTerminalIntegration.isInstalled {
                Text("Warp cannot be embedded; use its own hotkey window with a different shortcut.")
                Button("Open Warp", systemImage: "terminal.fill") {
                    WarpTerminalIntegration.openApplication()
                }
                Button("Warp Hotkey Setup Guide", systemImage: "questionmark.circle") {
                    WarpTerminalIntegration.openGuide()
                }
            } else {
                Text("Warp is not installed; Beacon uses its built-in terminal.")
                Button("Warp Hotkey Setup Guide", systemImage: "questionmark.circle") {
                    WarpTerminalIntegration.openGuide()
                }
            }
        } label: {
            Label("Terminal", systemImage: terminalSettingsSymbol)
        }
    }

    @ViewBuilder
    private var hotKeyStatusRow: some View {
        switch terminal.hotKeyStatus {
        case .inactive:
            Label("Shortcut inactive", systemImage: "minus.circle")
        case .registered:
            Label("Global Shortcut: ⌘J", systemImage: "checkmark.circle")
        case .failed(let message):
            Label("Shortcut unavailable", systemImage: "exclamationmark.triangle.fill")
                .help(message)
            Text(message)
        }
    }

    private var terminalSettingsSymbol: String {
        if case .failed = terminal.hotKeyStatus {
            return "exclamationmark.terminal"
        }
        return "terminal"
    }
}
