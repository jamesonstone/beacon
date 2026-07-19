import XCTest
@testable import Beacon

extension ModelsTests {
    func testUpToDateBacksplashRequiresNoWorkAndNoLoadingProjects() {
        XCTAssertTrue(UpToDatePresentation.shouldShow(inProgressCount: 0, loadingProjectCount: 0))
        XCTAssertFalse(UpToDatePresentation.shouldShow(inProgressCount: 1, loadingProjectCount: 0))
        XCTAssertFalse(UpToDatePresentation.shouldShow(inProgressCount: 0, loadingProjectCount: 1))
    }

    func testStackedDashboardPrioritizesProjectNameOverLaneTitle() {
        XCTAssertEqual(DashboardLanePresentation.projectNameSize, 15)
        XCTAssertEqual(DashboardLanePresentation.laneTitleSize, 13)
        XCTAssertGreaterThan(
            DashboardLanePresentation.projectNameSize,
            DashboardLanePresentation.laneTitleSize
        )
    }

    func testDashboardLaneIdentitiesUseDistinctPaletteAccents() {
        let local = TestSnapshots.lane(worktree: TestSnapshots.worktree)
        let issue = TestSnapshots.lane(issue: TestSnapshots.issue)
        let pullRequest = TestSnapshots.lane(
            pullRequest: TestSnapshots.pullRequest,
            issue: TestSnapshots.issue,
            worktree: TestSnapshots.worktree
        )

        XCTAssertEqual(DashboardLanePresentation.identity(for: local), .local)
        XCTAssertEqual(DashboardLanePresentation.identity(for: issue), .issue)
        XCTAssertEqual(DashboardLanePresentation.identity(for: pullRequest), .pullRequest)
        XCTAssertEqual(
            Set(DashboardLaneIdentity.allCases.map(\.accent)).count,
            DashboardLaneIdentity.allCases.count
        )
    }

    func testIgnoreActionAppearsOnlyOnFollowingCards() {
        XCTAssertTrue(DashboardLanePresentation.showsIgnoreAction(in: .following))
        XCTAssertFalse(DashboardLanePresentation.showsIgnoreAction(in: .parking))
        XCTAssertFalse(DashboardLanePresentation.showsIgnoreAction(in: .recent))
        XCTAssertFalse(DashboardLanePresentation.showsIgnoreAction(in: .quiet))
    }

    func testMergedCheckoutWarningUsesConfirmedKindAndSeverity() {
        var lane = TestSnapshots.lane(worktree: TestSnapshots.worktree)
        XCTAssertFalse(DashboardLanePresentation.showsCheckoutWarning(for: lane))

        lane.checkoutWarning = CheckoutWarningDetails(
            kind: "merged_remote_branch_deleted",
            severity: "warning",
            pullRequestNumber: 32,
            pullRequestURL: "https://github.com/owner/repo/pull/32",
            branch: "GH-31",
            base: "main",
            mergedAt: "2026-07-15T15:00:00Z",
            confirmedAt: "2026-07-15T16:00:00Z",
            message: "Merged branch remains checked out."
        )
        XCTAssertTrue(DashboardLanePresentation.showsCheckoutWarning(for: lane))
        XCTAssertFalse(DashboardLanePresentation.checkoutWarningIsCritical(for: lane))

        lane.checkoutWarning = CheckoutWarningDetails(
            kind: "merged_remote_branch_deleted",
            severity: "critical",
            pullRequestNumber: 32,
            pullRequestURL: nil,
            branch: "GH-31",
            base: "main",
            mergedAt: "2026-07-15T15:00:00Z",
            confirmedAt: "2026-07-15T16:00:00Z",
            message: "Local commits require review."
        )
        XCTAssertTrue(DashboardLanePresentation.checkoutWarningIsCritical(for: lane))
    }

    func testDecodesCompleteSchemaVersionThree() throws {
        let data = Data(Self.snapshotJSON.utf8)
        let snapshot = try JSONDecoder().decode(BeaconSnapshot.self, from: data)
        XCTAssertEqual(snapshot.schemaVersion, 3)
        XCTAssertEqual(snapshot.projects.first?.progress?.phase, "deliver")
        XCTAssertEqual(snapshot.summary.reviewReady, 1)
        XCTAssertEqual(snapshot.summary.trackedProjects, 1)
        XCTAssertEqual(snapshot.summary.untrackedProjects, 0)
        XCTAssertEqual(snapshot.summary.followingProjects, 1)
        XCTAssertEqual(snapshot.summary.recentProjects, 0)
        XCTAssertEqual(snapshot.summary.quietProjects, 0)
        XCTAssertEqual(snapshot.summary.openIssues, 1)
        XCTAssertEqual(snapshot.tracking?.path, "/Users/test/.config/beacon/tracking.yaml")
        XCTAssertEqual(snapshot.projects.first?.trackingState, "tracked")
        XCTAssertEqual(snapshot.projects.first?.followState, "following")
        XCTAssertEqual(snapshot.lanes.first?.pullRequest?.number, 42)
        XCTAssertEqual(snapshot.lanes.first?.pullRequest?.checks.success, 2)
        XCTAssertEqual(snapshot.lanes.first?.pullRequest?.feedback.unresolvedThreads, 1)
        XCTAssertEqual(snapshot.lanes.first?.issue?.number, 41)
        XCTAssertEqual(snapshot.lanes.first?.signals.issue, "open")
        XCTAssertEqual(snapshot.groups.ready, ["gh:owner/repo#42"])
        XCTAssertEqual(snapshot.workingSet?.active, ["gh:owner/repo#42"])
        XCTAssertEqual(snapshot.lanes.first?.attention?.delta, "CI changed from pending to success")
        XCTAssertEqual(snapshot.lanes.first?.attention?.tags, ["manual test", "release"])
        XCTAssertEqual(snapshot.lanes.first?.checkoutWarning?.pullRequestNumber, 42)
        XCTAssertEqual(snapshot.lanes.first?.checkoutWarning?.severity, "warning")
    }

    func testDashboardViewModesHaveStablePresentationContracts() {
        XCTAssertEqual(DashboardViewMode.allCases.map(\.rawValue), ["stacked", "tiles", "kanban", "overview"])
        XCTAssertEqual(DashboardViewMode.stacked.title, "Stacked")
        XCTAssertEqual(DashboardViewMode.tiles.symbol, "rectangle.grid.1x2")
        XCTAssertTrue(DashboardViewMode.kanban.title.contains("Experimental"))
        XCTAssertTrue(DashboardViewMode.overview.title.contains("Experimental"))
        XCTAssertEqual(DashboardViewMode.overview.symbol, "rectangle.grid.2x2")
    }

    func testViewModeMenuIdentityIgnoresUnrelatedParentRefreshes() {
        let first = DashboardViewModeMenu(
            mode: .stacked,
            themeID: BeaconThemePreference.defaultID.rawValue,
            increasedContrast: false,
            select: { _ in }
        )
        let refreshedParent = DashboardViewModeMenu(
            mode: .stacked,
            themeID: BeaconThemePreference.defaultID.rawValue,
            increasedContrast: false,
            select: { _ in }
        )
        let changedMode = DashboardViewModeMenu(
            mode: .tiles,
            themeID: BeaconThemePreference.defaultID.rawValue,
            increasedContrast: false,
            select: { _ in }
        )

        XCTAssertEqual(first, refreshedParent)
        XCTAssertNotEqual(first, changedMode)
    }

    func testTaxonomyInfoStaysOpenAcrossTriggerAndPopoverTraversal() {
        XCTAssertFalse(TaxonomyInfoPresentation.shouldDismiss(
            isPinned: false,
            triggerHovered: true,
            popoverHovered: false
        ))
        XCTAssertFalse(TaxonomyInfoPresentation.shouldDismiss(
            isPinned: false,
            triggerHovered: false,
            popoverHovered: true
        ))
        XCTAssertFalse(TaxonomyInfoPresentation.shouldDismiss(
            isPinned: true,
            triggerHovered: false,
            popoverHovered: false
        ))
        XCTAssertTrue(TaxonomyInfoPresentation.shouldDismiss(
            isPinned: false,
            triggerHovered: false,
            popoverHovered: false
        ))
    }

    func testDashboardDensitiesHaveStablePersistentIdentifiers() {
        XCTAssertEqual(DashboardDensity.storageKey, "beacon.dashboard.density")
        XCTAssertEqual(DashboardDensity.defaultDensity, .comfortable)
        XCTAssertEqual(DashboardDensity.allCases.map(\.rawValue), ["comfortable", "compact", "dense"])
        XCTAssertLessThan(DashboardDensity.dense.cardPadding, DashboardDensity.compact.cardPadding)
        XCTAssertLessThan(DashboardDensity.compact.cardPadding, DashboardDensity.comfortable.cardPadding)
    }

    func testOverviewMinimizesAndRestoresPriorNotesSize() {
        let entered = DashboardOverviewPresentation.notesTransition(
            from: .stacked,
            to: .overview,
            current: .eighty,
            lastExpanded: .half
        )
        XCTAssertEqual(entered.current, .minimized)
        XCTAssertEqual(entered.lastExpanded, .eighty)

        let exited = DashboardOverviewPresentation.notesTransition(
            from: .overview,
            to: .tiles,
            current: entered.current,
            lastExpanded: entered.lastExpanded
        )
        XCTAssertEqual(exited.current, .eighty)
        XCTAssertEqual(exited.lastExpanded, .eighty)
    }

    func testDashboardTabsKeepFollowingAsTheStableDefault() {
        XCTAssertEqual(DashboardTab.defaultTab, .following)
        XCTAssertEqual(DashboardTab.allCases.map(\.rawValue), ["following", "parking", "recent", "quiet"])
        XCTAssertEqual(DashboardTab.allCases[1], .parking)
        XCTAssertEqual(DashboardTab.parking.title, "Parking Lot")
        XCTAssertEqual(DashboardTab.recent.title, "Recently Updated")
        XCTAssertEqual(DashboardTab.quiet.symbol, "moon.stars.fill")
    }

    func testDashboardDestinationsReturnToFollowingWhenSelectedAgain() {
        let destinations: [DashboardDestination] = [
            .tab(.parking),
            .tab(.recent),
            .tab(.quiet),
            .projectInventory,
            .repositorySync,
            .dependencyLimits,
        ]

        for destination in destinations {
            XCTAssertEqual(DashboardDestination.following.toggled(selecting: destination), destination)
            XCTAssertEqual(destination.toggled(selecting: destination), .following)
        }
    }

    func testDashboardDestinationSwitchesDirectlyToDifferentSelection() {
        XCTAssertEqual(
            DashboardDestination.repositorySync.toggled(selecting: .dependencyLimits),
            .dependencyLimits
        )
    }

    func testDecodesRepositorySyncProtocolReport() throws {
        let data = Data(Self.repositorySyncEventJSON.utf8)
        let event = try JSONDecoder().decode(AgentEvent.self, from: data)
        let report = try XCTUnwrap(event.repositorySync)
        XCTAssertFalse(report.fetchAttempted)
        XCTAssertEqual(report.repositories.first?.projectID, "owner/repo")
        XCTAssertEqual(report.repositories.first?.currentBehind, 2)
        XCTAssertTrue(report.repositories.first?.canUpdate == true)
    }

    func testDependencyLimitPercentagesAndThresholds() throws {
        let data = Data(#"{"checked_at":"2026-07-14T12:30:00Z","dependencies":[{"name":"gh","buckets":[{"id":"graphql","name":"GraphQL","limit":5000,"used":1,"remaining":4999,"reset_at":"2026-07-14T13:00:00Z"},{"id":"core","name":"REST Core","limit":5000,"used":2500,"remaining":2500,"reset_at":"2026-07-14T13:00:00Z"},{"id":"search","name":"Search","limit":30,"used":23,"remaining":7,"reset_at":"2026-07-14T13:00:00Z"}]}]}"#.utf8)
        let report = try JSONDecoder().decode(DependencyLimitReport.self, from: data)

        XCTAssertEqual(report.dependencies.first?.name, "gh")
        XCTAssertEqual(report.dependencies.first?.buckets.map(\.usagePercent), [1, 50, 77])
        XCTAssertEqual(report.highestUsagePercent, 77)
        XCTAssertEqual(report.usageLevel, .critical)
        XCTAssertEqual(DependencyLimitPresentation.level(percent: 49, hasUsage: true), .healthy)
        XCTAssertEqual(DependencyLimitPresentation.level(percent: 50, hasUsage: true), .warning)
        XCTAssertEqual(DependencyLimitPresentation.level(percent: 75, hasUsage: true), .warning)
        XCTAssertEqual(DependencyLimitPresentation.level(percent: 76, hasUsage: true), .critical)
        XCTAssertEqual(DependencyLimitPresentation.level(percent: 0, hasUsage: false), .unmeasured)
    }
}
