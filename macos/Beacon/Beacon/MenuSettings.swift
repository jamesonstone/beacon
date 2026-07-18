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
            Button { manualTitle = ""; showingManualEditor = true } label: { Label("Add Manual Lane", systemImage: "plus.circle") }
            Divider()
            Button { showProjects(.following) } label: {
                Label(
                    dashboardDestination == .projectInventory ? "Return to Following" : "Manage Following",
                    systemImage: "star"
                )
            }
            Button { state.openConfig() } label: { Label("Open Config", systemImage: "slider.horizontal.3") }
            Menu {
                Picker("Font", selection: $fontFamilyValue) {
                    ForEach(BeaconFontFamily.allCases) { family in
                        Text(family.title).tag(family.rawValue)
                    }
                }
            } label: {
                Label("Font: \(BeaconFontFamily(rawValue: fontFamilyValue)?.title ?? BeaconTypography.defaultFamily.title)", systemImage: "textformat")
            }
            Menu {
                Picker("Font Size", selection: $fontSizeValue) {
                    ForEach(BeaconFontSize.allCases) { size in
                        Text(size.title).tag(size.rawValue)
                    }
                }
            } label: {
                Label("Font Size: \(fontSizeValue) pt", systemImage: "textformat.size")
            }
            Menu {
                Picker("Card Density", selection: $densityValue) {
                    ForEach(DashboardDensity.allCases) { density in
                        Label(density.title, systemImage: density.symbol).tag(density.rawValue)
                    }
                }
            } label: {
                let selected = DashboardDensity(rawValue: densityValue) ?? .comfortable
                Label("Card Density: \(selected.title)", systemImage: selected.symbol)
            }
            Button {
                dismissedEvidenceBadgesValue = "[]"
            } label: {
                Label("Restore Hidden Badges", systemImage: "eye")
            }
            .disabled(dismissedEvidenceBadges.isEmpty)
            Menu {
                integrationHealthRow(provider: "codex", title: "Codex")
                integrationHealthRow(provider: "claude-code", title: "Claude Code")
                Divider()
                Text("Installed means configured; active means Beacon observed the current callback.")
                Text("Codex may require hook trust. Claude Code may be blocked by managed policy.")
                Button { Task { await state.refreshIntegrationHealth() } } label: {
                    Label("Refresh Integration Health", systemImage: "arrow.clockwise")
                }
            } label: {
                Label("Agent Hook Health", systemImage: "bolt.horizontal.circle")
            }
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
                .foregroundStyle(BeaconPalette.neonGradient)
                .frame(width: 28, height: 28)
                .background(BeaconPalette.softGradient(BeaconPalette.lavender), in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(BeaconPalette.borderGradient(BeaconPalette.lavender), lineWidth: 0.7)
                }
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .fixedSize()
        .help("Settings")
    }

    @ViewBuilder
    private func integrationHealthRow(provider: String, title: String) -> some View {
        if let status = state.integrationHealth[provider] {
            Label(
                "\(title): \(integrationHealthLabel(status.state))",
                systemImage: integrationHealthSymbol(status.state)
            )
            .help(status.message ?? status.settingsPath)
        } else {
            Label("\(title): unavailable", systemImage: "questionmark.circle")
        }
    }

    private func integrationHealthLabel(_ value: String) -> String {
        value.replacingOccurrences(of: "_", with: " ").capitalized
    }

    private func integrationHealthSymbol(_ value: String) -> String {
        switch value {
        case "active": return "checkmark.circle.fill"
        case "installed": return "checkmark.circle"
        case "stale": return "exclamationmark.circle"
        case "error": return "exclamationmark.triangle.fill"
        default: return "minus.circle"
        }
    }
}
