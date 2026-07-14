---
kit_metadata_version: 1
artifact: spec
workflow_version: 2
phase: deliver
delivery_intent: ready_pull_request
clarification:
  status: ready
  confidence: 100
  unresolved_questions: 0
feature:
  id: "0015"
  slug: notes-agent-lifecycle
  dir: 0015-notes-agent-lifecycle
relationships:
  - type: builds_on
    target: 0005-beacon-background-agent
  - type: builds_on
    target: 0013-signal-note-tabs
references:
  - id: issue-25
    name: Restore Signal Notes editing and bound agent lifetime
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/25
    relation: implements
    read_policy: must
    used_for: user-approved delivery lane and acceptance contract
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: native macOS editing, process ownership, and agent authority
    status: active
skills:
  - name: figma:figma-swiftui
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/figma/2.0.14/skills/figma-swiftui/SKILL.md
    trigger: native SwiftUI and AppKit editor lifecycle repair
    required: true
---

# Notes and Agent Lifecycle

## Thesis

Beacon should keep Signal Notes directly editable and run its background agent
only while a user has activated Beacon through the CLI or macOS application.
Explicit focus transitions and one idempotent Go lifecycle authority prevent a
stale SwiftUI update or launchd's `KeepAlive` policy from outliving user intent.

## Context

Feature 0013 replaced the plain Signal Notes editor with a shared tabbed
workspace. The AppKit text view still mirrors SwiftUI focus state, but its
update pass currently resigns first responder whenever the binding is false.
A user click can therefore be revoked before AppKit reports that editing began.

Feature 0005 installs a user LaunchAgent with `KeepAlive`. The macOS termination
hook currently cancels only the Swift subscription, leaving launchd and the Go
agent running. Beacon already owns a safe `agent stop` boundary, but the app
does not invoke it and restart behavior is limited to explicit installation.

## Clarifications

- General and detail notes remain editable in both macOS surfaces.
- Programmatic blur still works, but an initial user focus attempt is never
  treated as a request to resign first responder.
- Closing the dashboard window alone does not quit Beacon or stop the agent;
  quitting the application does.
- Normal app termination synchronously unloads the LaunchAgent before the app
  process exits and also stops a directly served agent when present.
- App launch starts or reconnects to one agent without duplicating a running
  instance.
- Ordinary direct CLI work best-effort starts the agent on macOS. Explicit
  lifecycle, initialization, version, and diagnostic commands keep their
  current intentional behavior and never recursively auto-start.
- `agent stop` is idempotent, and `agent start` is the explicit idempotent
  counterpart to install.

## Requirements

1. Configure the native text view as editable and selectable and change focus
   reconciliation to resign only after a true-to-false binding transition.
2. Preserve native selection, undo, syntax styling, shared drafts, three-second
   autosave, Save/Revert, and current-line tracking.
3. Add an idempotent Go `Lifecycle.Start` and `beacon agent start`; reuse the
   existing user-only plist and avoid restarting a healthy agent.
4. Make `Lifecycle.Stop` safe to repeat while still unloading launchd before
   shutting down any remaining socket/PID-owned process.
5. Start the agent during macOS application startup before subscription retry,
   and synchronously stop it from `applicationWillTerminate`.
6. Best-effort activate the agent for ordinary direct CLI commands without
   polluting stdout, breaking non-macOS builds, or changing direct fallback.
7. Keep one user-scoped agent, reject duplicate foreground servers through the
   existing PID lock, and preserve user data and cached evidence across stops.
8. Reconcile README, constitution, progress, tests, and delivery evidence.

## Assumptions

- “Exit the application” means terminating the Beacon process, not closing the
  detachable dashboard while the menu extra remains active.
- The login item is an application activation and may start the agent.
- A lifecycle-start failure should remain visible in the macOS error state; a
  CLI command may continue through its existing direct fallback.
- Stopping the agent never removes its plist, notes, tracking state, or caches.

## Acceptance Criteria

- [x] AC1: Clicking General or a detail note obtains native editing focus and
  accepts text while explicit blur still resigns focus.
- [x] AC2: Editing continues to update the shared draft, live title/current
  line, autosave, Save/Revert, and tab-switch flush behavior.
- [x] AC3: `beacon agent start` starts a stopped installed agent, installs when
  needed, and is a no-op for a healthy running agent.
- [x] AC4: Repeated `beacon agent stop` succeeds and leaves no live socket/PID
  authority or loaded LaunchAgent.
- [x] AC5: macOS launch starts/reconnects once, normal termination synchronously
  stops the agent, and repeated launch/quit cycles do not orphan or duplicate it.
- [x] AC6: Ordinary direct macOS CLI work activates the agent best-effort while
  lifecycle/version/init/doctor commands and Linux builds remain stable.
- [x] AC7: Focused Go and Swift tests, complete Make validation, Linux build,
  Kit validation, diff hygiene, and hosted checks are reported exactly.

## Implementation Plan

1. Add focus-transition policy and native editor configuration with focused
   Swift tests.
2. Extend Go lifecycle and CLI activation with idempotence and focused tests.
3. Bind AppState startup and application termination to the bundled lifecycle
   helper with injected Swift lifecycle tests.
4. Reconcile documentation, run full validation, self-review, and deliver issue
   #25 on exact branch `GH-25` as a ready pull request.

## Agent Team Plan

- The primary agent owns specification, implementation, validation,
  documentation, and delivery.
- Work remains serial because focus state and process lifecycle are small but
  cross the same application startup and shutdown boundary.
- No subagents are used.

## Task Checklist

- [x] T1: Create issue #25, branch `GH-25`, and this canonical specification.
- [x] T2: Repair the native editor focus lifecycle and add Swift coverage.
- [x] T3: Implement idempotent Go start/stop and CLI activation coverage.
- [x] T4: Bind macOS launch/termination to the agent lifecycle and test it.
- [x] T5: Reconcile README, constitution, and project progress summary.
- [x] T6: Run complete validation and self-review.
- [x] T7: Commit, push, open the ready PR, and report hosted check evidence.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC2 | AppKit editor configuration and focus-transition unit tests plus existing Signal Notes shared-state tests |
| AC3-AC4 | Go lifecycle and CLI command tests for healthy, stopped, absent, and repeated operations |
| AC5 | Swift AppState/application lifecycle tests with an injected lifecycle controller |
| AC6 | CLI activation-routing tests, full Go suite, and Linux amd64 cross-build |
| AC7 | Full Make gate, `kit check --all`, `git diff --check`, and hosted PR checks |

## Reflection Notes

- Treating every false SwiftUI focus binding as a blur request let an unrelated
  update revoke a user click before AppKit emitted its first editing event. A
  true-to-false transition keeps explicit Save blur while leaving initial
  first-responder acquisition intact.
- Explicit `isEditable` and `isSelectable` configuration documents the native
  control contract, and the focused test makes the text view first responder
  and inserts text in addition to checking the transition policy.
- App startup and termination invoke the bundled lifecycle helper synchronously.
  This serializes start before stop and prevents an in-flight detached startup
  process from reloading launchd after the application has exited.
- Go start checks a bounded status request, leaves a healthy agent untouched,
  and refreshes a stopped agent's plist before bootstrap so CLI/app updates
  cannot retain a stale executable path. Stop
  unloads launchd first, then shuts down any remaining socket/PID authority and
  treats an already-stopped job as success.
- XCTest hosts skip the real app-delegate lifecycle because Xcode can tear down
  a test host without a normal AppKit termination callback. Injected model tests
  still cover start/stop ordering without mutating or orphaning the user agent.
- Direct CLI activation is best-effort and stdout-free. Agent management,
  initialization, version, and doctor paths are excluded so status inspection
  and stop cannot recursively start the process they manage.
- The first full-gate run exposed only macOS's Unix-socket path-length limit in
  a new timeout test. Moving that test to the repository's established short
  `/tmp` pattern fixed the test environment; the complete gate then passed.
- The initial hosted Linux job exposed a Darwin-only `launchctl` command-count
  assertion in the otherwise portable stop-idempotence test. The assertion now
  expects launchd commands only on macOS while continuing to exercise the
  already-stopped behavior on Linux.
- The SwiftUI skill reinforced keeping the editor as a native AppKit control
  inside the existing SwiftUI wrapper. No Figma artifact or custom input
  surface was needed.

## Documentation Updates

- Update README background-agent usage and shutdown behavior.
- Update the constitution's CLI and macOS lifecycle contracts.
- Refresh the project progress summary after Kit validation.

## Delivery Decision

The user explicitly requested a new issue, branch, and pull request. The lane is
issue #25, exact branch `GH-25`, and a ready PR targeting `main` with the
repository template and `Closes #25`.

## Evidence

- Pre-mutation recon found clean `main` synchronized with `origin/main`, no
  matching issue/PR or branch collision, authenticated GitHub user
  `jamesonstone`, and human git identity
  `Jameson Stone <jameson@stone.tc>`.
- Issue #25 was created and assigned to `jamesonstone`.
- `origin/main` was fetched and `GH-25` was created from its exact head.
- Go lifecycle coverage verifies first install/start, healthy no-op start,
  repeatable stop, bounded requests to an unresponsive connected agent, and
  direct-CLI activation exclusions.
- The macOS suite passes 69 tests, including native first-responder text input,
  XCTest isolation, and one-start/one-stop application ownership alongside the
  existing shared
  draft, autosave, Save/Revert, and tab-switch coverage.
- The complete local gate passed: `make fmt-check vet test test-race build
  release-test macos-test macos-build`, the Linux amd64 cross-build,
  `kit check --all` across all 15 specifications, and `git diff --check`.
- Live lifecycle smoke terminated two derived-data app instances gracefully and
  observed the LaunchAgent, socket, and PID disappear. Explicit start then
  reported one healthy agent with 80 cached projects; two consecutive stops
  succeeded and again left no loaded job, socket, or PID.
