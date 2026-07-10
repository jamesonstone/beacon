---
kit_metadata_version: 1
artifact: spec
workflow_version: 2
phase: deliver
delivery_intent: issue_branch_pr_in_progress
clarification:
  status: ready
  confidence: 100
  unresolved_questions: 0
feature:
  id: "0001"
  slug: beacon-v1
  dir: 0001-beacon-v1
references:
  - id: issue-1
    name: Implement Beacon work-lane review radar
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/1
    relation: supports
    read_policy: conditional
    used_for: delivery
    status: active
relationships: []
skills: []
---

# Beacon v1

## Thesis

Beacon is a read-only signal layer that identifies which agent-driven Git work lanes are ready for review and which need action. A lane is a linked worktree and branch, optionally correlated with an open GitHub pull request.

## Context

Repository-level status loses information when multiple coding-agent threads use worktrees in the same repository. Beacon therefore scans worktrees and pull requests separately, correlates them by repository and head branch, and preserves independent local, publication, CI, review, merge, and freshness signals.

The Go CLI is the source of truth. The macOS menu application is a native SwiftUI viewer over the CLI's versioned JSON output.

## Clarifications

- Progress is inferred from Git and GitHub evidence only; Codex task metadata is out of scope.
- All linked worktrees are scanned.
- Open pull requests authored by the authenticated GitHub user are included even when no matching local worktree exists.
- Pending or absent CI does not block human code review; failing or unknown CI does.
- Beacon may run a timeout-bounded fetch to refresh Git metadata.
- The default configuration path is `$HOME/.config/beacon/config.yaml`.
- The macOS app targets macOS 14 or later and bundles the same Go executable that is also distributed as the standalone CLI.
- V1 is a local developer build without signing, notarization, App Store packaging, notifications, or launch-at-login.

Clarification is complete. No unresolved assumptions remain for implementation.

## Requirements

1. Load a strict, versioned YAML configuration from the explicit flag, `BEACON_CONFIG`, or `$HOME/.config/beacon/config.yaml` in that order.
2. Discover all linked worktrees using stable NUL-delimited Git porcelain output.
3. Distinguish dirty, conflicted, unpublished, unpushed, behind, diverged, and published local states.
4. Query open GitHub pull requests through authenticated `gh` and normalize draft, CI, review, merge, and freshness state.
5. Correlate local and remote work by repository, branch, and head OID when available while retaining remote-only PRs.
6. Emit one stable JSON schema and a compact grouped terminal view.
7. Provide `scan`, `doctor`, `open`, `open-next`, `config`, and `version` commands.
8. Continue scanning healthy repositories when another repository fails.
9. Provide a native macOS 14+ `MenuBarExtra` application that polls and renders the bundled CLI.
10. Never edit working files, change branches, push, create PRs, or merge as part of scanning.

## Assumptions

- GitHub is the only remote provider in v1.
- `git` and authenticated `gh` are installed on the user's Mac.
- Configured repository paths exist and are Git repositories.
- Active local work means checked-out worktrees; unattached local branches are not enumerated.
- A metadata-only fetch is permitted and occurs no more frequently than the configured refresh interval.

## Acceptance Criteria

- [x] AC1: Strict config parsing applies documented precedence, defaults, validation, and `~` expansion.
- [x] AC2: Two linked worktrees in one repository produce two lanes.
- [x] AC3: Dirty, conflicted, unpushed, unpublished, behind, diverged, and published states are distinguished.
- [x] AC4: Open PRs normalize draft, CI, review, merge, and freshness signals.
- [x] AC5: Remote-only pull requests remain visible and branch/OID correlation is deterministic.
- [x] AC6: Pending/no CI may be review-ready; failed/unknown CI, conflicts, local unpublished work, and requested changes block readiness.
- [x] AC7: One repository failure produces an error without suppressing other lanes.
- [x] AC8: Human and JSON output contain the same ordered lanes and stable schema version 1.
- [x] AC9: All documented CLI commands behave with documented exit codes and stdout/stderr separation.
- [x] AC10: The macOS app decodes CLI JSON, prevents overlapping scans, retains the last good snapshot, and opens PR/worktree/config targets.
- [x] AC11: Go formatting, vet, unit/race tests, build, CLI smoke checks, and Xcode build/tests pass.
- [x] AC12: Inspection confirms no scanner path mutates tracked files, branches, commits, PRs, reviews, or merges.

## Implementation Plan

1. Define config and schema-v1 domain contracts, command runner, Git/GitHub scanners, lane policy, and output renderers in Go.
2. Wire the Cobra CLI commands and explicit exit behavior around the shared scanner.
3. Add table-driven parser/policy tests and temporary-repository integration tests.
4. Build the SwiftUI menu application with Codable schema models and an injectable process client.
5. Embed the Go helper through the Xcode build, then add docs and CI.
6. Run validation, self-review the complete diff, and deliver through issue `#1` and branch `GH-1`.

## Task Checklist

- [x] T1: Create the issue-backed delivery lane.
- [x] T2: Implement config, domain models, command runner, local scanner, GitHub scanner, policy, and outputs.
- [x] T3: Wire the CLI command surface.
- [x] T4: Complete Go unit and integration coverage.
- [x] T5: Implement and test the macOS menu application and bundled helper build.
- [x] T6: Add README guidance, CI, and examples.
- [x] T7: Run the validation map and record evidence.
- [x] T8: Reflect, self-review, and deliver the ready pull request.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1 | Config unit tests and `beacon config validate` |
| AC2-AC3 | Git parser tests and temporary multi-worktree integration test |
| AC4-AC6 | GitHub fixture and policy table tests |
| AC7 | Scanner partial-failure test |
| AC8 | JSON decoding and terminal golden tests |
| AC9 | CLI command tests plus doctor/scan smoke runs |
| AC10 | Swift model/state tests and manual menu smoke check |
| AC11 | `make fmt-check vet test test-race build macos-test` |
| AC12 | Diff review and command inventory inspection |

## Reflection Notes

- Modeling a lane rather than a repository preserves multiple worktrees and remote-only pull requests without collapsing simultaneous evidence.
- Readiness remains conservative when CI or merge state is unknown, while explicitly allowing pending or absent CI for human review.
- The macOS application and bundled helper require distinct executable names on case-insensitive filesystems; the helper is `beacon-cli`.
- The app and helper are both universal binaries even though v1 distribution remains a local developer workflow.
- macOS 14 is the minimum deployment target because the menu uses `ContentUnavailableView`.

## Documentation Updates

- Expand `README.md` with install, configuration, command, architecture, read-only boundary, and troubleshooting guidance.
- Keep the example configuration aligned with schema version 1.

## Delivery Decision

Deliver the complete v1 implementation as a ready-for-review pull request from `GH-1`, assigned to `jamesonstone`, with `Closes #1`.

## Evidence

- `make fmt-check` passed.
- `make vet` passed.
- `make test` passed, including a real two-worktree repository integration test and partial-repository-failure coverage.
- `make test-race` passed.
- `make build` produced the standalone `bin/beacon` executable.
- `beacon config validate`, `beacon doctor`, terminal scan, JSON scan, repository filtering, usage exit code 2, and `open-next` all passed against the default config.
- `make macos-test` passed 6 Swift tests with no failures.
- `file` confirmed the bundled `beacon-cli` contains arm64 and x86_64 slices.
- The final `Beacon.app` process remained alive for more than 30 seconds with no stderr output.
- While the app was running, its bundled helper returned `doctor.ok=true` and a schema-v1 scan containing a lane with zero errors.
- Ready pull request https://github.com/jamesonstone/beacon/pull/2 was created from `GH-1` and assigned to `jamesonstone`.
- GitHub CI passed: `go` in 40 seconds and `macos` in 1 minute 41 seconds.
