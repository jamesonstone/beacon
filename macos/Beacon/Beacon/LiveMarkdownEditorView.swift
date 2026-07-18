import AppKit
import SwiftUI

struct LiveMarkdownEditor: NSViewRepresentable {
    @Environment(\.beaconTheme) private var theme
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
        textView.drawsBackground = true
        textView.backgroundColor = theme.tokens.editorBackground.nsColor
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
        textView.insertionPointColor = theme.tokens.focus.nsColor
        textView.selectedTextAttributes = [
            .backgroundColor: theme.tokens.editorSelection.nsColor.withAlphaComponent(0.32),
            .foregroundColor: theme.tokens.editorText.nsColor,
        ]
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
        if context.coordinator.styleSignature != styleSignature {
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
        textView.isContinuousSpellCheckingEnabled = true
        textView.isGrammarCheckingEnabled = false
        textView.isAutomaticSpellingCorrectionEnabled = false
    }

    private func clamped(_ range: NSRange, length: Int) -> NSRange {
        let location = min(range.location, length)
        return NSRange(location: location, length: min(range.length, length - location))
    }

    private var styleSignature: String {
        "\(BeaconTypography.selectionSignature):\(theme.id.rawValue)"
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
            textView.backgroundColor = parent.theme.tokens.editorBackground.nsColor
            textView.insertionPointColor = parent.theme.tokens.focus.nsColor
            textView.selectedTextAttributes = [
                .backgroundColor: parent.theme.tokens.editorSelection.nsColor.withAlphaComponent(0.32),
                .foregroundColor: parent.theme.tokens.editorText.nsColor,
            ]
            LiveMarkdownStyler.apply(to: storage, range: range, theme: parent.theme)
            textView.typingAttributes = LiveMarkdownStyler.typingAttributes(theme: parent.theme)
            textView.setSelectedRange(selection)
            textView.undoManager?.enableUndoRegistration()
            styleSignature = parent.styleSignature
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
