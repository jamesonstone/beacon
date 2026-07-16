import AppKit
import SwiftUI

enum MarkdownStyleRole: Equatable {
    case heading(level: Int)
    case strong
    case emphasis
    case inlineCode
    case strikethrough
    case link
    case quote
    case divider
    case syntaxMarker
}
struct MarkdownStyleSpan: Equatable {
    let role: MarkdownStyleRole
    let range: NSRange
}

enum LiveMarkdownStyler {
    private static let heading = try? NSRegularExpression(
        pattern: #"^(#{1,6})[\t ]+(.+)$"#,
        options: .anchorsMatchLines
    )
    private static let list = try? NSRegularExpression(
        pattern: #"^[\t ]*((?:[-+*])|(?:\d+\.))[\t ]+"#,
        options: .anchorsMatchLines
    )
    private static let quote = try? NSRegularExpression(
        pattern: #"^[\t ]*(>+)[\t ]?.*$"#,
        options: .anchorsMatchLines
    )
    private static let divider = try? NSRegularExpression(
        pattern: #"^[\t ]{0,3}(?:(?:\*[\t ]*){3,}|(?:-[\t ]*){3,}|(?:_[\t ]*){3,})$"#,
        options: .anchorsMatchLines
    )
    private static let strong = try? NSRegularExpression(pattern: #"\*\*(?=\S)(.+?\S)\*\*"#)
    private static let emphasis = try? NSRegularExpression(pattern: #"(?<!\*)\*(?=\S)([^*\n]*?\S)\*(?!\*)"#)
    private static let inlineCode = try? NSRegularExpression(pattern: #"`([^`\n]+)`"#)
    private static let strikethrough = try? NSRegularExpression(pattern: #"~~(?=\S)(.+?\S)~~"#)
    private static let link = try? NSRegularExpression(pattern: #"\[([^\]\n]+)\]\(([^)\n]+)\)"#)

    private static let mint = NSColor(srgbRed: 0.42, green: 1.0, blue: 0.76, alpha: 1)
    private static let lavender = NSColor(srgbRed: 0.70, green: 0.58, blue: 1.0, alpha: 1)
    private static let cyan = NSColor(srgbRed: 0.20, green: 0.91, blue: 1.0, alpha: 1)
    private static let pink = NSColor(srgbRed: 1.0, green: 0.36, blue: 0.76, alpha: 1)

    static func spans(in source: String, range requestedRange: NSRange? = nil) -> [MarkdownStyleSpan] {
        let sourceLength = (source as NSString).length
        let fullRange = NSRange(location: 0, length: sourceLength)
        let target = requestedRange.map { NSIntersectionRange(fullRange, $0) } ?? fullRange
        guard target.length > 0 else { return [] }

        let segment = (source as NSString).substring(with: target)
        let segmentRange = NSRange(location: 0, length: (segment as NSString).length)
        var spans: [MarkdownStyleSpan] = []

        for match in matches(heading, text: segment, range: segmentRange) {
            let marker = shifted(match.range(at: 1), by: target.location)
            spans.append(MarkdownStyleSpan(
                role: .heading(level: marker.length),
                range: shifted(match.range, by: target.location)
            ))
            spans.append(MarkdownStyleSpan(role: .syntaxMarker, range: marker))
        }
        for match in matches(list, text: segment, range: segmentRange) {
            spans.append(MarkdownStyleSpan(
                role: .syntaxMarker,
                range: shifted(match.range(at: 1), by: target.location)
            ))
        }
        for match in matches(quote, text: segment, range: segmentRange) {
            spans.append(MarkdownStyleSpan(role: .quote, range: shifted(match.range, by: target.location)))
            spans.append(MarkdownStyleSpan(
                role: .syntaxMarker,
                range: shifted(match.range(at: 1), by: target.location)
            ))
        }
        appendMatches(divider, role: .divider, text: segment, range: segmentRange, offset: target.location, to: &spans)
        appendMatches(strong, role: .strong, text: segment, range: segmentRange, offset: target.location, to: &spans)
        appendMatches(emphasis, role: .emphasis, text: segment, range: segmentRange, offset: target.location, to: &spans)
        appendMatches(inlineCode, role: .inlineCode, text: segment, range: segmentRange, offset: target.location, to: &spans)
        appendMatches(strikethrough, role: .strikethrough, text: segment, range: segmentRange, offset: target.location, to: &spans)
        appendMatches(link, role: .link, text: segment, range: segmentRange, offset: target.location, to: &spans)
        return spans
    }

    static func apply(to textStorage: NSTextStorage, range requestedRange: NSRange? = nil) {
        let fullRange = NSRange(location: 0, length: textStorage.length)
        let target = requestedRange.map { NSIntersectionRange(fullRange, $0) } ?? fullRange
        guard target.length > 0 else { return }

        let baseParagraph = NSMutableParagraphStyle()
        baseParagraph.lineSpacing = 2
        baseParagraph.paragraphSpacing = 2
        textStorage.beginEditing()
        textStorage.setAttributes([
            .font: BeaconTypography.appKitFont(10),
            .foregroundColor: mint,
            .paragraphStyle: baseParagraph,
        ], range: target)

        for span in spans(in: textStorage.string, range: target) {
            switch span.role {
            case let .heading(level):
                let sizeOffsets: [CGFloat] = [8, 6, 4, 2, 1, 0]
                let offset = sizeOffsets[min(max(level - 1, 0), sizeOffsets.count - 1)]
                let paragraph = baseParagraph.mutableCopy() as? NSMutableParagraphStyle ?? baseParagraph
                paragraph.paragraphSpacingBefore = level <= 2 ? 8 : 4
                paragraph.paragraphSpacing = 4
                textStorage.addAttributes([
                    .font: BeaconTypography.appKitFont(10 + offset, weight: .bold),
                    .foregroundColor: mint,
                    .paragraphStyle: paragraph,
                ], range: span.range)
            case .strong:
                convertFont(in: textStorage, range: span.range, trait: .boldFontMask)
            case .emphasis:
                convertFont(in: textStorage, range: span.range, trait: .italicFontMask)
            case .inlineCode:
                textStorage.addAttributes([
                    .font: NSFont.monospacedSystemFont(
                        ofSize: BeaconTypography.resolvedSize(10),
                        weight: .medium
                    ),
                    .foregroundColor: cyan,
                    .backgroundColor: cyan.withAlphaComponent(0.10),
                ], range: span.range)
            case .strikethrough:
                textStorage.addAttribute(
                    .strikethroughStyle,
                    value: NSUnderlineStyle.single.rawValue,
                    range: span.range
                )
            case .link:
                textStorage.addAttributes([
                    .foregroundColor: cyan,
                    .underlineStyle: NSUnderlineStyle.single.rawValue,
                ], range: span.range)
            case .quote:
                let paragraph = baseParagraph.mutableCopy() as? NSMutableParagraphStyle ?? baseParagraph
                paragraph.headIndent = 14
                textStorage.addAttributes([
                    .foregroundColor: lavender,
                    .paragraphStyle: paragraph,
                ], range: span.range)
                convertFont(in: textStorage, range: span.range, trait: .italicFontMask)
            case .divider:
                textStorage.addAttributes([
                    .foregroundColor: pink,
                    .font: BeaconTypography.appKitFont(10, weight: .semibold),
                ], range: span.range)
            case .syntaxMarker:
                textStorage.addAttribute(.foregroundColor, value: lavender.withAlphaComponent(0.78), range: span.range)
            }
        }
        textStorage.endEditing()
    }

    static var typingAttributes: [NSAttributedString.Key: Any] {
        [.font: BeaconTypography.appKitFont(10), .foregroundColor: mint]
    }

    private static func matches(
        _ expression: NSRegularExpression?,
        text: String,
        range: NSRange
    ) -> [NSTextCheckingResult] {
        expression?.matches(in: text, range: range) ?? []
    }

    private static func appendMatches(
        _ expression: NSRegularExpression?,
        role: MarkdownStyleRole,
        text: String,
        range: NSRange,
        offset: Int,
        to spans: inout [MarkdownStyleSpan]
    ) {
        for match in matches(expression, text: text, range: range) {
            spans.append(MarkdownStyleSpan(role: role, range: shifted(match.range, by: offset)))
        }
    }

    private static func shifted(_ range: NSRange, by offset: Int) -> NSRange {
        NSRange(location: range.location + offset, length: range.length)
    }

    private static func convertFont(in storage: NSTextStorage, range: NSRange, trait: NSFontTraitMask) {
        storage.enumerateAttribute(.font, in: range) { value, subrange, _ in
            let font = value as? NSFont ?? BeaconTypography.appKitFont(10)
            let converted = NSFontManager.shared.convert(font, toHaveTrait: trait)
            storage.addAttribute(.font, value: converted, range: subrange)
        }
    }
}
