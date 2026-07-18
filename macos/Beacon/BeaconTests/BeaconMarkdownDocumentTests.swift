import SwiftUI
import XCTest
@testable import Beacon

final class BeaconMarkdownDocumentTests: XCTestCase {
    func testParserPreservesIssueDocumentBlockBoundaries() {
        let source = """
        ## Original ask

        Add an inventory export action.

        ## Scope boundaries

        - Add CSV and PDF exports.
        - Allow sorting by any exported column.

        ## Acceptance criteria

        1. The Browse header contains Export.
        2. CSV and PDF downloads work.
        """

        let blocks = BeaconMarkdownDocumentParser.blocks(from: source)

        XCTAssertEqual(blocks.map(\.kind), [
            .heading(level: 2),
            .paragraph,
            .heading(level: 2),
            .unorderedListItem(depth: 1),
            .unorderedListItem(depth: 1),
            .heading(level: 2),
            .orderedListItem(ordinal: 1, depth: 1),
            .orderedListItem(ordinal: 2, depth: 1),
        ])
        XCTAssertEqual(blocks.map(\.plainText), [
            "Original ask",
            "Add an inventory export action.",
            "Scope boundaries",
            "Add CSV and PDF exports.",
            "Allow sorting by any exported column.",
            "Acceptance criteria",
            "The Browse header contains Export.",
            "CSV and PDF downloads work.",
        ])
    }

    func testParserPreservesInlineFormattingAndLinks() throws {
        let source = "Use **strong**, *emphasis*, `code`, ~~removed~~, and [a link](https://example.test)."
        let block = try XCTUnwrap(BeaconMarkdownDocumentParser.blocks(from: source).first)
        let intents = block.content.runs.compactMap(\.inlinePresentationIntent)

        XCTAssertTrue(intents.contains { $0.contains(.stronglyEmphasized) })
        XCTAssertTrue(intents.contains { $0.contains(.emphasized) })
        XCTAssertTrue(intents.contains { $0.contains(.code) })
        XCTAssertTrue(intents.contains { $0.contains(.strikethrough) })
        XCTAssertEqual(
            block.content.runs.compactMap(\.link).map(\.absoluteString),
            ["https://example.test"]
        )
    }

    func testParserFormatsTasksQuotesCodeDividersAndTables() {
        let source = """
        > Keep evidence readable.

        - [ ] Verify the issue body.
        - [x] Preserve the link.

        ---

        | Field | Value |
        | --- | --- |
        | State | Open |

        ```swift
        let formatted = true
        ```
        """

        let blocks = BeaconMarkdownDocumentParser.blocks(from: source)

        XCTAssertEqual(blocks[0].kind, .quote(depth: 1))
        XCTAssertEqual(blocks[1].kind, .taskListItem(checked: false, depth: 1))
        XCTAssertEqual(blocks[1].plainText, "Verify the issue body.")
        XCTAssertEqual(blocks[2].kind, .taskListItem(checked: true, depth: 1))
        XCTAssertEqual(blocks[3].kind, .divider)
        XCTAssertEqual(blocks[4].kind, .tableRow(header: true))
        XCTAssertEqual(blocks[4].cells.map { String($0.characters) }, ["Field", "Value"])
        XCTAssertEqual(blocks[5].kind, .tableRow(header: false))
        XCTAssertEqual(blocks[5].cells.map { String($0.characters) }, ["State", "Open"])
        XCTAssertEqual(blocks[6].kind, .code(language: "swift"))
        XCTAssertTrue(blocks[6].plainText.contains("let formatted = true"))
    }

    func testParserReturnsNoBlocksForEmptyMarkdown() {
        XCTAssertTrue(BeaconMarkdownDocumentParser.blocks(from: "  \n\n ").isEmpty)
    }

    func testNestedMixedListsUseTheNearestListMarkerAndPreserveHardBreaks() {
        let source = """
        1. Parent
           - Child\u{20}\u{20}
             continued
        """

        let blocks = BeaconMarkdownDocumentParser.blocks(from: source)

        XCTAssertEqual(blocks[0].kind, .orderedListItem(ordinal: 1, depth: 1))
        XCTAssertEqual(blocks[1].kind, .unorderedListItem(depth: 2))
        XCTAssertEqual(blocks[1].plainText, "Child\ncontinued")
    }

    @MainActor
    func testDocumentRendersAcrossEveryTheme() throws {
        let source = """
        ## Summary

        **Formatted** body with [link](https://example.test).

        - [ ] One task

        | State | Value |
        | --- | --- |
        | CI | Passing |
        """

        for theme in BeaconThemeCatalog.all {
            let renderer = ImageRenderer(content:
                BeaconMarkdownDocument(source)
                    .environment(\.beaconTheme, theme)
                    .frame(width: 480)
                    .padding()
                    .background(theme.tokens.canvas.color)
            )
            let image = try XCTUnwrap(renderer.nsImage, "Failed to render \(theme.name)")
            XCTAssertGreaterThan(image.size.height, 100)
        }
    }
}
