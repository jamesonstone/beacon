import AppKit
import XCTest
@testable import Beacon

final class ModelsTests: XCTestCase {
    func testExternalActivityChipPrioritizesAttentionAndCountsProviders() {
        let records = [
            externalActivityRecord(provider: "codex", state: "working", session: "one"),
            externalActivityRecord(provider: "claude-code", state: "turn_finished", session: "two"),
            externalActivityRecord(provider: "claude-code", state: "needs_attention", session: "three"),
        ]

        let chip = ExternalActivityPresentation.chip(for: records)

        XCTAssertEqual(chip?.label, "Agents · Needs attention · 3")
        XCTAssertEqual(chip?.state, "needs_attention")
        XCTAssertEqual(chip?.sessionCount, 3)
        XCTAssertEqual(
            ExternalActivityPresentation.chip(for: [records[0]])?.label,
            "Codex · Working"
        )
    }

    func testExternalActivityCacheDecodesOutsideEvidenceSnapshot() throws {
        let data = Data(#"{"version":1,"records":[{"provider":"codex","state":"turn_finished","session_key":"hashed","project_id":"owner/repo","lane_id":"lane-31","observed_at":"2026-07-16T12:00:00Z","expires_at":"2026-07-16T13:00:00Z"}],"next_expiry":"2026-07-16T13:00:00Z"}"#.utf8)

        let activity = try JSONDecoder().decode(ExternalActivitySnapshot.self, from: data)

        XCTAssertEqual(activity.records.first?.laneID, "lane-31")
        XCTAssertEqual(activity.records.first?.state, "turn_finished")
        XCTAssertEqual(activity.nextExpiry, "2026-07-16T13:00:00Z")
    }

    func testSignalNotesSavedLabelIncludesFormattedAge() {
        XCTAssertEqual(
            SignalNotesPresentation.savedLabel(age: "2 minutes ago"),
            "Saved 2 minutes ago"
        )
    }

    func testNotesPresentationCyclesAndUsesRequestedFractions() {
        XCTAssertEqual(SignalNotesSize.half.nextCycled, .eighty)
        XCTAssertEqual(SignalNotesSize.eighty.nextCycled, .minimized)
        XCTAssertEqual(SignalNotesSize.minimized.nextCycled, .half)
        XCTAssertEqual(SignalNotesSize.half.heightFraction, 0.5)
        XCTAssertEqual(SignalNotesSize.eighty.heightFraction, 0.8)
        XCTAssertNil(SignalNotesSize.minimized.heightFraction)
        XCTAssertEqual(SignalNotesPresentation.expandedHeightFraction, 0.5)
        XCTAssertEqual(SignalNotesPresentation.enlargedHeightFraction, 0.8)
    }

    func testSpaceMotionProducesStableLoopPhase() {
        let start = Date(timeIntervalSinceReferenceDate: 0)
        XCTAssertEqual(BeaconSpaceMotion.phase(at: start, duration: 10), 0)
        XCTAssertEqual(BeaconSpaceMotion.phase(at: start.addingTimeInterval(2.5), duration: 10), 0.25)
        XCTAssertEqual(BeaconSpaceMotion.phase(at: start.addingTimeInterval(12.5), duration: 10), 0.25)

        let startOffset = BeaconSpaceMotion.orbitOffset(
            at: 0,
            horizontalRadius: 7,
            verticalRadius: 5.5
        )
        let quarterOffset = BeaconSpaceMotion.orbitOffset(
            at: 0.25,
            horizontalRadius: 7,
            verticalRadius: 5.5
        )
        XCTAssertEqual(startOffset.width, 7, accuracy: 0.001)
        XCTAssertEqual(startOffset.height, 0, accuracy: 0.001)
        XCTAssertEqual(quarterOffset.width, 0, accuracy: 0.001)
        XCTAssertEqual(quarterOffset.height, 5.5, accuracy: 0.001)
    }

    func testSignalNotesLiveMarkdownStylesWithoutChangingSource() {
        let source = "## Plan\n\n**Ship carefully.**\n> Verify `main`.\n[Open](https://example.test)\n---"
        let spans = LiveMarkdownStyler.spans(in: source)
        XCTAssertTrue(spans.contains { $0.role == .heading(level: 2) })
        XCTAssertTrue(spans.contains { $0.role == .strong })
        XCTAssertTrue(spans.contains { $0.role == .quote })
        XCTAssertTrue(spans.contains { $0.role == .inlineCode })
        XCTAssertTrue(spans.contains { $0.role == .link })
        XCTAssertTrue(spans.contains { $0.role == .divider })

        let storage = NSTextStorage(string: source)
        LiveMarkdownStyler.apply(to: storage)
        XCTAssertEqual(storage.string, source)

        let headingFont = storage.attribute(.font, at: 0, effectiveRange: nil) as? NSFont
        let bodyLocation = (source as NSString).range(of: "Ship").location
        let bodyFont = storage.attribute(.font, at: bodyLocation, effectiveRange: nil) as? NSFont
        XCTAssertGreaterThan(try XCTUnwrap(headingFont).pointSize, try XCTUnwrap(bodyFont).pointSize)
    }

    func testSignalNotesWrappedListContentUsesHangingIndent() throws {
        let source = """
        [] Verify collection dates and exception retries after the first rendered line wraps
        - [ ] Confirm standard task behavior
        - Confirm top-level bullet behavior
            - Confirm nested bullet behavior
        [ ] Confirm bare checkbox behavior
        10. Confirm numbered list behavior
        """
        let storage = NSTextStorage(string: source)

        LiveMarkdownStyler.apply(to: storage)

        func paragraphStyle(at text: String) throws -> NSParagraphStyle {
            let location = (source as NSString).range(of: text).location
            return try XCTUnwrap(
                storage.attribute(.paragraphStyle, at: location, effectiveRange: nil) as? NSParagraphStyle
            )
        }

        let taskStyle = try paragraphStyle(at: "Verify collection")
        let standardTaskStyle = try paragraphStyle(at: "Confirm standard")
        let bulletStyle = try paragraphStyle(at: "Confirm top-level")
        let nestedStyle = try paragraphStyle(at: "Confirm nested")
        let bareTaskStyle = try paragraphStyle(at: "Confirm bare")
        let numberedStyle = try paragraphStyle(at: "Confirm numbered")
        XCTAssertEqual(taskStyle.firstLineHeadIndent, 0)
        XCTAssertGreaterThan(taskStyle.headIndent, 0)
        XCTAssertGreaterThan(standardTaskStyle.headIndent, 0)
        XCTAssertGreaterThan(nestedStyle.headIndent, bulletStyle.headIndent)
        XCTAssertGreaterThan(bareTaskStyle.headIndent, 0)
        XCTAssertGreaterThan(numberedStyle.headIndent, 0)
        XCTAssertEqual(storage.string, source)

        let layoutManager = NSLayoutManager()
        let textContainer = NSTextContainer(
            size: NSSize(width: 180, height: CGFloat.greatestFiniteMagnitude)
        )
        textContainer.lineFragmentPadding = 0
        layoutManager.addTextContainer(textContainer)
        storage.addLayoutManager(layoutManager)
        layoutManager.ensureLayout(for: textContainer)

        var lineOrigins: [CGFloat] = []
        layoutManager.enumerateLineFragments(
            forGlyphRange: layoutManager.glyphRange(for: textContainer)
        ) { _, usedRect, _, _, stop in
            lineOrigins.append(usedRect.minX)
            if lineOrigins.count == 2 {
                stop.pointee = true
            }
        }

        XCTAssertGreaterThan(lineOrigins.count, 1)
        XCTAssertEqual(try XCTUnwrap(lineOrigins.first), 0, accuracy: 0.5)
        XCTAssertEqual(lineOrigins[1], taskStyle.headIndent, accuracy: 0.5)
    }

    func testSignalNotesEditorIsWritableAndOnlyResignsAfterFocusTransition() {
        let textView = NSTextView()
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 320, height: 200),
            styleMask: [.titled],
            backing: .buffered,
            defer: false
        )
        window.contentView = textView

        LiveMarkdownEditor.configureEditing(on: textView)

        XCTAssertTrue(textView.isEditable)
        XCTAssertTrue(textView.isSelectable)
        XCTAssertTrue(window.makeFirstResponder(textView))
        textView.insertText("General note", replacementRange: NSRange(location: 0, length: 0))
        XCTAssertEqual(textView.string, "General note")
        XCTAssertFalse(LiveMarkdownEditorFocusPolicy.shouldResign(
            wasFocused: false,
            isFocused: false
        ))
        XCTAssertFalse(LiveMarkdownEditorFocusPolicy.shouldResign(
            wasFocused: true,
            isFocused: true
        ))
        XCTAssertTrue(LiveMarkdownEditorFocusPolicy.shouldResign(
            wasFocused: true,
            isFocused: false
        ))
    }

    func testSignalNotesEditorEnablesLeanSpellCheckingAndPreservesIndicators() throws {
        let textView = NSTextView()
        textView.string = "mispelled note"

        LiveMarkdownEditor.configureEditing(on: textView)

        XCTAssertTrue(textView.isContinuousSpellCheckingEnabled)
        XCTAssertFalse(textView.isGrammarCheckingEnabled)
        XCTAssertFalse(textView.isAutomaticSpellingCorrectionEnabled)

        let layoutManager = try XCTUnwrap(textView.layoutManager)
        let spellingState = NSAttributedString.SpellingState.spelling.rawValue
        layoutManager.addTemporaryAttribute(
            .spellingState,
            value: spellingState,
            forCharacterRange: NSRange(location: 0, length: 9)
        )

        LiveMarkdownStyler.apply(to: try XCTUnwrap(textView.textStorage))

        let preservedState = layoutManager.temporaryAttribute(
            .spellingState,
            atCharacterIndex: 0,
            effectiveRange: nil
        ) as? NSNumber
        XCTAssertEqual(preservedState?.intValue, spellingState)
    }

}
