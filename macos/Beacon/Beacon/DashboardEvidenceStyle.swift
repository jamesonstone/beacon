import SwiftUI

enum EvidenceBadgeDismissals {
    private static let separator = "\u{1F}"

    static func key(laneID: String, dimension: String, value: String) -> String {
        [laneID, dimension.lowercased(), value.lowercased()].joined(separator: separator)
    }

    static func decode(_ value: String) -> Set<String> {
        guard let data = value.data(using: .utf8),
              let keys = try? JSONDecoder().decode([String].self, from: data)
        else { return [] }
        return Set(keys)
    }

    static func encode(_ keys: Set<String>) -> String {
        guard let data = try? JSONEncoder().encode(keys.sorted()),
              let value = String(data: data, encoding: .utf8)
        else { return "[]" }
        return value
    }
}

enum EvidenceTaxonomy {
    static func pullRequestFeedbackLabel(_ count: Int) -> String {
        "PR feedback · \(count)"
    }
}

enum RichHoverPresentation {
    static let openDelay: Duration = .seconds(3)
    static let closeDelay: Duration = .milliseconds(250)

    static func cardDetailEnabled(evidenceHoverLaneID: String?, laneID: String) -> Bool {
        evidenceHoverLaneID != laneID
    }
}

struct DismissibleEvidenceBadge: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    let text: String
    let symbol: String
    let accent: Color
    let emphasized: Bool
    let onDismiss: () -> Void
    @State private var isHovered = false

    var body: some View {
        HStack(spacing: 3) {
            Label(text, systemImage: symbol)
                .font(BeaconTypography.medium(9))
            Button(action: onDismiss) {
                Image(systemName: "xmark")
                    .font(.system(size: 7, weight: .bold))
                    .frame(width: 9, height: 9)
            }
            .buttonStyle(.plain)
            .opacity(isHovered ? 1 : 0)
            .allowsHitTesting(isHovered)
            .accessibilityLabel("Hide \(text) badge")
        }
        .foregroundStyle(accent)
        .padding(.leading, 6)
        .padding(.trailing, 4)
        .padding(.vertical, 3)
        .background(BeaconThemePreference.current().tokens.surfaceRaised.color, in: Capsule())
        .overlay {
            Capsule()
                .strokeBorder(accent.opacity(emphasized ? 0.8 : 0.34), lineWidth: 0.6)
        }
        .shadow(color: emphasized ? accent.opacity(0.28) : .clear, radius: 2)
        .onHover { isHovered = $0 }
        .animation(reduceMotion ? nil : .easeOut(duration: 0.12), value: isHovered)
    }
}

struct RichHoverPopover<PopoverContent: View>: ViewModifier {
    let enabled: Bool
    let content: () -> PopoverContent
    @State private var isPresented = false
    @State private var isPinned = false
    @State private var triggerHovered = false
    @State private var popoverHovered = false
    @State private var pendingTask: Task<Void, Never>?
    @FocusState private var isFocused: Bool

    func body(content trigger: Content) -> some View {
        trigger
            .focusable()
            .focused($isFocused)
            .onChange(of: isFocused) { _, focused in
                focused ? presentImmediately() : scheduleClose()
            }
            .onChange(of: enabled) { _, isEnabled in
                pendingTask?.cancel()
                if isEnabled, isFocused {
                    presentImmediately()
                } else if isEnabled, triggerHovered {
                    scheduleOpen()
                } else if !isEnabled, !isPinned {
                    isPresented = false
                }
            }
            .onHover { hovered in
                triggerHovered = hovered
                hovered ? scheduleOpen() : scheduleClose()
            }
            .popover(isPresented: $isPresented, arrowEdge: .trailing) {
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        Text("Details")
                            .font(BeaconTypography.semibold(12))
                        Spacer()
                        Button {
                            isPinned.toggle()
                        } label: {
                            Label(isPinned ? "Unpin" : "Pin", systemImage: isPinned ? "pin.fill" : "pin")
                                .labelStyle(.iconOnly)
                        }
                        .buttonStyle(.plain)
                        .help(isPinned ? "Allow this detail panel to close" : "Keep this detail panel open")
                        Button {
                            isPinned = false
                            isPresented = false
                        } label: {
                            Image(systemName: "xmark")
                        }
                        .buttonStyle(.plain)
                        .help("Close details")
                    }
                    Divider()
                    ScrollView { content() }
                }
                .padding(12)
                .frame(width: 520, height: 420)
                .background(BeaconThemePreference.current().tokens.surfaceOverlay.color)
                .onHover { hovered in
                    popoverHovered = hovered
                    if !hovered { scheduleClose() }
                }
                .onExitCommand {
                    isPinned = false
                    isPresented = false
                }
            }
            .accessibilityAction(named: "Show details") {
                isPresented = true
                isPinned = true
            }
    }

    private func presentImmediately() {
        pendingTask?.cancel()
        guard enabled else { return }
        isPresented = true
    }

    private func scheduleOpen() {
        pendingTask?.cancel()
        guard enabled else { return }
        pendingTask = Task { @MainActor in
            try? await Task.sleep(for: RichHoverPresentation.openDelay)
            guard !Task.isCancelled, enabled, triggerHovered else { return }
            isPresented = true
        }
    }

    private func scheduleClose() {
        pendingTask?.cancel()
        guard !isPinned else { return }
        pendingTask = Task { @MainActor in
            try? await Task.sleep(for: RichHoverPresentation.closeDelay)
            guard !Task.isCancelled, !triggerHovered, !popoverHovered, !isFocused else { return }
            isPresented = false
        }
    }
}

extension View {
    func richHoverPopover<Content: View>(
        enabled: Bool = true,
        @ViewBuilder content: @escaping () -> Content
    ) -> some View {
        modifier(RichHoverPopover(enabled: enabled, content: content))
    }
}
