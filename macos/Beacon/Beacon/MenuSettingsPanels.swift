import SwiftUI

enum BeaconSettingsPanel: String, CaseIterable, Identifiable {
    case appearance
    case terminal
    case ollama
    case agentHooks

    var id: String { rawValue }

    var title: String {
        switch self {
        case .appearance: "Appearance"
        case .terminal: "Terminal"
        case .ollama: "Ollama Model"
        case .agentHooks: "Agent Hook Health"
        }
    }

    var symbol: String {
        switch self {
        case .appearance: "circle.lefthalf.filled"
        case .terminal: "terminal"
        case .ollama: "brain.head.profile"
        case .agentHooks: "bolt.horizontal.circle"
        }
    }

    var height: CGFloat {
        switch self {
        case .appearance: 430
        case .terminal: 330
        case .ollama: 320
        case .agentHooks: 240
        }
    }
}

extension MenuView {
    func settingsPanelButton(_ panel: BeaconSettingsPanel) -> some View {
        Button {
            presentSettingsPanel(panel)
        } label: {
            Label(settingsPanelMenuTitle(panel), systemImage: settingsPanelSymbol(panel))
        }
        .accessibilityLabel("Open \(panel.title) settings")
    }

    func settingsPanelContent(_ panel: BeaconSettingsPanel) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Label(panel.title, systemImage: settingsPanelSymbol(panel))
                    .font(BeaconTypography.bold(14))
                    .foregroundStyle(theme.tokens.accent.color)
                Spacer()
                Button {
                    settingsPanel = nil
                } label: {
                    Image(systemName: "xmark")
                }
                .buttonStyle(.plain)
                .help("Close \(panel.title) settings")
                .accessibilityLabel("Close \(panel.title) settings")
            }
            Divider()
            ScrollView {
                settingsPanelBody(panel)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .padding(14)
        .frame(width: 390, height: panel.height, alignment: .top)
        .background(theme.tokens.surfaceOverlay.color)
        .onExitCommand { settingsPanel = nil }
        .accessibilityElement(children: .contain)
        .accessibilityLabel("\(panel.title) settings")
    }

    private func presentSettingsPanel(_ panel: BeaconSettingsPanel) {
        Task { @MainActor in
            await Task.yield()
            settingsPanel = panel
        }
    }

    private func settingsPanelMenuTitle(_ panel: BeaconSettingsPanel) -> String {
        guard panel == .ollama else { return panel.title }
        let model = state.ollamaConfiguredModel.isEmpty ? "Automatic" : state.ollamaConfiguredModel
        return "Ollama Model: \(model)"
    }

    private func settingsPanelSymbol(_ panel: BeaconSettingsPanel) -> String {
        if panel == .terminal, case .failed = terminal.shortcutStatus {
            return "exclamationmark.terminal"
        }
        return panel.symbol
    }

    @ViewBuilder
    private func settingsPanelBody(_ panel: BeaconSettingsPanel) -> some View {
        switch panel {
        case .appearance:
            appearanceSettingsPanel
        case .terminal:
            terminalSettingsPanel
        case .ollama:
            ollamaSettingsPanel
        case .agentHooks:
            agentHookSettingsPanel
        }
    }

    private var terminalSettingsPanel: some View {
        VStack(alignment: .leading, spacing: 14) {
            Button {
                terminal.toggle()
            } label: {
                Label(terminal.isVisible ? "Hide Terminal" : "Show Terminal", systemImage: "terminal")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)

            Picker("Position", selection: $terminal.edge) {
                ForEach(TerminalEdge.allCases) { edge in
                    Label(edge.title, systemImage: edge.symbol).tag(edge)
                }
            }
            .pickerStyle(.segmented)

            Picker("Height", selection: $terminal.height) {
                ForEach(TerminalHeight.allCases) { height in
                    Text("\(height.title) · \(Int(height.fraction * 100))%").tag(height)
                }
            }
            .pickerStyle(.segmented)

            Divider()
            terminalShortcutStatus
            Divider()

            if WarpTerminalIntegration.isInstalled {
                Text("Warp cannot be embedded; use its own terminal window while Warp is active.")
                    .foregroundStyle(theme.tokens.textMuted.color)
                HStack {
                    Button("Open Warp", systemImage: "terminal.fill") {
                        WarpTerminalIntegration.openApplication()
                    }
                    Button("Warp Hotkey Setup Guide", systemImage: "questionmark.circle") {
                        WarpTerminalIntegration.openGuide()
                    }
                }
            } else {
                Text("Warp is not installed; Beacon uses its built-in terminal.")
                    .foregroundStyle(theme.tokens.textMuted.color)
                Button("Warp Hotkey Setup Guide", systemImage: "questionmark.circle") {
                    WarpTerminalIntegration.openGuide()
                }
            }
        }
    }

    @ViewBuilder
    private var terminalShortcutStatus: some View {
        switch terminal.shortcutStatus {
        case .inactive:
            Label("Shortcut inactive", systemImage: "minus.circle")
        case .registered:
            Label("App Shortcut: ⌘J", systemImage: "checkmark.circle")
            Text("Available only while Beacon is active.")
                .foregroundStyle(theme.tokens.textMuted.color)
        case .failed(let message):
            Label("Shortcut unavailable", systemImage: "exclamationmark.triangle.fill")
                .help(message)
            Text(message)
                .foregroundStyle(theme.tokens.textMuted.color)
        }
    }

    private var ollamaSettingsPanel: some View {
        VStack(alignment: .leading, spacing: 10) {
            if state.isLoadingOllamaModels {
                ProgressView("Loading local models…")
            } else if state.ollamaModels.isEmpty {
                ContentUnavailableView(
                    state.ollamaError ?? "No local Ollama models found",
                    systemImage: "brain.head.profile"
                )
            } else {
                ForEach(state.ollamaModels) { model in
                    Button {
                        Task { await state.setOllamaDefaultModel(model.name) }
                    } label: {
                        HStack {
                            Text(model.name)
                            Spacer()
                            if model.name == state.ollamaConfiguredModel {
                                Image(systemName: "checkmark.circle.fill")
                                    .foregroundStyle(theme.tokens.success.color)
                            }
                        }
                        .padding(8)
                        .frame(maxWidth: .infinity)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .background(theme.tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 7))
                    .disabled(state.isSavingOllamaDefault)
                }
            }
            if let notice = state.ollamaNotice {
                Text(notice)
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(theme.tokens.textMuted.color)
            }
            Divider()
            Button {
                Task { await state.refreshOllamaModels() }
            } label: {
                Label("Refresh Local Models", systemImage: "arrow.clockwise")
            }
            .disabled(state.isLoadingOllamaModels)
        }
    }

    private var agentHookSettingsPanel: some View {
        VStack(alignment: .leading, spacing: 10) {
            integrationHealthRow(provider: "codex", title: "Codex")
            integrationHealthRow(provider: "claude-code", title: "Claude Code")
            Divider()
            Text("Installed means configured; active means Beacon observed the current callback.")
            Text("Codex may require hook trust. Claude Code may be blocked by managed policy.")
                .foregroundStyle(theme.tokens.textMuted.color)
            Button {
                Task { await state.refreshIntegrationHealth() }
            } label: {
                Label("Refresh Integration Health", systemImage: "arrow.clockwise")
            }
        }
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
        case "active": "checkmark.circle.fill"
        case "installed": "checkmark.circle"
        case "stale": "exclamationmark.circle"
        case "error": "exclamationmark.triangle.fill"
        default: "minus.circle"
        }
    }
}
