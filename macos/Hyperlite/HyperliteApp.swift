import Foundation
import SwiftUI

@main
struct HyperliteApp: App {
    @StateObject private var state = HyperliteState()

    var body: some Scene {
        MenuBarExtra {
            HyperlitePopover(state: state)
        } label: {
            Image(systemName: "circle.fill")
                .accessibilityLabel("Hyperlite popover")
        }
        .menuBarExtraStyle(.window)
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
        case .scanFailed(let message): "bctl scan failed: (message)"
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
            }
        }
        .padding(12)
        .frame(width: 330)
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
