import AppKit
import SwiftUI

struct BeaconSRGB: Equatable, Hashable {
    let red: Double
    let green: Double
    let blue: Double
    let hex: String

    init(hex: String) {
        let normalized = hex.uppercased()
        let digits = normalized.hasPrefix("#") ? String(normalized.dropFirst()) : normalized
        precondition(digits.count == 6, "Theme colors must use six-digit sRGB hex values")
        guard let value = UInt64(digits, radix: 16) else {
            preconditionFailure("Invalid theme color: \(hex)")
        }
        red = Double((value >> 16) & 0xFF) / 255
        green = Double((value >> 8) & 0xFF) / 255
        blue = Double(value & 0xFF) / 255
        self.hex = "#\(digits)"
    }

    var color: Color {
        Color(.sRGB, red: red, green: green, blue: blue, opacity: 1)
    }

    var nsColor: NSColor {
        NSColor(srgbRed: red, green: green, blue: blue, alpha: 1)
    }

    var relativeLuminance: Double {
        func linear(_ component: Double) -> Double {
            component <= 0.04045
                ? component / 12.92
                : pow((component + 0.055) / 1.055, 2.4)
        }
        return 0.2126 * linear(red) + 0.7152 * linear(green) + 0.0722 * linear(blue)
    }

    func contrastRatio(with other: BeaconSRGB) -> Double {
        let lighter = max(relativeLuminance, other.relativeLuminance)
        let darker = min(relativeLuminance, other.relativeLuminance)
        return (lighter + 0.05) / (darker + 0.05)
    }
}

struct BeaconProjectWatermarkPalette: Equatable {
    let base: BeaconSRGB
    let highlightLeading: BeaconSRGB
    let highlightCenter: BeaconSRGB
    let highlightTrailing: BeaconSRGB

    var namedValues: [(name: String, value: BeaconSRGB)] {
        [
            ("base", base),
            ("highlightLeading", highlightLeading),
            ("highlightCenter", highlightCenter),
            ("highlightTrailing", highlightTrailing),
        ]
    }
}

struct BeaconThemeTokens: Equatable {
    let canvas: BeaconSRGB
    let surface: BeaconSRGB
    let surfaceRaised: BeaconSRGB
    let surfaceOverlay: BeaconSRGB
    let border: BeaconSRGB
    let borderStrong: BeaconSRGB
    let textPrimary: BeaconSRGB
    let textSecondary: BeaconSRGB
    let textMuted: BeaconSRGB
    let accent: BeaconSRGB
    let focus: BeaconSRGB
    let success: BeaconSRGB
    let warning: BeaconSRGB
    let danger: BeaconSRGB
    let info: BeaconSRGB
    let identityLocal: BeaconSRGB
    let identityPullRequest: BeaconSRGB
    let identityIssue: BeaconSRGB
    let editorBackground: BeaconSRGB
    let editorText: BeaconSRGB
    let editorHeading: BeaconSRGB
    let editorLink: BeaconSRGB
    let editorCode: BeaconSRGB
    let editorCodeBackground: BeaconSRGB
    let editorQuote: BeaconSRGB
    let editorSyntax: BeaconSRGB
    let editorSelection: BeaconSRGB

    var namedValues: [(name: String, value: BeaconSRGB)] {
        [
            ("canvas", canvas), ("surface", surface),
            ("surfaceRaised", surfaceRaised), ("surfaceOverlay", surfaceOverlay),
            ("border", border), ("borderStrong", borderStrong),
            ("textPrimary", textPrimary), ("textSecondary", textSecondary),
            ("textMuted", textMuted), ("accent", accent), ("focus", focus),
            ("success", success), ("warning", warning), ("danger", danger),
            ("info", info), ("identityLocal", identityLocal),
            ("identityPullRequest", identityPullRequest),
            ("identityIssue", identityIssue),
            ("editorBackground", editorBackground), ("editorText", editorText),
            ("editorHeading", editorHeading), ("editorLink", editorLink),
            ("editorCode", editorCode), ("editorCodeBackground", editorCodeBackground),
            ("editorQuote", editorQuote), ("editorSyntax", editorSyntax),
            ("editorSelection", editorSelection),
        ]
    }

    var normalTextPairs: [(name: String, foreground: BeaconSRGB, background: BeaconSRGB)] {
        let surfaces = [
            ("canvas", canvas), ("surface", surface),
            ("raised", surfaceRaised), ("overlay", surfaceOverlay),
        ]
        let interfaceText = [
            ("primary", textPrimary), ("secondary", textSecondary),
            ("muted", textMuted), ("accent", accent),
            ("success", success), ("warning", warning),
            ("danger", danger), ("info", info),
            ("Local identity", identityLocal),
            ("PR identity", identityPullRequest),
            ("Issue identity", identityIssue),
        ]
        return interfaceText.flatMap { role in
            surfaces.map { surface in
                ("\(role.0) on \(surface.0)", role.1, surface.1)
            }
        } + [
            ("editor text", editorText, editorBackground),
            ("editor heading", editorHeading, editorBackground),
            ("editor link", editorLink, editorBackground),
            ("editor code", editorCode, editorCodeBackground),
            ("editor quote", editorQuote, editorBackground),
            ("editor syntax", editorSyntax, editorBackground),
        ]
    }

    var indicatorPairs: [(name: String, foreground: BeaconSRGB, background: BeaconSRGB)] {
        let surfaces = [
            ("canvas", canvas), ("surface", surface),
            ("raised", surfaceRaised), ("overlay", surfaceOverlay),
        ]
        let interfaceIndicators = [("strong border", borderStrong), ("focus", focus)]
        return interfaceIndicators.flatMap { role in
            surfaces.map { surface in
                ("\(role.0) on \(surface.0)", role.1, surface.1)
            }
        } + [
            ("selection", editorSelection, editorBackground),
        ]
    }
}

struct BeaconTerminalPalette: Equatable {
    let background: BeaconSRGB
    let foreground: BeaconSRGB
    let cursor: BeaconSRGB
    let selection: BeaconSRGB
    let ansiColors: [BeaconSRGB]

    init(tokens: BeaconThemeTokens) {
        background = tokens.canvas
        foreground = tokens.textPrimary
        cursor = tokens.focus
        selection = tokens.editorSelection
        ansiColors = [
            tokens.canvas,
            tokens.danger,
            tokens.success,
            tokens.warning,
            tokens.info,
            tokens.identityIssue,
            tokens.accent,
            tokens.textSecondary,
            tokens.textMuted,
            tokens.danger,
            tokens.success,
            tokens.warning,
            tokens.focus,
            tokens.identityIssue,
            tokens.accent,
            tokens.textPrimary,
        ]
    }

    var readableTextPairs: [(name: String, foreground: BeaconSRGB, background: BeaconSRGB)] {
        let names = [
            "red", "green", "yellow", "blue", "magenta", "cyan", "white",
            "bright black", "bright red", "bright green", "bright yellow",
            "bright blue", "bright magenta", "bright cyan", "bright white",
        ]
        return zip(names, ansiColors.dropFirst()).map { name, color in
            ("ANSI \(name)", color, background)
        } + [
            ("default foreground", foreground, background),
            ("cursor", cursor, background),
        ]
    }
}

enum BeaconThemeID: String, CaseIterable, Identifiable {
    case lobsterNebula = "lobster-nebula"
    case pampasMoon = "pampas-moon"
    case solarizedDark = "solarized-dark"
    case monokai
    case selenizedDark = "selenized-dark"

    var id: String { rawValue }
}

enum BeaconThemeAppearance: String {
    case dark
    case light

    var colorScheme: ColorScheme {
        self == .dark ? .dark : .light
    }
}

struct BeaconTheme: Identifiable, Equatable {
    let id: BeaconThemeID
    let name: String
    let detail: String
    let appearance: BeaconThemeAppearance
    let signatureAccent: BeaconSRGB
    let isRecommended: Bool
    let projectWatermark: BeaconProjectWatermarkPalette
    let tokens: BeaconThemeTokens

    var terminalPalette: BeaconTerminalPalette {
        BeaconTerminalPalette(tokens: tokens)
    }

    var accessibilityName: String {
        if isRecommended { return "\(name), recommended" }
        if appearance == .light { return "\(name), high-readability light theme" }
        return name
    }

    var brandGradient: LinearGradient {
        LinearGradient(
            colors: [
                signatureAccent.color,
                tokens.identityPullRequest.color,
                tokens.identityIssue.color,
            ],
            startPoint: .leading,
            endPoint: .trailing
        )
    }
}

enum BeaconThemePreference {
    static let storageKey = "beacon.appearance.theme.v1"
    static let defaultID = BeaconThemeID.lobsterNebula

    static func resolvedID(_ storedValue: String?) -> BeaconThemeID {
        guard let storedValue, let value = BeaconThemeID(rawValue: storedValue) else {
            return defaultID
        }
        return value
    }

    static func current(in defaults: UserDefaults = .standard) -> BeaconTheme {
        BeaconThemeCatalog.theme(for: resolvedID(defaults.string(forKey: storageKey)))
    }

    static func persist(_ id: BeaconThemeID, in defaults: UserDefaults = .standard) {
        defaults.set(id.rawValue, forKey: storageKey)
    }
}

private struct BeaconThemeEnvironmentKey: EnvironmentKey {
    static let defaultValue = BeaconThemeCatalog.lobsterNebula
}

extension EnvironmentValues {
    var beaconTheme: BeaconTheme {
        get { self[BeaconThemeEnvironmentKey.self] }
        set { self[BeaconThemeEnvironmentKey.self] = newValue }
    }
}

struct BeaconThemePreview: View {
    let theme: BeaconTheme
    let isSelected: Bool

    var body: some View {
        HStack(spacing: 3) {
            swatch(theme.tokens.canvas.color)
            swatch(theme.tokens.surface.color)
            swatch(theme.signatureAccent.color)
            swatch(theme.tokens.success.color)
            swatch(theme.tokens.identityIssue.color)
        }
        .padding(3)
        .background(theme.tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 5))
        .overlay {
            RoundedRectangle(cornerRadius: 5)
                .strokeBorder(
                    (isSelected ? theme.tokens.focus : theme.tokens.borderStrong).color,
                    lineWidth: isSelected ? 2 : 1
                )
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("\(theme.accessibilityName) color preview")
    }

    private func swatch(_ color: Color) -> some View {
        RoundedRectangle(cornerRadius: 2)
            .fill(color)
            .frame(width: 10, height: 14)
    }
}
