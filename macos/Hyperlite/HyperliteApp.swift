import AppKit
import Foundation
import SwiftUI

@main
struct HyperliteApp: App {
    @StateObject private var state = HyperliteState()

    var body: some Scene {
        MenuBarExtra {
            HyperlitePopover(state: state)
        } label: {
            HyperliteMenuBarLabel(attentionCount: state.attentionCount(maxAgeDays: 10))
        }
        .menuBarExtraStyle(.window)

        Settings {
            HyperliteSettingsView()
        }
    }
}

private struct HyperliteMenuBarLabel: View {
    let attentionCount: Int

    var body: some View {
        HStack(spacing: 2) {
            Image(systemName: attentionCount == 0 ? "sparkles" : "wand.and.stars")
                .symbolRenderingMode(.hierarchical)
                .foregroundStyle(attentionCount == 0 ? .mint : .orange)
            if attentionCount > 0 {
                Text(attentionCount > 99 ? "99+" : "\(attentionCount)")
                    .font(.system(size: 9, weight: .bold, design: .rounded))
                    .foregroundStyle(.orange)
                    .monospacedDigit()
            }
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(
            attentionCount == 0
                ? "Hyperlite, nothing needs attention"
                : "Hyperlite, \(attentionCount) items need attention"
        )
    }
}

@MainActor
final class HyperliteState: ObservableObject {
    @Published private(set) var scan: HyperliteWorkScan?
    @Published private(set) var isRefreshing = false
    @Published private(set) var errorMessage: String?

    private var refreshTask: Task<Void, Never>?

    func items(maxAgeDays: Int, now: Date = Date()) -> [HyperliteWorkItem] {
        guard let scan else { return [] }
        return HyperlitePresentation.items(scan: scan, maxAgeDays: maxAgeDays, now: now)
    }

    func attentionCount(maxAgeDays: Int, now: Date = Date()) -> Int {
        items(maxAgeDays: maxAgeDays, now: now).filter(\.needsAttention).count
    }

    init() {
        refresh(includeNetwork: false)
    }

    deinit {
        refreshTask?.cancel()
    }

    func refresh() {
        refresh(includeNetwork: true)
    }

    private func refresh(includeNetwork: Bool) {
        guard !isRefreshing else { return }
        isRefreshing = true
        refreshTask?.cancel()
        refreshTask = Task { [weak self] in
            guard let self else { return }
            do {
                let arguments = includeNetwork ? ["--json"] : ["--json", "--no-refresh"]
                let data = try await Self.runBctl(arguments: arguments)
                let decoder = JSONDecoder()
                decoder.dateDecodingStrategy = .iso8601
                scan = try decoder.decode(HyperliteWorkScan.self, from: data)
                errorMessage = nil
            } catch is CancellationError {
                return
            } catch {
                errorMessage = error.localizedDescription
            }
            isRefreshing = false
        }
    }

    private static func runBctl(arguments: [String]) async throws -> Data {
        let executable = Bundle.main.bundleURL.appendingPathComponent("Contents/MacOS/bctl")
        guard FileManager.default.isExecutableFile(atPath: executable.path) else {
            throw HyperliteError.helperMissing
        }
        return try await withCheckedThrowingContinuation { continuation in
            let process = Process()
            let output = Pipe()
            let errors = Pipe()
            process.executableURL = executable
            process.arguments = arguments
            process.standardOutput = output
            process.standardError = errors
            process.terminationHandler = { process in
                let data = output.fileHandleForReading.readDataToEndOfFile()
                guard process.terminationStatus == 0 else {
                    let message = String(data: errors.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8)?
                        .trimmingCharacters(in: .whitespacesAndNewlines)
                    continuation.resume(throwing: HyperliteError.scanFailed(message ?? "bctl exited with status \(process.terminationStatus)"))
                    return
                }
                continuation.resume(returning: data)
            }
            do {
                try process.run()
            } catch {
                continuation.resume(throwing: error)
            }
        }
    }
}

private enum HyperliteError: LocalizedError {
    case helperMissing
    case scanFailed(String)

    var errorDescription: String? {
        switch self {
        case .helperMissing: "Hyperlite's bctl helper is unavailable"
        case .scanFailed(let message): "bctl scan failed: \(message)"
        }
    }
}

struct HyperlitePopover: View {
    @ObservedObject var state: HyperliteState
    @AppStorage("hyperlite.max-age-days") private var maxAgeDays = 10

    private var visibleItems: [HyperliteWorkItem] {
        state.items(maxAgeDays: maxAgeDays)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Hyperlite").font(.headline)
                    Text("What needs your attention?")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Text("\(visibleItems.filter(\.needsAttention).count)")
                    .font(.system(.title3, design: .rounded).weight(.bold))
                    .foregroundStyle(visibleItems.contains(where: \.needsAttention) ? .orange : .green)
            }

            if let errorMessage = state.errorMessage {
                Label(errorMessage, systemImage: "exclamationmark.triangle")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else if state.scan == nil {
                ProgressView("Running bctl…")
                    .controlSize(.small)
            } else if visibleItems.isEmpty {
                Label("No work in progress", systemImage: "checkmark.circle")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(visibleItems) { item in
                    HyperliteRow(item: item)
                }
            }

            Divider()
            HStack {
                Picker("Show last", selection: $maxAgeDays) {
                    ForEach(HyperlitePresentation.supportedAgeWindows, id: \.self) { days in
                        Text("Last \(days) days").tag(days)
                    }
                }
                .pickerStyle(.menu)
                .labelsHidden()
                .fixedSize()
                Spacer()
                Text(state.scan.map { "bctl · \(HyperlitePresentation.ageLabel(for: $0.generatedAt))" } ?? "Waiting for bctl")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                Button { state.refresh() } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .buttonStyle(.borderless)
                .disabled(state.isRefreshing)
                SettingsLink {
                    Image(systemName: "gearshape")
                }
                .buttonStyle(.borderless)
                .help("Settings")
            }

            Button("Quit Hyperlite") {
                NSApplication.shared.terminate(nil)
            }
            .buttonStyle(.borderless)
            .foregroundStyle(.secondary)
        }
        .padding(12)
        .frame(width: 330)
    }
}

struct HyperliteSettingsView: View {
    @AppStorage("hyperlite.hotkey") private var hotkey = "Control+Shift+H"

    var body: some View {
        Form {
            Section("General") {
                TextField("Hot key", text: $hotkey)
                    .textFieldStyle(.roundedBorder)
                Text("Default: Control+Shift+H")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
        .frame(width: 360)
        .padding()
    }
}

private struct HyperliteRow: View {
    let item: HyperliteWorkItem

    var body: some View {
        HStack(alignment: .top, spacing: 8) {
            Circle()
                .fill(item.needsAttention ? .orange : .blue)
                .frame(width: 7, height: 7)
                .padding(.top, 4)
            VStack(alignment: .leading, spacing: 2) {
                Text(item.repository).font(.system(size: 12, weight: .semibold))
                Text(item.title)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                Text(item.nextAction.replacingOccurrences(of: "_", with: " ").capitalized)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            Spacer(minLength: 4)
            Text(HyperlitePresentation.ageLabel(for: item.updatedAt))
                .font(.caption2.monospacedDigit())
                .foregroundStyle(.secondary)
        }
    }
}
