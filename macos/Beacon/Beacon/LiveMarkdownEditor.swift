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

struct LiveMarkdownEditor: NSViewRepresentable {
    @Binding var text: String
    @Binding var isFocused: Bool
    @Binding var currentLine: String
    let accessibilityLabel: String

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    func makeNSView(context: Context) -> NSScrollView {
        let scrollView = NSScrollView()
        let textView = NSTextView(frame: .zero)
        scrollView.documentView = textView
        scrollView.hasVerticalScroller = true
        scrollView.autohidesScrollers = true
        scrollView.drawsBackground = false

        textView.delegate = context.coordinator
        Self.configureEditing(on: textView)
        textView.drawsBackground = false
        textView.isRichText = false
        textView.importsGraphics = false
        textView.allowsUndo = true
        textView.usesFindBar = true
        textView.isVerticallyResizable = true
        textView.isHorizontallyResizable = false
        textView.autoresizingMask = [.width]
        textView.textContainer?.widthTracksTextView = true
        textView.textContainer?.containerSize = NSSize(
            width: 0,
            height: CGFloat.greatestFiniteMagnitude
        )
        textView.textContainerInset = NSSize(width: 8, height: 8)
        textView.isAutomaticQuoteSubstitutionEnabled = false
        textView.isAutomaticDashSubstitutionEnabled = false
        textView.isAutomaticTextReplacementEnabled = false
        textView.insertionPointColor = NSColor(srgbRed: 0.20, green: 0.91, blue: 1.0, alpha: 1)
        textView.setAccessibilityLabel(accessibilityLabel)
        textView.string = text
        context.coordinator.applyFullStyle(to: textView)
        return scrollView
    }

    func updateNSView(_ scrollView: NSScrollView, context: Context) {
        guard let textView = scrollView.documentView as? NSTextView else { return }
        context.coordinator.parent = self
        var needsFullStyle = false
        if textView.string != text {
            let selection = textView.selectedRange()
            textView.string = text
            textView.setSelectedRange(clamped(selection, length: textView.string.utf16.count))
            needsFullStyle = true
        }
        if context.coordinator.styleSignature != BeaconTypography.selectionSignature {
            needsFullStyle = true
        }
        if needsFullStyle {
            context.coordinator.applyFullStyle(to: textView)
        }
        if LiveMarkdownEditorFocusPolicy.shouldResign(
            wasFocused: context.coordinator.wasFocused,
            isFocused: isFocused
        ), textView.window?.firstResponder === textView {
            textView.window?.makeFirstResponder(nil)
        }
        context.coordinator.wasFocused = isFocused
    }

    static func configureEditing(on textView: NSTextView) {
        textView.isEditable = true
        textView.isSelectable = true
    }

    private func clamped(_ range: NSRange, length: Int) -> NSRange {
        let location = min(range.location, length)
        return NSRange(location: location, length: min(range.length, length - location))
    }

    final class Coordinator: NSObject, NSTextViewDelegate {
        var parent: LiveMarkdownEditor
        var styleSignature = ""
        var wasFocused: Bool
        private var pendingEditRange: NSRange?

        init(parent: LiveMarkdownEditor) {
            self.parent = parent
            wasFocused = parent.isFocused
        }

        func textDidBeginEditing(_ notification: Notification) {
            wasFocused = true
            parent.isFocused = true
            if let textView = notification.object as? NSTextView {
                updateCurrentLine(from: textView)
            }
        }

        func textDidEndEditing(_ notification: Notification) {
            wasFocused = false
            parent.isFocused = false
        }

        func textView(
            _ textView: NSTextView,
            shouldChangeTextIn affectedCharRange: NSRange,
            replacementString: String?
        ) -> Bool {
            pendingEditRange = NSRange(
                location: affectedCharRange.location,
                length: (replacementString as NSString?)?.length ?? 0
            )
            return true
        }

        func textDidChange(_ notification: Notification) {
            guard let textView = notification.object as? NSTextView else { return }
            parent.text = textView.string
            applyStyle(
                to: textView,
                range: editedParagraphRange(in: textView, editedRange: pendingEditRange)
            )
            pendingEditRange = nil
            updateCurrentLine(from: textView)
        }

        func textViewDidChangeSelection(_ notification: Notification) {
            guard let textView = notification.object as? NSTextView else { return }
            updateCurrentLine(from: textView)
        }

        func applyFullStyle(to textView: NSTextView) {
            applyStyle(to: textView, range: nil)
        }

        func updateCurrentLine(from textView: NSTextView) {
            let source = textView.string as NSString
            guard source.length > 0 else {
                parent.currentLine = ""
                return
            }
            let location = min(textView.selectedRange().location, source.length - 1)
            let lineRange = source.lineRange(for: NSRange(location: location, length: 0))
            parent.currentLine = source.substring(with: lineRange)
                .trimmingCharacters(in: .whitespacesAndNewlines)
        }

        private func applyStyle(to textView: NSTextView, range: NSRange?) {
            guard let storage = textView.textStorage else { return }
            let selection = textView.selectedRange()
            textView.undoManager?.disableUndoRegistration()
            LiveMarkdownStyler.apply(to: storage, range: range)
            textView.typingAttributes = LiveMarkdownStyler.typingAttributes
            textView.setSelectedRange(selection)
            textView.undoManager?.enableUndoRegistration()
            styleSignature = BeaconTypography.selectionSignature
        }

        private func editedParagraphRange(in textView: NSTextView, editedRange: NSRange?) -> NSRange? {
            let source = textView.string as NSString
            guard source.length > 0 else { return nil }
            let edit = editedRange ?? textView.selectedRange()
            let start = max(0, min(edit.location, source.length) - 1)
            let editEnd = min(source.length, edit.location + edit.length)
            let end = min(source.length, max(start + 1, editEnd + 1))
            return source.paragraphRange(for: NSRange(location: start, length: end - start))
        }
    }
}

enum LiveMarkdownEditorFocusPolicy {
    static func shouldResign(wasFocused: Bool, isFocused: Bool) -> Bool {
        wasFocused && !isFocused
    }
}
