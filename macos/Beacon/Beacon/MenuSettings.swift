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
                Menu {
                    ForEach(BeaconThemeCatalog.all) { candidate in
                        Button {
                            themeIDValue = candidate.id.rawValue
                            terminal.refreshAppearance()
                        } label: {
                            HStack(spacing: 7) {
                                BeaconThemePreview(
                                    theme: candidate,
                                    isSelected: theme.id == candidate.id
                                )
                                VStack(alignment: .leading, spacing: 1) {
                                    Text(candidate.name)
                                    Text(candidate.detail)
                                        .foregroundStyle(candidate.tokens.textMuted.color)
                                }
                                Spacer()
                                if theme.id == candidate.id {
                                    Image(systemName: "checkmark.circle.fill")
                                }
                            }
                        }
                        .accessibilityLabel(candidate.accessibilityName)
                        .help(candidate.detail)
                    }
                } label: {
                    Label("Theme: \(theme.name)", systemImage: "paintpalette")
                }
                Menu {
                    Picker("Font", selection: $fontFamilyValue) {
                        ForEach(BeaconFontCatalog.selectionOptions, id: \.self) { family in
                            Text(
                                family == BeaconTypography.defaultFamily
                                    ? "\(family) — Default"
                                    : family
                            )
                            .font(.custom(family, size: 12))
                            .tag(family)
                        }
                    }
                } label: {
                    Label(
                        "Font: \(BeaconFontCatalog.displayName(for: fontFamilyValue))",
                        systemImage: "textformat"
                    )
                }
                Menu {
                    Picker("Font Size", selection: $fontSizeValue) {
                        ForEach(BeaconFontSize.allCases) { size in
                            Text(size.title).tag(size.rawValue)
                        }
                    }
                } label: {
                    Label("Text Size: \(fontSizeValue) pt", systemImage: "textformat.size")
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
            } label: {
                Label("Appearance", systemImage: "circle.lefthalf.filled")
            }
            terminalSettingsMenu
            ollamaSettingsMenu
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
                .foregroundStyle(theme.tokens.accent.color)
                .frame(width: 28, height: 28)
                .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(interfaceBorderColor, lineWidth: colorSchemeContrast == .increased ? 1.1 : 0.7)
                }
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .fixedSize()
        .help("Settings")
    }

    private var ollamaSettingsMenu: some View {
        Menu {
            if state.isLoadingOllamaModels {
                Label("Loading local models…", systemImage: "arrow.triangle.2.circlepath")
            } else if state.ollamaModels.isEmpty {
                Text(state.ollamaError ?? "No local Ollama models found")
            } else {
                ForEach(state.ollamaModels) { model in
                    Button {
                        Task { await state.setOllamaDefaultModel(model.name) }
                    } label: {
                        HStack {
                            Text(model.name)
                            if model.name == state.ollamaConfiguredModel {
                                Image(systemName: "checkmark.circle.fill")
                            }
                        }
                    }
                    .disabled(state.isSavingOllamaDefault)
                }
            }
            if let notice = state.ollamaNotice {
                Divider()
                Text(notice)
            }
            Divider()
            Button {
                Task { await state.refreshOllamaModels() }
            } label: {
                Label("Refresh Local Models", systemImage: "arrow.clockwise")
            }
            .disabled(state.isLoadingOllamaModels)
        } label: {
            Label(
                "Ollama Model: \(ollamaDefaultModelLabel)",
                systemImage: "brain.head.profile"
            )
        }
    }

    private var ollamaDefaultModelLabel: String {
        state.ollamaConfiguredModel.isEmpty ? "Automatic" : state.ollamaConfiguredModel
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
