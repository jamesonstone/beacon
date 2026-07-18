import AppKit
import SwiftUI

enum BeaconSwitcherScope {
    case all
    case notes

    var title: String { self == .all ? "Quick Switcher" : "Tab Search" }
    var prompt: String { self == .all ? "Search commands, tabs, projects, and lanes" : "Search Notes" }
}
struct BeaconCommandItem: Identifiable {
    let id: String
    let title: String
    let detail: String
    let symbol: String
    let keywords: String
    let deletableNote: AgentNoteTab?
    let action: @MainActor () -> Void

    init(
        id: String,
        title: String,
        detail: String,
        symbol: String,
        keywords: String,
        deletableNote: AgentNoteTab? = nil,
        action: @escaping @MainActor () -> Void
    ) {
        self.id = id
        self.title = title
        self.detail = detail
        self.symbol = symbol
        self.keywords = keywords
        self.deletableNote = deletableNote
        self.action = action
    }

    func matches(_ query: String) -> Bool {
        let needle = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !needle.isEmpty else { return true }
        return "\(title) \(detail) \(keywords)".lowercased().contains(needle)
    }
}

struct BeaconQuickSwitcher: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    let scope: BeaconSwitcherScope
    let commands: [BeaconCommandItem]
    @Binding var query: String
    @Binding var selection: Int
    let onDeleteNote: (AgentNoteTab) -> Void
    let dismiss: () -> Void
    @FocusState private var searchFocused: Bool

    private var results: [BeaconCommandItem] {
        commands.filter { $0.matches(query) }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: scope == .all ? "command" : "doc.text.magnifyingglass")
                    .foregroundStyle(BeaconThemePreference.current().tokens.info.color)
                Text(scope.title)
                    .font(BeaconTypography.semibold(11))
                    .foregroundStyle(BeaconThemePreference.current().tokens.success.color)
                Spacer()
                Text(scope == .all ? "⌘K" : "⌘P")
                    .font(BeaconTypography.regular(8))
                    .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
            }

            TextField(scope.prompt, text: $query)
                .textFieldStyle(.roundedBorder)
                .focused($searchFocused)
                .onSubmit(performSelection)
                .onChange(of: query) { _, _ in selection = 0 }

            if results.isEmpty {
                ContentUnavailableView("No matches", systemImage: "magnifyingglass")
                    .frame(maxWidth: .infinity, minHeight: 120)
            } else {
                ScrollViewReader { proxy in
                    ScrollView {
                        LazyVStack(spacing: 3) {
                            ForEach(Array(results.enumerated()), id: \.element.id) { index, command in
                                HStack(spacing: 4) {
                                    Button {
                                        selection = index
                                        performSelection()
                                    } label: {
                                        HStack(spacing: 8) {
                                            Image(systemName: command.symbol)
                                                .frame(width: 15)
                                                .foregroundStyle(index == selection ? BeaconThemePreference.current().tokens.success.color : BeaconThemePreference.current().tokens.info.color)
                                            VStack(alignment: .leading, spacing: 1) {
                                                Text(command.title)
                                                    .font(BeaconTypography.medium(9))
                                                    .foregroundStyle(BeaconThemePreference.current().tokens.success.color)
                                                    .lineLimit(1)
                                                if !command.detail.isEmpty {
                                                    Text(command.detail)
                                                        .font(BeaconTypography.regular(7))
                                                        .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
                                                        .lineLimit(1)
                                                }
                                            }
                                            Spacer()
                                            if index == selection {
                                                Image(systemName: "return")
                                                    .font(.system(size: 8, weight: .semibold))
                                                    .foregroundStyle(BeaconThemePreference.current().tokens.textSecondary.color)
                                            }
                                        }
                                        .padding(.leading, 8)
                                        .frame(height: 34)
                                        .contentShape(Rectangle())
                                    }
                                    .buttonStyle(.plain)
                                    .frame(maxWidth: .infinity)

                                    if let tab = command.deletableNote {
                                        Button {
                                            dismiss()
                                            onDeleteNote(tab)
                                        } label: {
                                            Image(systemName: "trash")
                                                .font(.system(size: 9, weight: .semibold))
                                                .foregroundStyle(BeaconThemePreference.current().tokens.danger.color)
                                                .frame(width: 25, height: 25)
                                                .contentShape(Rectangle())
                                        }
                                        .buttonStyle(.plain)
                                        .help("Delete \(tab.title)")
                                        .accessibilityLabel("Delete \(tab.title) note")
                                    }
                                }
                                .padding(.trailing, 5)
                                .background(
                                    index == selection
                                        ? BeaconThemePreference.current().tokens.surfaceRaised.color
                                        : BeaconThemePreference.current().tokens.surface.color,
                                    in: RoundedRectangle(cornerRadius: 7)
                                )
                                .id(command.id)
                            }
                        }
                    }
                    .onChange(of: selection) { _, latest in
                        guard results.indices.contains(latest) else { return }
                        withAnimation(reduceMotion ? nil : .easeOut(duration: 0.08)) {
                            proxy.scrollTo(results[latest].id)
                        }
                    }
                }
            }

            Text("↑↓ navigate · Return open · Esc dismiss")
                .font(BeaconTypography.regular(7))
                .foregroundStyle(BeaconThemePreference.current().tokens.textMuted.color)
        }
        .padding(12)
        .frame(maxWidth: 390, maxHeight: 390)
        .background(BeaconThemePreference.current().tokens.surfaceOverlay.color, in: RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12)
                .strokeBorder(BeaconThemePreference.current().tokens.borderStrong.color, lineWidth: 1)
        }
        .shadow(color: Color.black.opacity(0.18), radius: 8, y: 4)
        .task {
            await Task.yield()
            searchFocused = true
        }
        .onKeyPress(.upArrow) { moveSelection(-1); return .handled }
        .onKeyPress(.downArrow) { moveSelection(1); return .handled }
        .onKeyPress(.escape) { dismiss(); return .handled }
    }

    private func moveSelection(_ delta: Int) {
        guard !results.isEmpty else { return }
        selection = (selection + delta + results.count) % results.count
    }

    @MainActor
    private func performSelection() {
        guard results.indices.contains(selection) else { return }
        let command = results[selection]
        dismiss()
        command.action()
    }
}
