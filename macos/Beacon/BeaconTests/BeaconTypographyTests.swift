import AppKit
import XCTest
@testable import Beacon

final class BeaconTypographyTests: XCTestCase {
    func testUsesJetBrainsMonoDefaultAndElevenPointMinimum() {
        XCTAssertEqual(BeaconTypography.familyKey, "beacon.dashboard.font-family")
        XCTAssertEqual(BeaconTypography.defaultFamily, "JetBrainsMono Nerd Font")
        XCTAssertEqual(BeaconTypography.defaultBaseSize, 12)
        XCTAssertEqual(BeaconTypography.resolvedSize(7, baseSize: 11), 11)
        XCTAssertEqual(BeaconTypography.resolvedSize(10, baseSize: 12), 12)
        XCTAssertEqual(BeaconTypography.resolvedSize(17, baseSize: 14), 21)
    }

    func testFontCatalogListsDefaultFirstAndEveryInstalledFamilyOnce() {
        let options = BeaconFontCatalog.selectionOptions(installedFamilies: [
            "Zed Sans",
            " Alpha Mono ",
            "alpha mono",
            "JetBrainsMono Nerd Font",
            "",
        ])

        XCTAssertEqual(options, ["JetBrainsMono Nerd Font", "Alpha Mono", "Zed Sans"])
    }

    func testFontCatalogResolvesInstalledSelectionAndSafeFallbacks() {
        let installed = ["JetBrainsMono Nerd Font", "Menlo"]

        XCTAssertEqual(
            BeaconFontCatalog.resolvedFamily(storedFamily: "menlo", installedFamilies: installed),
            "Menlo"
        )
        XCTAssertEqual(
            BeaconFontCatalog.resolvedFamily(storedFamily: "Missing Font", installedFamilies: installed),
            "JetBrainsMono Nerd Font"
        )
        XCTAssertNil(
            BeaconFontCatalog.resolvedFamily(
                storedFamily: "Missing Font",
                installedFamilies: ["Menlo"]
            )
        )
    }

    func testFontPreferencePersistsAnInstalledFamily() throws {
        let suiteName = "BeaconTypographyTests.\(UUID().uuidString)"
        let defaults = try XCTUnwrap(UserDefaults(suiteName: suiteName))
        defer { defaults.removePersistentDomain(forName: suiteName) }

        defaults.set("Menlo", forKey: BeaconTypography.familyKey)

        XCTAssertEqual(BeaconTypography.requestedFamily(in: defaults), "Menlo")
        XCTAssertEqual(
            BeaconTypography.resolvedFamily(in: defaults, installedFamilies: ["Menlo"]),
            "Menlo"
        )
    }

    func testAppKitRegularAndCodeRolesShareTheInstalledFamilyResolution() throws {
        let family = try XCTUnwrap(BeaconFontCatalog.installedFamilies.first)
        let regular = BeaconTypography.appKitFont(
            10,
            weight: .regular,
            requestedFamily: family,
            installedFamilies: [family]
        )
        let code = BeaconTypography.appKitFont(
            10,
            weight: .medium,
            requestedFamily: family,
            installedFamilies: [family],
            monospacedFallback: true
        )

        XCTAssertEqual(regular.familyName, family)
        XCTAssertEqual(code.familyName, family)
    }
}
