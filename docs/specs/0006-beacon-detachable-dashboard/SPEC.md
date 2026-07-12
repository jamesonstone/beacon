---
kit_metadata_version: 1
artifact: spec
workflow_version: 2
phase: deliver
delivery_intent: existing_ready_pull_request
clarification:
  status: ready
  confidence: 100
  unresolved_questions: 0
feature:
  id: "0006"
  slug: beacon-detachable-dashboard
  dir: 0006-beacon-detachable-dashboard
relationships:
  - type: builds_on
    target: 0005-beacon-background-agent
references:
  - id: issue-3
    name: Add managed project tracking and automatic reactivation
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/3
    relation: implements
    read_policy: must
    used_for: user-selected existing delivery lane
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: shared evidence, client boundaries, and macOS behavior
    status: active
skills: []
---

# Beacon Detachable macOS Dashboard

## Thesis

Beacon should remain immediately available from its menu-bar extra while also
behaving like a regular macOS application with a compact dashboard that users
can reach through the Dock and Command-Tab when the menu-bar item is obscured.

## Context

The existing `LSUIElement` menu application is intentionally compact, but a
camera notch or crowded menu bar can make its only entry point inaccessible.
The Swift client already owns a shared `AppState` and complete dashboard view,
so the smallest durable change is a second presentation surface over the same
state rather than another scanner, schema, or policy implementation.

## Clarifications

- Beacon becomes a regular macOS application and remains visible in the Dock
  and Command-Tab while running.
- Ordinary launches open one compact dashboard window. Closing that window
  leaves the menu-bar extra, shared state, and background-agent connection
  running.
- Dock reopening, Command-Tab activation with no visible dashboard, and an
  explicit menu action reopen the same singleton window.
- The menu and detached window render the same SwiftUI dashboard over one
  `AppState`; they do not start separate subscriptions or scans.
- The menu remains 430 by 540 points. The detached window defaults to 480 by
  620 points and cannot resize below the menu dimensions.
- `Open Beacon at Login` is off by default and uses an embedded Service
  Management login item. Login launches are quiet until the user explicitly
  activates Beacon.
- The login helper passes `--login`, opens the containing application without
  activation, and exits. It performs no scanning or repository access.
- Beacon retains the macOS 14 minimum and its developer-local unsandboxed
  boundary.
- The application icon uses the existing cyan, lavender, pink, and dark-space
  visual language with an original lighthouse/radar mark.

## Requirements

1. Remove agent-only application presentation and make Beacon visible in the
   Dock and Command-Tab without removing `MenuBarExtra`.
2. Add one singleton detachable window hosting the same dashboard and shared
   `AppState` as the menu surface.
3. Open the dashboard on ordinary launch and user activation, keep the process
   alive after window close, and suppress the window for login launches.
4. Add an `Open Dashboard` action to the menu and retain existing scan, open,
   configuration, project-management, and quit actions.
5. Add an embedded `BeaconLoginItem.app`, register it with `SMAppService`, and
   expose registration, approval, and failure state in the shared UI.
6. Add a complete macOS application-icon asset catalog and package it in local
   and release builds.
7. Preserve the Go authority, snapshot schema v2, read-only scanner boundary,
   background-agent protocol, and deterministic evidence policy unchanged.
8. Preserve healthy state when window or login-item operations fail and avoid
   duplicate agent subscriptions or overlapping scans.

## Assumptions

- Users accept a persistent Dock icon as the supported consequence of regular
  Command-Tab presence.
- The embedded login helper and main app are signed together in the containing
  application bundle.
- Login-item approval may require the user to visit System Settings; Beacon
  reports this state but does not bypass macOS authorization.
- Closing the window is not equivalent to quitting Beacon; Quit remains an
  explicit menu action.

## Acceptance Criteria

- [x] AC1: Beacon appears in the Dock and Command-Tab and retains its existing
  menu-bar extra.
- [x] AC2: An ordinary launch opens exactly one 480 by 620 dashboard window and
  closing it leaves Beacon running.
- [x] AC3: Menu, Dock, and Command-Tab activation reopen the singleton window
  without duplicating subscriptions or scans.
- [x] AC4: Menu and window surfaces show identical state and retain existing
  dashboard, project-tracking, and open-item behavior.
- [x] AC5: The login toggle registers and unregisters the embedded helper,
  reports approval/errors, and defaults to off.
- [x] AC6: A helper-driven `--login` launch starts Beacon quietly; later user
  activation opens the dashboard.
- [x] AC7: Local and release app bundles contain the Go helper, login helper,
  neon-space app icon, correct bundle identifiers, and no `LSUIElement=true`.
- [x] AC8: Existing Go, schema, agent, CLI, Swift, packaging, and read-only
  behavior remain unchanged and all required validation passes.

## Implementation Plan

1. Stabilize the existing loader and race-test edits and isolate CLI tests from
   the user's live background-agent socket.
2. Extract a reusable dashboard surface and add regular-app lifecycle plus a
   singleton SwiftUI-hosting window over the shared state.
3. Add login-item state management, the embedded helper target, and quiet-login
   lifecycle handling.
4. Add the application icon, Xcode embedding/signing settings, and package
   assertions.
5. Update project documentation, run full validation, and deliver focused
   commits to existing branch `GH-3` and ready PR #4.

## Task Checklist

- [x] T1: Complete and validate pending loader/race-test work.
- [x] T2: Implement and test the shared dashboard and singleton window.
- [x] T3: Implement and test login registration, helper launch, and lifecycle.
- [x] T4: Add and verify icon, Xcode targets, embedding, and packaging.
- [x] T5: Update constitution, progress summary, README, and release evidence.
- [x] T6: Run complete validation and update PR #4.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC4 | Swift lifecycle/state tests plus manual menu, Dock, Command-Tab, close, and reopen checks |
| AC5-AC6 | injectable login-service tests, helper argument/path tests, and isolated login-launch smoke |
| AC7 | Xcode Debug/Release bundle inspection, icon/helper assertions, and codesign verification |
| AC8 | Go test/race/build, Xcode test/build, release packaging, Kit check, and diff hygiene |

## Reflection Notes

One shared state object and an AppKit-owned singleton hosting window kept the
SwiftUI surfaces simple while making macOS 14 launch, close, and reopen behavior
explicit. A separate login helper was necessary to distinguish quiet login
startup from ordinary launches without weakening the regular-app Dock boundary.

## Documentation Updates

- Update `docs/CONSTITUTION.md` for the regular-app, shared-dashboard, and login
  helper boundaries.
- Update `docs/PROJECT_PROGRESS_SUMMARY.md` for feature 0006.
- Update `README.md` with detachable-window and login-item usage.

## Delivery Decision

Per the user's explicit choice, deliver this distinct feature on issue #3,
branch `GH-3`, and ready PR #4 rather than creating a new lane.

## Evidence

- `make fmt-check vet test test-race build release-test` passed.
- `make macos-test` passed 31 Swift tests after the queued tracking addition.
- `kit check --all` passed all seven feature specifications.
- Debug and Release bundles contained universal Go and login helpers,
  `AppIcon.icns`, the expected helper bundle identifier, and no `LSUIElement`.
- Release packaging completed with ad-hoc signature verification and all CLI,
  macOS, and checksum artifacts.
- A normal Release application launch created the expected regular Beacon
  process; lifecycle tests cover singleton reopen and quiet login launch.
