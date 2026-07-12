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
