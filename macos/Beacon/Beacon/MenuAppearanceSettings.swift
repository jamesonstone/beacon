import SwiftUI

extension MenuView {
    var appearanceSettingsPanel: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Theme")
                .font(BeaconTypography.semibold(11))
                .foregroundStyle(theme.tokens.textSecondary.color)
            VStack(spacing: 6) {
                ForEach(BeaconThemeCatalog.all) { candidate in
                    Button {
                        themeIDValue = candidate.id.rawValue
                        terminal.refreshAppearance()
                    } label: {
                        HStack(spacing: 9) {
                            BeaconThemePreview(
                                theme: candidate,
                                isSelected: theme.id == candidate.id
                            )
                            VStack(alignment: .leading, spacing: 1) {
                                Text(candidate.name)
                                    .font(BeaconTypography.medium(11))
                                Text(candidate.detail)
                                    .font(BeaconTypography.regular(9))
                                    .foregroundStyle(theme.tokens.textMuted.color)
                            }
                            Spacer()
                            if theme.id == candidate.id {
                                Image(systemName: "checkmark.circle.fill")
                                    .foregroundStyle(theme.tokens.success.color)
                            }
                        }
                        .padding(7)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .background(theme.tokens.surfaceRaised.color, in: RoundedRectangle(cornerRadius: 7))
                    .overlay {
                        RoundedRectangle(cornerRadius: 7)
                            .strokeBorder(
                                theme.id == candidate.id
                                    ? theme.tokens.focus.color
                                    : theme.tokens.border.color,
                                lineWidth: theme.id == candidate.id ? 1.4 : 0.7
                            )
                    }
                    .accessibilityLabel(candidate.accessibilityName)
                    .accessibilityValue(theme.id == candidate.id ? "Selected" : "Not selected")
                    .help(candidate.detail)
                }
            }
            Divider()
            fontPicker
            textSizePicker
            densityPicker
        }
    }

    private var fontPicker: some View {
        HStack {
            Text("Font")
            Spacer()
            Picker("Font", selection: $fontFamilyValue) {
                ForEach(BeaconFontCatalog.selectionOptions, id: \.self) { family in
                    Text(
                        family == BeaconTypography.defaultFamily
                            ? "\(family) — Default"
                            : family
                    )
                    .font(.custom(family, size: 12))
                    .tag(family)
                }
            }
            .labelsHidden()
            .pickerStyle(.menu)
            .frame(width: 230)
        }
    }

    private var textSizePicker: some View {
        HStack {
            Text("Text Size")
            Spacer()
            Picker("Text Size", selection: $fontSizeValue) {
                ForEach(BeaconFontSize.allCases) { size in
                    Text(size.title).tag(size.rawValue)
                }
            }
            .labelsHidden()
            .pickerStyle(.segmented)
            .frame(width: 230)
        }
    }

    private var densityPicker: some View {
        HStack {
            Text("Card Density")
            Spacer()
            Picker("Card Density", selection: $densityValue) {
                ForEach(DashboardDensity.allCases) { density in
                    Text(density.title).tag(density.rawValue)
                }
            }
            .labelsHidden()
            .pickerStyle(.segmented)
            .frame(width: 230)
        }
    }
}
