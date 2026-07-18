import Foundation
import SwiftUI

struct BeaconMarkdownDocument: View {
    @Environment(\.beaconTheme) private var theme
    let source: String
    let baseSize: CGFloat

    init(_ source: String, baseSize: CGFloat = 10) {
        self.source = source
        self.baseSize = baseSize
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            ForEach(BeaconMarkdownDocumentParser.blocks(from: source)) { block in
                blockView(block)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .textSelection(.enabled)
        .tint(theme.tokens.editorLink.color)
    }

    @ViewBuilder
    private func blockView(_ block: BeaconMarkdownBlock) -> some View {
        switch block.kind {
        case let .heading(level):
            markdownText(block.content)
                .font(BeaconTypography.semibold(headingSize(level)))
                .foregroundStyle(theme.tokens.editorHeading.color)
                .padding(.top, level <= 2 ? 3 : 1)
        case .paragraph:
            markdownText(block.content)
                .font(BeaconTypography.regular(baseSize))
                .foregroundStyle(theme.tokens.editorText.color)
                .lineSpacing(2)
        case let .unorderedListItem(depth):
            listRow(marker: "•", content: block.content, depth: depth)
        case let .orderedListItem(ordinal, depth):
            listRow(marker: "\(ordinal).", content: block.content, depth: depth)
        case let .taskListItem(checked, depth):
            taskRow(checked: checked, content: block.content, depth: depth)
        case let .quote(depth):
            quoteRow(content: block.content, depth: depth)
        case .code:
            ScrollView(.horizontal, showsIndicators: false) {
                markdownText(block.content)
                    .font(BeaconTypography.code(baseSize))
                    .foregroundStyle(theme.tokens.editorCode.color)
                    .padding(8)
            }
            .background(theme.tokens.editorCodeBackground.color, in: RoundedRectangle(cornerRadius: 6))
        case .divider:
            Divider().overlay(theme.tokens.borderStrong.color)
        case let .tableRow(header):
            tableRow(cells: block.cells, header: header)
        }
    }

    private func listRow(marker: String, content: AttributedString, depth: Int) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 7) {
            Text(marker)
                .font(BeaconTypography.medium(baseSize))
                .foregroundStyle(theme.tokens.editorSyntax.color)
                .frame(minWidth: 16, alignment: .trailing)
            markdownText(content)
                .font(BeaconTypography.regular(baseSize))
                .foregroundStyle(theme.tokens.editorText.color)
                .lineSpacing(2)
        }
        .padding(.leading, CGFloat(max(0, depth - 1)) * 14)
    }

    private func taskRow(checked: Bool, content: AttributedString, depth: Int) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 7) {
            Image(systemName: checked ? "checkmark.square.fill" : "square")
                .font(.system(size: BeaconTypography.resolvedSize(baseSize), weight: .medium))
                .foregroundStyle(checked ? theme.tokens.success.color : theme.tokens.editorSyntax.color)
                .accessibilityLabel(checked ? "Completed" : "Not completed")
            markdownText(content)
                .font(BeaconTypography.regular(baseSize))
                .foregroundStyle(theme.tokens.editorText.color)
                .lineSpacing(2)
        }
        .padding(.leading, CGFloat(max(0, depth - 1)) * 14)
    }

    private func quoteRow(content: AttributedString, depth: Int) -> some View {
        HStack(alignment: .top, spacing: 8) {
            RoundedRectangle(cornerRadius: 1)
                .fill(theme.tokens.editorQuote.color)
                .frame(width: 3)
            markdownText(content)
                .font(BeaconTypography.regular(baseSize).italic())
                .foregroundStyle(theme.tokens.editorQuote.color)
                .lineSpacing(2)
        }
        .padding(.leading, CGFloat(max(0, depth - 1)) * 10)
    }

    private func tableRow(cells: [AttributedString], header: Bool) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(alignment: .top, spacing: 0) {
                ForEach(Array(cells.enumerated()), id: \.offset) { _, cell in
                    markdownText(cell)
                        .font(header ? BeaconTypography.semibold(baseSize) : BeaconTypography.regular(baseSize))
                        .foregroundStyle(header ? theme.tokens.editorHeading.color : theme.tokens.editorText.color)
                        .frame(minWidth: 120, maxWidth: 220, alignment: .leading)
                        .padding(7)
                        .background(header ? theme.tokens.surfaceRaised.color : theme.tokens.surface.color)
                        .overlay { Rectangle().strokeBorder(theme.tokens.border.color, lineWidth: 0.6) }
                }
            }
        }
    }

    private func markdownText(_ content: AttributedString) -> Text {
        Text(themedInlineContent(content))
    }

    private func themedInlineContent(_ content: AttributedString) -> AttributedString {
        var result = content
        let styles = result.runs.map { ($0.range, $0.inlinePresentationIntent, $0.link) }
        for (range, inlineIntent, link) in styles {
            if inlineIntent?.contains(.code) == true {
                result[range].font = BeaconTypography.code(baseSize)
                result[range].foregroundColor = theme.tokens.editorCode.color
                result[range].backgroundColor = theme.tokens.editorCodeBackground.color
            }
            if link != nil {
                result[range].foregroundColor = theme.tokens.editorLink.color
            }
        }
        return result
    }

    private func headingSize(_ level: Int) -> CGFloat {
        let sizes: [CGFloat] = [16, 14, 13, 12, 11, 11]
        return sizes[min(max(level - 1, 0), sizes.count - 1)]
    }
}
