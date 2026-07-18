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

    func testUpToDateBacksplashRequiresNoWorkAndNoLoadingProjects() {
        XCTAssertTrue(UpToDatePresentation.shouldShow(inProgressCount: 0, loadingProjectCount: 0))
        XCTAssertFalse(UpToDatePresentation.shouldShow(inProgressCount: 1, loadingProjectCount: 0))
        XCTAssertFalse(UpToDatePresentation.shouldShow(inProgressCount: 0, loadingProjectCount: 1))
    }

    func testDashboardTypographyUsesSelectableSystemDesignsAndTwelvePointDefault() {
        XCTAssertEqual(BeaconFontFamily.allCases.map(\.rawValue), ["system", "rounded", "monospaced", "serif"])
        XCTAssertEqual(BeaconTypography.defaultBaseSize, 12)
        XCTAssertEqual(BeaconTypography.resolvedSize(10, baseSize: 12), 12)
        XCTAssertEqual(BeaconTypography.resolvedSize(17, baseSize: 14), 21)
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
