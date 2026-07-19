import XCTest
@testable import Beacon

final class DashboardFittedModeTests: XCTestCase {
    func testFittedLayoutSelectsLargestNoOverflowScale() {
        let available = CGSize(width: 1_000, height: 500)
        let layout = FittedFollowingPresentation.layout(
            in: available,
            sectionItemCounts: [4, 3, 2]
        )

        XCTAssertEqual(layout.columns, 4)
        XCTAssertEqual(layout.rows, 3)
        XCTAssertEqual(layout.scale, 1)
        XCTAssertLessThanOrEqual(layout.contentSize.width * layout.scale, available.width)
        XCTAssertLessThanOrEqual(layout.contentSize.height * layout.scale, available.height)
    }

    func testFittedLayoutScalesEverySectionIntoCompactBounds() {
        let available = CGSize(width: 430, height: 180)
        let layout = FittedFollowingPresentation.layout(
            in: available,
            sectionItemCounts: [4, 3, 2]
        )

        XCTAssertEqual(layout.columns, 4)
        XCTAssertEqual(layout.rows, 3)
        XCTAssertLessThan(layout.scale, 1)
        XCTAssertGreaterThan(layout.scale, 0)
        XCTAssertLessThanOrEqual(layout.contentSize.width * layout.scale, available.width + 0.001)
        XCTAssertLessThanOrEqual(layout.contentSize.height * layout.scale, available.height + 0.001)
    }

    func testFittedLayoutHandlesNoFollowingItems() {
        let layout = FittedFollowingPresentation.layout(
            in: CGSize(width: 430, height: 180),
            sectionItemCounts: [0, 0, 0]
        )

        XCTAssertEqual(layout.columns, 1)
        XCTAssertEqual(layout.rows, 0)
        XCTAssertEqual(layout.scale, 1)
        XCTAssertEqual(layout.contentSize, .zero)
    }

    func testFittedModeLocksNotesToHalfAndRestoresPreviousSize() {
        let entered = DashboardViewModePresentation.notesTransition(
            from: .stacked,
            to: .fitted,
            current: .eighty,
            lastExpanded: .half
        )
        XCTAssertEqual(entered.current, .half)
        XCTAssertEqual(entered.lastExpanded, .eighty)
        XCTAssertEqual(
            DashboardViewModePresentation.notesHeight(
                in: 800,
                mode: .fitted,
                size: entered.current
            ),
            400
        )

        let exited = DashboardViewModePresentation.notesTransition(
            from: .fitted,
            to: .tiles,
            current: entered.current,
            lastExpanded: entered.lastExpanded
        )
        XCTAssertEqual(exited.current, .eighty)
        XCTAssertEqual(exited.lastExpanded, .eighty)
    }

    func testOverviewTransitionPreservesSizeRememberedBeforeFittedMode() {
        let transition = DashboardViewModePresentation.notesTransition(
            from: .fitted,
            to: .overview,
            current: .half,
            lastExpanded: .eighty
        )

        XCTAssertEqual(transition.current, .minimized)
        XCTAssertEqual(transition.lastExpanded, .eighty)
    }
}
