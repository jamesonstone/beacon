import Foundation
import SwiftUI

enum BeaconMarkdownBlockKind: Equatable {
    case paragraph
    case heading(level: Int)
    case unorderedListItem(depth: Int)
    case orderedListItem(ordinal: Int, depth: Int)
    case taskListItem(checked: Bool, depth: Int)
    case quote(depth: Int)
    case code(language: String?)
    case divider
    case tableRow(header: Bool)
}

struct BeaconMarkdownBlock: Identifiable {
    let id: Int
    let kind: BeaconMarkdownBlockKind
    let content: AttributedString
    let cells: [AttributedString]

    var plainText: String { String(content.characters) }
}

enum BeaconMarkdownDocumentParser {
    private struct CellAccumulator {
        let id: Int
        var content: AttributedString
    }

    private struct BlockAccumulator {
        let id: Int
        let intent: PresentationIntent?
        var content = AttributedString()
        var cells: [CellAccumulator] = []
        var lastLeafID: Int?

        mutating func append(_ fragment: AttributedString, intent: PresentationIntent?) {
            if let cellID = BeaconMarkdownDocumentParser.tableCellID(in: intent) {
                if cells.last?.id == cellID {
                    cells[cells.count - 1].content.append(fragment)
                } else {
                    cells.append(CellAccumulator(id: cellID, content: fragment))
                }
                return
            }

            let leafID = intent?.components.first?.identity
            if !content.characters.isEmpty, leafID != lastLeafID {
                content.append(AttributedString("\n\n"))
            }
            content.append(fragment)
            lastLeafID = leafID
        }
    }

    static func blocks(from source: String) -> [BeaconMarkdownBlock] {
        guard !source.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return [] }
        guard let document = try? AttributedString(markdown: source) else {
            return [BeaconMarkdownBlock(
                id: 0,
                kind: .paragraph,
                content: AttributedString(source),
                cells: []
            )]
        }

        var accumulators: [BlockAccumulator] = []
        for run in document.runs {
            let intent = run.presentationIntent
            let identity = blockID(in: intent) ?? -(accumulators.count + 1)
            let fragment = AttributedString(document[run.range])
            if accumulators.last?.id != identity {
                accumulators.append(BlockAccumulator(id: identity, intent: intent))
            }
            accumulators[accumulators.count - 1].append(fragment, intent: intent)
        }

        return accumulators.map { accumulator in
            var content = accumulator.content
            let kind = kind(for: accumulator.intent, content: content)
            if case .taskListItem = kind {
                content = removingTaskMarker(from: content)
            }
            return BeaconMarkdownBlock(
                id: accumulator.id,
                kind: kind,
                content: content,
                cells: accumulator.cells.map(\.content)
            )
        }
    }

    private static func blockID(in intent: PresentationIntent?) -> Int? {
        guard let components = intent?.components else { return nil }
        return components.first(where: { isTableRow($0.kind) })?.identity
            ?? components.first(where: { isListItem($0.kind) })?.identity
            ?? components.first(where: { isQuote($0.kind) })?.identity
            ?? components.first?.identity
    }

    private static func tableCellID(in intent: PresentationIntent?) -> Int? {
        intent?.components.first(where: {
            if case .tableCell = $0.kind { return true }
            return false
        })?.identity
    }

    private static func kind(
        for intent: PresentationIntent?,
        content: AttributedString
    ) -> BeaconMarkdownBlockKind {
        let components = intent?.components ?? []
        for component in components {
            if case let .header(level) = component.kind { return .heading(level: level) }
            if case let .codeBlock(language) = component.kind { return .code(language: language) }
            if case .thematicBreak = component.kind { return .divider }
            if case .tableHeaderRow = component.kind { return .tableRow(header: true) }
            if case .tableRow = component.kind { return .tableRow(header: false) }
        }

        let listDepth = max(1, components.filter { isList($0.kind) }.count)
        if let item = components.first(where: { isListItem($0.kind) }) {
            if let task = taskMarker(in: content) {
                return .taskListItem(checked: task.checked, depth: listDepth)
            }
            let nearestList = components.first(where: { isList($0.kind) })
            let ordered = nearestList.map {
                if case .orderedList = $0.kind { return true }
                return false
            } ?? false
            if case let .listItem(ordinal) = item.kind, ordered {
                return .orderedListItem(ordinal: ordinal, depth: listDepth)
            }
            return .unorderedListItem(depth: listDepth)
        }

        let quoteDepth = components.filter { isQuote($0.kind) }.count
        return quoteDepth > 0 ? .quote(depth: quoteDepth) : .paragraph
    }

    private static func taskMarker(in content: AttributedString) -> (checked: Bool, length: Int)? {
        let value = String(content.characters)
        guard value.count >= 3, value.first == "[", value.dropFirst(2).first == "]" else { return nil }
        let marker = value[value.index(after: value.startIndex)]
        guard marker == " " || marker == "x" || marker == "X" else { return nil }
        let markerLength = value.dropFirst(3).first == " " ? 4 : 3
        return (marker != " ", markerLength)
    }

    private static func removingTaskMarker(from content: AttributedString) -> AttributedString {
        guard let marker = taskMarker(in: content) else { return content }
        var result = content
        let end = result.characters.index(result.startIndex, offsetBy: marker.length)
        result.removeSubrange(result.startIndex ..< end)
        return result
    }

    private static func isTableRow(_ kind: PresentationIntent.Kind) -> Bool {
        if case .tableHeaderRow = kind { return true }
        if case .tableRow = kind { return true }
        return false
    }

    private static func isListItem(_ kind: PresentationIntent.Kind) -> Bool {
        if case .listItem = kind { return true }
        return false
    }

    private static func isList(_ kind: PresentationIntent.Kind) -> Bool {
        if case .orderedList = kind { return true }
        if case .unorderedList = kind { return true }
        return false
    }

    private static func isQuote(_ kind: PresentationIntent.Kind) -> Bool {
        if case .blockQuote = kind { return true }
        return false
    }
}
