import SwiftUI

extension MenuView {
    @ViewBuilder
    func laneDetailPanel(_ lane: WorkLane) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 3) {
                    Label(workItemTitle(lane), systemImage: laneIdentitySymbol(lane))
                        .font(BeaconTypography.semibold(14))
                    Text(laneReference(lane))
                        .font(BeaconTypography.medium(10))
                        .foregroundStyle(DashboardLanePresentation.identity(for: lane).accent.color)
                }
                Spacer()
                if let target = AppState.openTarget(for: lane) {
                    Link("Open", destination: target)
                }
            }

            if let issue = lane.issue {
                detailMarkdown(issue.body, empty: "This issue has no description.", truncated: issue.bodyTruncated == true)
                detailMetadata("Assignees", values: issue.assignees)
                detailMetadata("Labels", values: issue.labels)
            } else if let pullRequest = lane.pullRequest {
                detailMarkdown(pullRequest.body, empty: "This pull request has no description.", truncated: pullRequest.bodyTruncated == true)
                Label("\(pullRequest.headRefName) → \(pullRequest.baseRefName)", systemImage: "arrow.triangle.pull")
                    .font(BeaconTypography.medium(10))
                    .foregroundStyle(BeaconPalette.cyan)
                if !pullRequest.closingIssues.isEmpty {
                    VStack(alignment: .leading, spacing: 5) {
                        Text("Linked issues").font(BeaconTypography.semibold(11))
                        ForEach(pullRequest.closingIssues, id: \.number) { issue in
                            if let url = URL(string: issue.url) {
                                Link("Issue #\(issue.number) · \(issue.title)", destination: url)
                                    .font(BeaconTypography.medium(10))
                            }
                        }
                    }
                }
            } else {
                VStack(alignment: .leading, spacing: 5) {
                    Label(lane.repository, systemImage: "folder")
                    if !lane.branch.isEmpty { Label(lane.branch, systemImage: "arrow.triangle.branch") }
                    if let path = lane.worktree?.path { Label(path, systemImage: "internaldrive") }
                }
                .font(BeaconTypography.medium(10))
                .foregroundStyle(BeaconPalette.cyan)
            }

            Divider()
            Label(actionLabel(lane.nextAction), systemImage: "arrow.right.circle.fill")
                .font(BeaconTypography.semibold(11))
                .foregroundStyle(DashboardLanePresentation.identity(for: lane).accent.color)
            detailList("Why Beacon chose this", symbol: "lightbulb", values: lane.reasons)
            detailList("Warnings", symbol: "exclamationmark.triangle", values: lane.warnings)
            detailList("Blockers", symbol: "hand.raised.fill", values: lane.blockers)
            if let attention = lane.attention {
                Text("\(attention.delta) · updated \(timeSinceActivity(lane.updatedAt))")
                    .font(BeaconTypography.regular(9))
                    .foregroundStyle(BeaconPalette.lavender)
                detailMetadata("Local tags", values: attention.tags ?? [])
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .textSelection(.enabled)
    }

    @ViewBuilder
    func reviewFeedbackPanel(_ lane: WorkLane, feedback: FeedbackSummary) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Label(
                "PR #\(lane.pullRequest?.number ?? 0) · \(feedback.unresolvedThreads) unresolved review \(feedback.unresolvedThreads == 1 ? "thread" : "threads")",
                systemImage: "text.bubble.fill"
            )
            .font(BeaconTypography.semibold(13))
            .foregroundStyle(BeaconPalette.pink)

            if feedback.threadsTruncated == true {
                Label("GitHub returned only the first page of review threads.", systemImage: "ellipsis.circle")
                    .font(BeaconTypography.medium(10))
                    .foregroundStyle(BeaconPalette.gold)
            }

            let threads = feedback.threads ?? []
            if threads.isEmpty {
                Text("Detailed comments are unavailable in this cached snapshot. Refresh Beacon to load author, file, body, timestamp, and direct links.")
                    .font(BeaconTypography.regular(10))
                    .foregroundStyle(BeaconPalette.lavender)
            } else {
                ForEach(threads) { thread in
                    VStack(alignment: .leading, spacing: 7) {
                        HStack {
                            Label(reviewThreadLocation(thread), systemImage: "doc.text.magnifyingglass")
                                .font(BeaconTypography.semibold(10))
                            if thread.outdated {
                                Text("Outdated")
                                    .font(BeaconTypography.medium(8))
                                    .foregroundStyle(BeaconPalette.gold)
                            }
                            Spacer()
                        }
                        ForEach(thread.comments) { comment in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text(comment.author.isEmpty ? "Unknown reviewer" : "@\(comment.author)")
                                        .font(BeaconTypography.semibold(10))
                                    Spacer()
                                    Text(timeSinceActivity(comment.updatedAt))
                                        .font(BeaconTypography.regular(8))
                                        .foregroundStyle(BeaconPalette.lavender)
                                }
                                markdownText(comment.body)
                                    .font(BeaconTypography.regular(10))
                                HStack {
                                    if comment.bodyTruncated {
                                        Label("Comment truncated", systemImage: "ellipsis.circle")
                                            .foregroundStyle(BeaconPalette.gold)
                                    }
                                    Spacer()
                                    if let url = URL(string: comment.url) {
                                        Link("Open feedback", destination: url)
                                    }
                                }
                                .font(BeaconTypography.medium(9))
                            }
                            .padding(8)
                            .background(BeaconPalette.softGradient(BeaconPalette.pink), in: RoundedRectangle(cornerRadius: 7))
                        }
                        if thread.commentsTruncated {
                            Label("Additional comments are available on GitHub.", systemImage: "ellipsis.bubble")
                                .font(BeaconTypography.medium(9))
                                .foregroundStyle(BeaconPalette.gold)
                        }
                    }
                    .padding(9)
                    .overlay {
                        RoundedRectangle(cornerRadius: 8)
                            .strokeBorder(BeaconPalette.pink.opacity(0.3), lineWidth: 0.7)
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .textSelection(.enabled)
    }

    @ViewBuilder
    private func detailMarkdown(_ body: String?, empty: String, truncated: Bool) -> some View {
        let value = body?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        VStack(alignment: .leading, spacing: 5) {
            if value.isEmpty {
                Text(empty).foregroundStyle(BeaconPalette.lavender)
            } else {
                markdownText(value)
            }
            if truncated {
                Label("Description truncated; open on GitHub for the complete text.", systemImage: "ellipsis.circle")
                    .font(BeaconTypography.medium(9))
                    .foregroundStyle(BeaconPalette.gold)
            }
        }
        .font(BeaconTypography.regular(10))
    }

    func markdownText(_ value: String) -> Text {
        guard let attributed = try? AttributedString(markdown: value) else { return Text(value) }
        return Text(attributed)
    }

    @ViewBuilder
    private func detailMetadata(_ title: String, values: [String]) -> some View {
        if !values.isEmpty {
            Text("\(title): \(values.joined(separator: ", "))")
                .font(BeaconTypography.regular(9))
                .foregroundStyle(BeaconPalette.lavender)
        }
    }

    @ViewBuilder
    private func detailList(_ title: String, symbol: String, values: [String]) -> some View {
        if !values.isEmpty {
            VStack(alignment: .leading, spacing: 3) {
                Label(title, systemImage: symbol).font(BeaconTypography.semibold(10))
                ForEach(values, id: \.self) { value in
                    Text("• \(value)").font(BeaconTypography.regular(9))
                }
            }
        }
    }

    private func laneIdentitySymbol(_ lane: WorkLane) -> String {
        switch DashboardLanePresentation.identity(for: lane) {
        case .local: "internaldrive"
        case .pullRequest: "arrow.triangle.pull"
        case .issue: "smallcircle.filled.circle"
        }
    }

    private func laneReference(_ lane: WorkLane) -> String {
        if let pullRequest = lane.pullRequest { return "\(lane.github) · PR #\(pullRequest.number)" }
        if let issue = lane.issue { return "\(lane.github) · Issue #\(issue.number)" }
        return [lane.repository, lane.branch].filter { !$0.isEmpty }.joined(separator: " · ")
    }

    private func reviewThreadLocation(_ thread: ReviewThreadDetails) -> String {
        guard let line = thread.displayLine else { return thread.path.isEmpty ? "General feedback" : thread.path }
        return "\(thread.path):\(line)"
    }
}
