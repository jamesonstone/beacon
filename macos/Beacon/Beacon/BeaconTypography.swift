import AppKit
import SwiftUI

enum BeaconFontCatalog {
    static let defaultFamily = "JetBrainsMono Nerd Font"

    static var installedFamilies: [String] {
        normalized(NSFontManager.shared.availableFontFamilies)
    }

    static var selectionOptions: [String] {
        selectionOptions(installedFamilies: installedFamilies)
    }

    static func selectionOptions(installedFamilies: [String]) -> [String] {
        var families = normalized(installedFamilies)
        if let defaultIndex = families.firstIndex(where: { matchesDefault($0) }) {
            let installedDefault = families.remove(at: defaultIndex)
            return [installedDefault] + families
        }
        return [defaultFamily] + families
    }

    static func resolvedFamily(
        storedFamily: String?,
        installedFamilies: [String]
    ) -> String? {
        let requested = requestedFamily(from: storedFamily)
        return match(requested, in: installedFamilies)
            ?? match(defaultFamily, in: installedFamilies)
    }

    static func resolvedFamily(storedFamily: String?) -> String? {
        resolvedFamily(storedFamily: storedFamily, installedFamilies: installedFamilies)
    }

    static func displayName(for storedFamily: String) -> String {
        if let selected = match(storedFamily, in: installedFamilies) {
            return selected
        }
        if let fallback = match(defaultFamily, in: installedFamilies) {
            return "\(fallback) (Fallback)"
        }
        return "System Font (Fallback)"
    }

    static func requestedFamily(from storedFamily: String?) -> String {
        let trimmed = storedFamily?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return trimmed.isEmpty ? defaultFamily : trimmed
    }

    private static func match(_ requested: String, in installedFamilies: [String]) -> String? {
        installedFamilies.first {
            $0.compare(
                requested,
                options: [.caseInsensitive, .diacriticInsensitive]
            ) == .orderedSame
        }
    }

    private static func matchesDefault(_ family: String) -> Bool {
        family.compare(
            defaultFamily,
            options: [.caseInsensitive, .diacriticInsensitive]
        ) == .orderedSame
    }

    private static func normalized(_ families: [String]) -> [String] {
        var seen: Set<String> = []
        return families.compactMap { family in
            let trimmed = family.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty else { return nil }
            let key = trimmed.folding(
                options: [.caseInsensitive, .diacriticInsensitive],
                locale: .current
            )
            guard seen.insert(key).inserted else { return nil }
            return trimmed
        }
        .sorted { $0.localizedStandardCompare($1) == .orderedAscending }
    }
}

enum BeaconFontSize: Int, CaseIterable, Identifiable {
    case compact = 11
    case standard = 12
    case comfortable = 13
    case large = 14
    case extraLarge = 16

    var id: Int { rawValue }
    var title: String { "\(rawValue) pt" }
}

enum BeaconTypography {
    static let familyKey = "beacon.dashboard.font-family"
    static let baseSizeKey = "beacon.dashboard.font-size"
    static let defaultFamily = BeaconFontCatalog.defaultFamily
    static let defaultBaseSize = BeaconFontSize.standard.rawValue

    static func regular(_ size: CGFloat) -> Font {
        preferred(size: size, weight: .regular)
    }

    static func medium(_ size: CGFloat) -> Font {
        preferred(size: size, weight: .medium)
    }

    static func semibold(_ size: CGFloat) -> Font {
        preferred(size: size, weight: .semibold)
    }

    static func bold(_ size: CGFloat) -> Font {
        preferred(size: size, weight: .bold)
    }

    static func identifier(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        preferred(size: size, weight: weight, monospacedFallback: true)
    }

    static func code(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        preferred(size: size, weight: weight, monospacedFallback: true)
    }

    static func counter(_ size: CGFloat, weight: Font.Weight = .medium) -> Font {
        preferred(size: size, weight: weight, monospacedFallback: true)
    }

    static func appKitFont(_ size: CGFloat, weight: NSFont.Weight = .regular) -> NSFont {
        appKitFont(
            size,
            weight: weight,
            requestedFamily: requestedFamily,
            installedFamilies: BeaconFontCatalog.installedFamilies
        )
    }

    static func appKitCodeFont(_ size: CGFloat, weight: NSFont.Weight = .regular) -> NSFont {
        appKitFont(
            size,
            weight: weight,
            requestedFamily: requestedFamily,
            installedFamilies: BeaconFontCatalog.installedFamilies,
            monospacedFallback: true
        )
    }

    static func appKitFont(
        _ size: CGFloat,
        weight: NSFont.Weight,
        requestedFamily: String,
        installedFamilies: [String],
        monospacedFallback: Bool = false
    ) -> NSFont {
        let pointSize = resolvedSize(size)
        if let family = BeaconFontCatalog.resolvedFamily(
            storedFamily: requestedFamily,
            installedFamilies: installedFamilies
        ) {
            let descriptor = NSFontDescriptor(fontAttributes: [
                .family: family,
                .traits: [NSFontDescriptor.TraitKey.weight: weight],
            ])
            if let font = NSFont(descriptor: descriptor, size: pointSize) {
                return font
            }
        }
        if monospacedFallback {
            return NSFont.monospacedSystemFont(ofSize: pointSize, weight: weight)
        }
        return NSFont.systemFont(ofSize: pointSize, weight: weight)
    }

    static var selectionSignature: String {
        "\(resolvedFamily ?? "system"):\(selectedBaseSize):\(BeaconThemePreference.current().id.rawValue)"
    }

    static func requestedFamily(in defaults: UserDefaults) -> String {
        BeaconFontCatalog.requestedFamily(from: defaults.string(forKey: familyKey))
    }

    static func resolvedFamily(
        in defaults: UserDefaults,
        installedFamilies: [String]
    ) -> String? {
        BeaconFontCatalog.resolvedFamily(
            storedFamily: requestedFamily(in: defaults),
            installedFamilies: installedFamilies
        )
    }

    static func resolvedSize(_ size: CGFloat) -> CGFloat {
        resolvedSize(size, baseSize: selectedBaseSize)
    }

    static func resolvedSize(_ size: CGFloat, baseSize: Int) -> CGFloat {
        max(11, size + CGFloat(baseSize - 10))
    }

    private static var selectedBaseSize: Int {
        let value = UserDefaults.standard.integer(forKey: baseSizeKey)
        return BeaconFontSize(rawValue: value)?.rawValue ?? defaultBaseSize
    }

    private static var requestedFamily: String {
        requestedFamily(in: .standard)
    }

    private static var resolvedFamily: String? {
        BeaconFontCatalog.resolvedFamily(storedFamily: requestedFamily)
    }

    private static func preferred(
        size: CGFloat,
        weight: Font.Weight,
        monospacedFallback: Bool = false
    ) -> Font {
        let pointSize = resolvedSize(size)
        if let resolvedFamily {
            return .custom(resolvedFamily, size: pointSize).weight(weight)
        }
        return .system(
            size: pointSize,
            weight: weight,
            design: monospacedFallback ? .monospaced : .default
        )
    }
}
