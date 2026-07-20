import AppKit
import SwiftUI

extension MenuView {
    var settingsMenu: some View {
        Menu {
            if surface == .menu {
                Button(action: openDashboard) { Label("Open Dashboard", systemImage: "macwindow") }
            }
            Button { state.openTopItem() } label: { Label("Open Top Item", systemImage: "arrow.up.forward.app") }
                .disabled(state.inProgressCount == 0)
            Button { manualTitle = ""; showingManualEditor = true } label: {
                Label("Add Manual Lane", systemImage: "plus.circle")
            }
            Divider()
            Button { showProjects(.following) } label: {
                Label(
                    dashboardDestination == .projectInventory ? "Return to Following" : "Manage Following",
                    systemImage: "star"
                )
            }
            Button { state.openConfig() } label: {
                Label("Open Config", systemImage: "slider.horizontal.3")
            }
            settingsPanelButton(.appearance)
            settingsPanelButton(.terminal)
            settingsPanelButton(.ollama)
            Button {
                dismissedEvidenceBadgesValue = "[]"
            } label: {
                Label("Restore Hidden Badges", systemImage: "eye")
            }
            .disabled(dismissedEvidenceBadges.isEmpty)
            settingsPanelButton(.agentHooks)
            Divider()
            Toggle(
                "Open Beacon at Login",
                isOn: Binding(
                    get: { loginItem.isEnabled },
                    set: { loginItem.setEnabled($0) }
                )
            )
            if loginItem.requiresApproval {
                Button("Approve in Settings") {
                    loginItem.openSystemSettings()
                }
            }
            if !state.agentAvailable {
                Button { Task { await state.enableAgent() } } label: {
                    Label("Enable Background Agent", systemImage: "antenna.radiowaves.left.and.right")
                }
            }
            Divider()
            Button("Quit Beacon", systemImage: "power") { NSApplication.shared.terminate(nil) }
        } label: {
            Image(systemName: "gearshape.fill")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(theme.tokens.accent.color)
                .frame(width: 28, height: 28)
                .background(
                    BeaconThemePreference.current().tokens.surfaceRaised.color,
                    in: RoundedRectangle(cornerRadius: 8)
                )
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(
                            interfaceBorderColor,
                            lineWidth: colorSchemeContrast == .increased ? 1.1 : 0.7
                        )
                }
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .fixedSize()
        .help("Settings")
        .popover(item: $settingsPanel, arrowEdge: .top) { panel in
            settingsPanelContent(panel)
        }
    }
}
