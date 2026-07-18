import AppKit
import SwiftUI
import XCTest
@testable import Beacon

final class BeaconThemeTests: XCTestCase {
    func testCatalogHasExactlyFiveCanonicalThemes() {
        XCTAssertEqual(
            BeaconThemeCatalog.all.map(\.id.rawValue),
            ["lobster-nebula", "pampas-moon", "solarized-dark", "monokai", "selenized-dark"]
        )
        XCTAssertEqual(
            BeaconThemeCatalog.all.map(\.name),
            ["Lobster Nebula", "Pampas Moon", "Solarized Dark", "Monokai", "Selenized Dark"]
        )
        XCTAssertEqual(BeaconThemeCatalog.all.filter(\.isRecommended).map(\.id), [.lobsterNebula])
        XCTAssertEqual(BeaconThemeCatalog.all.filter { $0.appearance == .light }.map(\.id), [.pampasMoon])
        XCTAssertEqual(BeaconThemePreference.defaultID, .lobsterNebula)
    }

    func testEveryThemeHasEverySemanticToken() {
        for theme in BeaconThemeCatalog.all {
            let values = theme.tokens.namedValues
            XCTAssertEqual(values.count, 27, theme.name)
            XCTAssertEqual(Set(values.map(\.name)).count, 27, theme.name)
            for entry in values {
                XCTAssertTrue(
                    entry.value.hex.range(of: #"^#[0-9A-F]{6}$"#, options: .regularExpression) != nil,
                    "\(theme.name) has invalid \(entry.name) token"
                )
            }
        }
    }

    func testCorePaletteSignaturesMatchTheCanonicalThemes() {
        let expected: [BeaconThemeID: [String]] = [
            .lobsterNebula: ["#151619", "#1C1E22", "#F3F0E8", "#C7C2B8", "#E9785D"],
            .pampasMoon: ["#F7F4ED", "#FFFFFF", "#292724", "#5F5A53", "#A95135"],
            .solarizedDark: ["#002B36", "#073642", "#EEE8D5", "#93A1A1", "#268BD2"],
            .monokai: ["#272822", "#32332C", "#F8F8F2", "#CFCFC2", "#66D9EF"],
            .selenizedDark: ["#103C48", "#184956", "#CAD8D9", "#ADBCBC", "#58A3FF"],
        ]
        for theme in BeaconThemeCatalog.all {
            XCTAssertEqual(
                [
                    theme.tokens.canvas.hex,
                    theme.tokens.surface.hex,
                    theme.tokens.textPrimary.hex,
                    theme.tokens.textSecondary.hex,
                    theme.signatureAccent.hex,
                ],
                expected[theme.id] ?? [],
                theme.name
            )
        }
    }

    func testStableIDFallbackAndPersistence() throws {
        XCTAssertEqual(BeaconThemePreference.storageKey, "beacon.appearance.theme.v1")
        XCTAssertEqual(BeaconThemePreference.resolvedID(nil), .lobsterNebula)
        XCTAssertEqual(BeaconThemePreference.resolvedID("unknown-theme"), .lobsterNebula)
        XCTAssertEqual(BeaconThemeCatalog.theme(forStoredID: "pampas-moon").id, .pampasMoon)

        let suiteName = "BeaconThemeTests.\(UUID().uuidString)"
        let suite = try XCTUnwrap(UserDefaults(suiteName: suiteName))
        defer { suite.removePersistentDomain(forName: suiteName) }
        XCTAssertEqual(BeaconThemePreference.current(in: suite).id, .lobsterNebula)
        BeaconThemePreference.persist(.monokai, in: suite)
        XCTAssertEqual(suite.string(forKey: BeaconThemePreference.storageKey), "monokai")
        XCTAssertEqual(BeaconThemePreference.current(in: suite).id, .monokai)
    }

    func testNormalTextMeetsWCAGAAAcrossThemes() {
        for theme in BeaconThemeCatalog.all {
            for pair in theme.tokens.normalTextPairs {
                XCTAssertGreaterThanOrEqual(
                    pair.foreground.contrastRatio(with: pair.background),
                    4.5,
                    "\(theme.name) \(pair.name) must meet 4.5:1"
                )
            }
        }
    }

    func testIndicatorsMeetThreeToOneAcrossThemes() {
        for theme in BeaconThemeCatalog.all {
            for pair in theme.tokens.indicatorPairs {
                XCTAssertGreaterThanOrEqual(
                    pair.foreground.contrastRatio(with: pair.background),
                    3,
                    "\(theme.name) \(pair.name) must meet 3:1"
                )
            }
        }
    }

    func testTerminalPalettesAreCompleteAndReadableAcrossThemes() {
        for theme in BeaconThemeCatalog.all {
            let palette = theme.terminalPalette
            XCTAssertEqual(palette.ansiColors.count, 16, theme.name)
            XCTAssertEqual(palette.ansiColors.first, palette.background, theme.name)

            for pair in palette.readableTextPairs {
                XCTAssertGreaterThanOrEqual(
                    pair.foreground.contrastRatio(with: pair.background),
                    4.5,
                    "\(theme.name) \(pair.name) must meet 4.5:1"
                )
            }
        }
    }

    func testClassicRawAccentsUseAccessibleSemanticAliasesWhenNeeded() {
        let solarized = BeaconThemeCatalog.solarizedDark
        let selenized = BeaconThemeCatalog.selenizedDark
        XCTAssertLessThan(solarized.signatureAccent.contrastRatio(with: solarized.tokens.surface), 4.5)
        XCTAssertGreaterThanOrEqual(solarized.tokens.accent.contrastRatio(with: solarized.tokens.surface), 4.5)
        XCTAssertLessThan(selenized.signatureAccent.contrastRatio(with: selenized.tokens.surface), 4.5)
        XCTAssertGreaterThanOrEqual(selenized.tokens.accent.contrastRatio(with: selenized.tokens.surface), 4.5)
    }

    func testIdentityGrammarIsLabelAndSymbolRedundant() {
        XCTAssertEqual(DashboardLaneIdentity.allCases.map(\.title), ["Local", "Pull Request", "Issue"])
        XCTAssertEqual(Set(DashboardLaneIdentity.allCases.map(\.symbol)).count, 3)
        XCTAssertEqual(Set(DashboardLaneIdentity.allCases.map(\.accent)).count, 3)
    }

    func testDependencyStatusGrammarIsLabelAndSymbolRedundant() {
        let levels: [DependencyUsageLevel] = [.unmeasured, .healthy, .warning, .critical]
        XCTAssertEqual(levels.map(\.title), ["Unmeasured", "Healthy", "Warning", "Critical"])
        XCTAssertEqual(Set(levels.map(\.symbol)).count, levels.count)
    }

    @MainActor
    func testEveryThemePreviewRenders() throws {
        for theme in BeaconThemeCatalog.all {
            let root = BeaconThemePreview(theme: theme, isSelected: true)
                .frame(width: 92, height: 28)
                .preferredColorScheme(theme.appearance.colorScheme)
            let hosting = NSHostingView(rootView: root)
            hosting.frame = NSRect(x: 0, y: 0, width: 92, height: 28)
            hosting.layoutSubtreeIfNeeded()
            let representation = try XCTUnwrap(hosting.bitmapImageRepForCachingDisplay(in: hosting.bounds))
            hosting.cacheDisplay(in: hosting.bounds, to: representation)
            let png = try XCTUnwrap(representation.representation(using: .png, properties: [:]))
            XCTAssertGreaterThan(png.count, 100, "\(theme.name) preview did not render")
        }
    }

    @MainActor
    func testEveryThemeSemanticSmokeViewRenders() throws {
        for theme in BeaconThemeCatalog.all {
            let root = ThemeSemanticSmokeView(theme: theme)
                .frame(width: 520, height: 300)
                .environment(\.beaconTheme, theme)
                .preferredColorScheme(theme.appearance.colorScheme)
            let hosting = NSHostingView(rootView: root)
            hosting.frame = NSRect(x: 0, y: 0, width: 520, height: 300)
            hosting.layoutSubtreeIfNeeded()
            let representation = try XCTUnwrap(hosting.bitmapImageRepForCachingDisplay(in: hosting.bounds))
            hosting.cacheDisplay(in: hosting.bounds, to: representation)
            let png = try XCTUnwrap(representation.representation(using: .png, properties: [:]))
            XCTAssertGreaterThan(png.count, 1_000, "\(theme.name) semantic smoke view did not render")
        }
    }
}

private struct ThemeSemanticSmokeView: View {
    let theme: BeaconTheme

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(theme.name).font(BeaconTypography.bold(18))
                Spacer()
                BeaconThemePreview(theme: theme, isSelected: true)
            }
            HStack(spacing: 8) {
                semanticLabel("Ready", symbol: "checkmark.circle.fill", color: theme.tokens.success.color)
                semanticLabel("Waiting", symbol: "clock.fill", color: theme.tokens.warning.color)
                semanticLabel("Blocked", symbol: "exclamationmark.triangle.fill", color: theme.tokens.danger.color)
                semanticLabel("Recent", symbol: "sparkles", color: theme.tokens.info.color)
            }
            HStack(spacing: 8) {
                identityCard("Local", symbol: "laptopcomputer", color: theme.tokens.identityLocal.color)
                identityCard("PR #40", symbol: "arrow.triangle.pull", color: theme.tokens.identityPullRequest.color)
                identityCard("Issue #39", symbol: "smallcircle.filled.circle", color: theme.tokens.identityIssue.color)
            }
            VStack(alignment: .leading, spacing: 5) {
                Text("Signal Notes").font(BeaconTypography.semibold(13)).foregroundStyle(theme.tokens.editorHeading.color)
                Text("Review `semantic tokens` and [open feedback](https://example.com).")
                    .font(BeaconTypography.regular(11))
                    .foregroundStyle(theme.tokens.editorText.color)
                Text("let theme = \"\(theme.id.rawValue)\"")
                    .font(BeaconTypography.identifier(11))
                    .foregroundStyle(theme.tokens.editorCode.color)
            }
            .padding(10)
            .background(theme.tokens.editorBackground.color, in: RoundedRectangle(cornerRadius: 8))
            .overlay { RoundedRectangle(cornerRadius: 8).strokeBorder(theme.tokens.borderStrong.color) }
        }
        .padding(14)
        .foregroundStyle(theme.tokens.textPrimary.color)
        .background(theme.tokens.canvas.color)
    }

    private func semanticLabel(_ text: String, symbol: String, color: Color) -> some View {
        Label(text, systemImage: symbol)
            .font(BeaconTypography.medium(11))
            .foregroundStyle(color)
            .padding(7)
            .background(theme.tokens.surfaceRaised.color, in: Capsule())
    }

    private func identityCard(_ text: String, symbol: String, color: Color) -> some View {
        Label(text, systemImage: symbol)
            .font(BeaconTypography.identifier(11, weight: .medium))
            .foregroundStyle(color)
            .frame(maxWidth: .infinity, minHeight: 62)
            .background(theme.tokens.surface.color, in: RoundedRectangle(cornerRadius: 8))
            .overlay { RoundedRectangle(cornerRadius: 8).strokeBorder(theme.tokens.border.color) }
    }
}
