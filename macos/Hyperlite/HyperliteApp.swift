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
            Image(systemName: state.attentionCount > 0 ? "exclamationmark.circle.fill" : "circle.fill")
                .accessibilityLabel("Hyperlite, (state.attentionCount) items need attention")
        }
        .menuBarExtraStyle(.window)
    }
}

@MainActor
final class HyperliteState: ObservableObject {
    @Published private(set) var snapshot: BeaconSnapshot?
    @Published private(set) var isRefreshing = false
    @Published private(set) var errorMessage: String?

    private let client = AgentClient()
    private var subscriptionTask: Task<Void, Never>?
    private var helperProcess: Process?

    var items: [HyperliteItem] {
        guard let snapshot else { return [] }
        return HyperlitePresentation.items(snapshot: snapshot, activity: .empty)
    }

    var attentionCount: Int { items.filter(\.attention).count }

    init() {
        start()
    }

    deinit {
        subscriptionTask?.cancel()
        helperProcess?.terminate()
    }

    func refresh() {
        guard !isRefreshing else { return }
        isRefreshing = true
        Task {
            do {
                try await ensureAgent()
                _ = try await client.refresh(project: nil)
                try await loadSnapshot()
                errorMessage = nil
            } catch {
                errorMessage = error.localizedDescription
            }
            isRefreshing = false
        }
    }

    private func start() {
        subscriptionTask = Task {
            do {
                try await ensureAgent()
                try await loadSnapshot()
                let stream = try await client.subscribe()
                for try await event in stream {
                    if let snapshot = event.snapshot {
                        self.snapshot = snapshot
                        self.errorMessage = nil
                    }
                }
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    private func loadSnapshot() async throws {
        let event = try await client.snapshot()
        guard let snapshot = event.snapshot else {
            throw AgentClientError.invalidResponse("snapshot was missing")
        }
        self.snapshot = snapshot
    }

    private func ensureAgent() async throws {
        do {
            _ = try await client.status()
            return
        } catch {
            try startBundledAgent()
            for _ in 0..<20 {
                try await Task.sleep(for: .milliseconds(150))
                if (try? await client.status()) != nil { return }
            }
            throw error
        }
    }

    private func startBundledAgent() throws {
        guard helperProcess == nil else { return }
        let executable = Bundle.main.bundleURL
            .appendingPathComponent("Contents/MacOS/beacon-cli")
        guard FileManager.default.isExecutableFile(atPath: executable.path) else {
            throw AgentClientError.connection("bundled agent helper is missing")
        }
        let process = Process()
        process.executableURL = executable
        process.arguments = ["agent", "start"]
        process.standardOutput = FileHandle.nullDevice
        process.standardError = FileHandle.nullDevice
        try process.run()
        helperProcess = process
    }
}

struct HyperlitePopover: View {
    @ObservedObject var state: HyperliteState

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
                Text("\(state.attentionCount)")
                    .font(.system(.title3, design: .rounded).weight(.bold))
                    .foregroundStyle(state.attentionCount == 0 ? .green : .orange)
            }

            if let errorMessage = state.errorMessage {
                Label(errorMessage, systemImage: "exclamationmark.triangle")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else if state.snapshot == nil {
                ProgressView("Connecting…")
                    .controlSize(.small)
            } else if state.items.isEmpty {
                Label("No active work", systemImage: "checkmark.circle")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(state.items) { item in
                    HyperliteRow(item: item)
                }
            }

            Divider()
            HStack {
                Text(state.snapshot.map { "Updated \(HyperlitePresentation.ageLabel(for: HyperlitePresentationDate.parse($0.generatedAt)))" } ?? "Waiting for evidence")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                Spacer()
                Button { state.refresh() } label: {
                    Image(systemName: state.isRefreshing ? "arrow.clockwise" : "arrow.clockwise")
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
    let item: HyperliteItem

    var body: some View {
        HStack(alignment: .top, spacing: 8) {
            Circle()
                .fill(item.attention ? .orange : .blue)
                .frame(width: 7, height: 7)
                .padding(.top, 4)
            VStack(alignment: .leading, spacing: 2) {
                Text(item.lane.repository).font(.system(size: 12, weight: .semibold))
                Text(item.lane.nextAction)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }
            Spacer(minLength: 4)
            Text(HyperlitePresentation.ageLabel(for: item.ageDate))
                .font(.caption2.monospacedDigit())
                .foregroundStyle(.secondary)
        }
    }
}
