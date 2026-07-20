import Foundation

enum BeaconThemeCatalog {
    static let all: [BeaconTheme] = [
        lobsterNebula,
        pampasMoon,
        solarizedDark,
        monokai,
        selenizedDark,
    ]

    static func theme(for id: BeaconThemeID) -> BeaconTheme {
        all.first(where: { $0.id == id }) ?? lobsterNebula
    }

    static func theme(forStoredID value: String?) -> BeaconTheme {
        theme(for: BeaconThemePreference.resolvedID(value))
    }

    static let lobsterNebula = BeaconTheme(
        id: .lobsterNebula,
        name: "Lobster Nebula",
        detail: "Recommended · warm, quiet dark",
        appearance: .dark,
        signatureAccent: color("#E9785D"),
        isRecommended: true,
        projectWatermark: BeaconProjectWatermarkPalette(
            base: color("#0B0C0E"), highlightLeading: color("#16070A"),
            highlightCenter: color("#041019"), highlightTrailing: color("#120811")
        ),
        tokens: BeaconThemeTokens(
            canvas: color("#151619"), surface: color("#1C1E22"),
            surfaceRaised: color("#24272C"), surfaceOverlay: color("#2C3036"),
            border: color("#3C4149"), borderStrong: color("#747A84"),
            textPrimary: color("#F3F0E8"), textSecondary: color("#C7C2B8"),
            textMuted: color("#A6A197"), accent: color("#E9785D"),
            focus: color("#F59A82"), success: color("#78C995"),
            warning: color("#F0C36E"), danger: color("#FF8E80"),
            info: color("#76C7F2"), identityLocal: color("#78C995"),
            identityPullRequest: color("#76C7F2"), identityIssue: color("#E6A0C4"),
            editorBackground: color("#17191D"), editorText: color("#F3F0E8"),
            editorHeading: color("#F59A82"), editorLink: color("#76C7F2"),
            editorCode: color("#F0C36E"), editorCodeBackground: color("#272A30"),
            editorQuote: color("#C7C2B8"), editorSyntax: color("#A6A197"),
            editorSelection: color("#F59A82")
        )
    )

    static let pampasMoon = BeaconTheme(
        id: .pampasMoon,
        name: "Pampas Moon",
        detail: "High-readability light",
        appearance: .light,
        signatureAccent: color("#A95135"),
        isRecommended: false,
        projectWatermark: BeaconProjectWatermarkPalette(
            base: color("#F4F0E9"), highlightLeading: color("#F7EFEC"),
            highlightCenter: color("#ECF3F7"), highlightTrailing: color("#F5EFF3")
        ),
        tokens: BeaconThemeTokens(
            canvas: color("#F7F4ED"), surface: color("#FFFFFF"),
            surfaceRaised: color("#F7F4ED"), surfaceOverlay: color("#FFFFFF"),
            border: color("#D8D1C6"), borderStrong: color("#777067"),
            textPrimary: color("#292724"), textSecondary: color("#5F5A53"),
            textMuted: color("#716B63"), accent: color("#A95135"),
            focus: color("#8E3F2B"), success: color("#2D6B43"),
            warning: color("#6F5200"), danger: color("#9E352F"),
            info: color("#1F5D86"), identityLocal: color("#2D6B43"),
            identityPullRequest: color("#1F5D86"), identityIssue: color("#7B3E62"),
            editorBackground: color("#FFFFFF"), editorText: color("#292724"),
            editorHeading: color("#A95135"), editorLink: color("#1F5D86"),
            editorCode: color("#6F5200"), editorCodeBackground: color("#F0ECE3"),
            editorQuote: color("#5F5A53"), editorSyntax: color("#716B63"),
            editorSelection: color("#A95135")
        )
    )

    static let solarizedDark = BeaconTheme(
        id: .solarizedDark,
        name: "Solarized Dark",
        detail: "Classic balanced dark",
        appearance: .dark,
        signatureAccent: color("#268BD2"),
        isRecommended: false,
        projectWatermark: BeaconProjectWatermarkPalette(
            base: color("#002B36"), highlightLeading: color("#002432"),
            highlightCenter: color("#0A2825"), highlightTrailing: color("#1C1D2A")
        ),
        tokens: BeaconThemeTokens(
            canvas: color("#002B36"), surface: color("#073642"),
            surfaceRaised: color("#06313B"), surfaceOverlay: color("#04303A"),
            border: color("#38616A"), borderStrong: color("#8FA7A7"),
            textPrimary: color("#EEE8D5"), textSecondary: color("#93A1A1"),
            textMuted: color("#8FA0A0"), accent: color("#5DB7E8"),
            focus: color("#76C8F4"), success: color("#75C79B"),
            warning: color("#EBCB70"), danger: color("#FF8F84"),
            info: color("#72C5F2"), identityLocal: color("#75C79B"),
            identityPullRequest: color("#72C5F2"), identityIssue: color("#E7A4C7"),
            editorBackground: color("#002B36"), editorText: color("#EEE8D5"),
            editorHeading: color("#76C8F4"), editorLink: color("#72C5F2"),
            editorCode: color("#EBCB70"), editorCodeBackground: color("#0C4452"),
            editorQuote: color("#93A1A1"), editorSyntax: color("#8FA0A0"),
            editorSelection: color("#76C8F4")
        )
    )

    static let monokai = BeaconTheme(
        id: .monokai,
        name: "Monokai",
        detail: "Crisp code-forward dark",
        appearance: .dark,
        signatureAccent: color("#66D9EF"),
        isRecommended: false,
        projectWatermark: BeaconProjectWatermarkPalette(
            base: color("#272822"), highlightLeading: color("#13242A"),
            highlightCenter: color("#182711"), highlightTrailing: color("#251820")
        ),
        tokens: BeaconThemeTokens(
            canvas: color("#272822"), surface: color("#32332C"),
            surfaceRaised: color("#2D2E27"), surfaceOverlay: color("#292A24"),
            border: color("#5C5E54"), borderStrong: color("#9C9E91"),
            textPrimary: color("#F8F8F2"), textSecondary: color("#CFCFC2"),
            textMuted: color("#B1B1A6"), accent: color("#66D9EF"),
            focus: color("#8BE9FD"), success: color("#A6E22E"),
            warning: color("#E6C85C"), danger: color("#FF7A90"),
            info: color("#66D9EF"), identityLocal: color("#A6E22E"),
            identityPullRequest: color("#66D9EF"), identityIssue: color("#E8A3C4"),
            editorBackground: color("#272822"), editorText: color("#F8F8F2"),
            editorHeading: color("#8BE9FD"), editorLink: color("#66D9EF"),
            editorCode: color("#E6C85C"), editorCodeBackground: color("#3B3D35"),
            editorQuote: color("#CFCFC2"), editorSyntax: color("#B1B1A6"),
            editorSelection: color("#8BE9FD")
        )
    )

    static let selenizedDark = BeaconTheme(
        id: .selenizedDark,
        name: "Selenized Dark",
        detail: "Calm blue-green dark",
        appearance: .dark,
        signatureAccent: color("#58A3FF"),
        isRecommended: false,
        projectWatermark: BeaconProjectWatermarkPalette(
            base: color("#103C48"), highlightLeading: color("#123346"),
            highlightCenter: color("#123A30"), highlightTrailing: color("#292F3D")
        ),
        tokens: BeaconThemeTokens(
            canvas: color("#103C48"), surface: color("#184956"),
            surfaceRaised: color("#143F4B"), surfaceOverlay: color("#103C48"),
            border: color("#477581"), borderStrong: color("#9AB4B7"),
            textPrimary: color("#CAD8D9"), textSecondary: color("#ADBCBC"),
            textMuted: color("#A3B4B4"), accent: color("#83BCFF"),
            focus: color("#9CCAFF"), success: color("#80CFA0"),
            warning: color("#E8C97A"), danger: color("#FF9A8E"),
            info: color("#83BCFF"), identityLocal: color("#80CFA0"),
            identityPullRequest: color("#83BCFF"), identityIssue: color("#E7A7C8"),
            editorBackground: color("#103C48"), editorText: color("#CAD8D9"),
            editorHeading: color("#9CCAFF"), editorLink: color("#83BCFF"),
            editorCode: color("#E8C97A"), editorCodeBackground: color("#205866"),
            editorQuote: color("#ADBCBC"), editorSyntax: color("#A3B4B4"),
            editorSelection: color("#9CCAFF")
        )
    )

    private static func color(_ hex: String) -> BeaconSRGB {
        BeaconSRGB(hex: hex)
    }
}
