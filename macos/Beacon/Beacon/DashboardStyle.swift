import AppKit
import SwiftUI

enum DashboardViewMode: String, CaseIterable, Identifiable {
    case stacked
    case tiles
    case kanban

    var id: String { rawValue }

    var title: String {
        switch self {
        case .stacked: "Stacked"
        case .tiles: "Horizontal Tiles"
        case .kanban: "Kanban (Experimental)"
        }
    }

    var symbol: String {
        switch self {
        case .stacked: "rectangle.stack"
        case .tiles: "rectangle.grid.1x2"
        case .kanban: "rectangle.split.3x1"
        }
    }
}

enum BeaconTypography {
    static func regular(_ size: CGFloat) -> Font {
        preferred("JetBrainsMonoNFM-Regular", size: size, weight: .regular)
    }

    static func medium(_ size: CGFloat) -> Font {
        preferred("JetBrainsMonoNFM-Medium", size: size, weight: .medium)
    }

    static func semibold(_ size: CGFloat) -> Font {
        preferred("JetBrainsMonoNFM-SemiBold", size: size, weight: .semibold)
    }

    static func bold(_ size: CGFloat) -> Font {
        preferred("JetBrainsMonoNFM-Bold", size: size, weight: .bold)
    }

    private static func preferred(_ name: String, size: CGFloat, weight: Font.Weight) -> Font {
        guard NSFont(name: name, size: size) != nil else {
            return .system(size: size, weight: weight, design: .monospaced)
        }
        return .custom(name, size: size)
    }
}

enum NeonWave {
    static let cycle: TimeInterval = 6
    static let gradient = LinearGradient(
        colors: [
            BeaconPalette.cyan,
            BeaconPalette.mint,
            BeaconPalette.lavender,
            BeaconPalette.pink,
            BeaconPalette.gold,
            BeaconPalette.cyan,
        ],
        startPoint: .leading,
        endPoint: .trailing
    )

    static func phase(at date: Date) -> Double {
        let elapsed = date.timeIntervalSinceReferenceDate.truncatingRemainder(dividingBy: cycle)
        return (elapsed < 0 ? elapsed + cycle : elapsed) / cycle
    }

    static func rotation(at date: Date) -> Angle {
        .degrees(phase(at: date) * 360)
    }
}

struct NeonWaveWordmark: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    private let text: String

    init(_ text: String) {
        self.text = text
    }

    var body: some View {
        TimelineView(.animation(minimumInterval: 1.0 / 20.0, paused: reduceMotion)) { context in
            Text(text)
                .foregroundStyle(NeonWave.gradient)
                .hueRotation(reduceMotion ? .zero : NeonWave.rotation(at: context.date))
                .shadow(color: BeaconPalette.pink.opacity(0.28), radius: 2)
                .accessibilityLabel(text)
        }
    }
}

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

struct DismissibleEvidenceBadge: View {
    let text: String
    let accent: Color
    let emphasized: Bool
    let onDismiss: () -> Void
    @State private var isHovered = false

    var body: some View {
        HStack(spacing: 3) {
            Text(text)
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
        .background(BeaconPalette.softGradient(accent), in: Capsule())
        .overlay {
            Capsule()
                .strokeBorder(accent.opacity(emphasized ? 0.8 : 0.34), lineWidth: 0.6)
        }
        .shadow(color: emphasized ? accent.opacity(0.28) : .clear, radius: 2)
        .onHover { isHovered = $0 }
        .animation(.easeOut(duration: 0.12), value: isHovered)
    }
}
