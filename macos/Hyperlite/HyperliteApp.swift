import AppKit
import Carbon.HIToolbox
import Foundation
import SwiftUI

@main
struct HyperliteApp: App {
    @NSApplicationDelegateAdaptor(HyperliteApplicationDelegate.self) private var applicationDelegate

    var body: some Scene {
        Settings {
            HyperliteSettingsView()
        }
    }
}

final class HyperliteApplicationDelegate: NSObject, NSApplicationDelegate {
    private var state: HyperliteState!
    private var statusItem: NSStatusItem!
    private var popover: NSPopover!
    private var hotKey: HyperliteHotKeyController!

    func applicationDidFinishLaunching(_ notification: Notification) {
        state = HyperliteState()
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        if let button = statusItem.button {
            button.image = NSImage(systemSymbolName: "rocket.fill", accessibilityDescription: "Hyperlite")
            button.image?.isTemplate = false
            button.title = "✦"
            button.font = .systemFont(ofSize: 10, weight: .bold)
            button.imagePosition = .imageLeading
            button.toolTip = "Hyperlite — Control+Shift+H"
            button.target = self
            button.action = #selector(togglePopover)
        }

        popover = NSPopover()
        popover.behavior = .transient
        popover.animates = true
        popover.contentViewController = NSHostingController(rootView: HyperlitePopover(state: state))

        hotKey = HyperliteHotKeyController { [weak self] in
            self?.togglePopover()
        }
        hotKey.start()
    }

    @objc private func togglePopover() {
        guard let button = statusItem.button else { return }
        if popover.isShown {
            popover.performClose(nil)
        } else {
            NSApp.activate(ignoringOtherApps: true)
            popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        hotKey.stop()
        if let statusItem {
            NSStatusBar.system.removeStatusItem(statusItem)
        }
    }
}

final class HyperliteHotKeyController {
    private let action: () -> Void
    private var eventHandler: EventHandlerRef?
    private var hotKeyRef: EventHotKeyRef?

    init(action: @escaping () -> Void) {
        self.action = action
    }

    func start() {
        guard eventHandler == nil else { return }
        var eventSpec = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))
        let context = UnsafeMutableRawPointer(Unmanaged.passUnretained(self).toOpaque())
        InstallEventHandler(
            GetApplicationEventTarget(),
            { _, _, userData in
                guard let userData else { return noErr }
                Unmanaged<HyperliteHotKeyController>.fromOpaque(userData).takeUnretainedValue().action()
                return noErr
            },
            1,
            &eventSpec,
            context,
            &eventHandler
        )

        var hotKeyID = EventHotKeyID(signature: fourCharCode("HLIT"), id: 1)
        RegisterEventHotKey(
            UInt32(kVK_ANSI_H),
            UInt32(controlKey | shiftKey),
            hotKeyID,
            GetApplicationEventTarget(),
            0,
            &hotKeyRef
        )
    }

    func stop() {
        if let hotKeyRef {
            UnregisterEventHotKey(hotKeyRef)
        }
        if let eventHandler {
            RemoveEventHandler(eventHandler)
        }
        hotKeyRef = nil
        eventHandler = nil
    }

    deinit {
        stop()
    }
}

private func fourCharCode(_ value: String) -> OSType {
    value.utf8.reduce(0) { ($0 << 8) | OSType($1) }
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
