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
  id: "0019"
  slug: drop-down-terminal
  dir: 0019-drop-down-terminal
relationships:
  - type: builds_on
    target: 0006-beacon-detachable-dashboard
  - type: builds_on
    target: 0012-repository-sync-ui-refresh
references:
  - id: issue-43
    name: Add drop-down terminal hotkey
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/43
    relation: implements
    read_policy: must
    used_for: original request, scope, acceptance criteria, and delivery lane
    status: active
  - id: warp-global-hotkey
    name: Warp Global Hotkey
    type: web
    target: https://docs.warp.dev/terminal/windows/global-hotkey
    relation: informs
    read_policy: must
    used_for: supported Warp dedicated-window behavior and setup guidance
    status: active
  - id: warp-uri-scheme
    name: Warp URI Scheme
    type: web
    target: https://docs.warp.dev/terminal/more-features/uri-scheme
    relation: constrains
    read_policy: must
    used_for: public Warp integration boundary
    status: active
  - id: swiftterm-1-11-2
    name: SwiftTerm v1.11.2
    type: dependency
    target: https://github.com/migueldeicaza/SwiftTerm/releases/tag/v1.11.2
    relation: implements
    read_policy: must
    used_for: native AppKit terminal rendering and local pseudo-terminal lifecycle
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: native macOS boundary, permissions, lifecycle, and validation
    status: active
skills:
  - name: github:github
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/github/0.1.8-2841cf9749ae/skills/github/SKILL.md
    trigger: GitHub issue and delivery orientation
    required: true
  - name: github:yeet
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/github/0.1.8-2841cf9749ae/skills/yeet/SKILL.md
    trigger: user-requested branch, commit, push, and pull request delivery
    required: true
---

# Drop-down Terminal

## Thesis

Beacon should make a real local shell available with one Command-J toggle while
Beacon is active, without intercepting Command-J in any other application. The
terminal should retain its session while hidden, animate
from a persisted top or bottom edge inside the current Beacon dashboard window
bounds, and remain a presentation-only macOS feature that does not alter Beacon
evidence, agent protocol, or scanner policy.

## Context

Beacon is a native macOS 14 AppKit and SwiftUI application with a retained
detached-window controller, application-owned lifecycle, user-default-backed
settings, and no App Sandbox entitlement. Before this feature it had no
terminal emulator or Swift Package Manager dependency.

Warp provides the requested dedicated global-hotkey window, including edge,
screen, and size preferences. Its public URI scheme can open a window, tab, or
launch configuration, but it does not embed Warp or configure and toggle that
dedicated window from another application. Beacon must not mutate undocumented
Warp preferences or control Warp windows through Accessibility. A Beacon-owned
terminal is therefore the supported integrated implementation; Settings can
acknowledge an installed Warp application and open Warp plus its official setup
guide as an external alternative.

SwiftTerm v1.11.2 is an MIT-licensed Swift package with an AppKit
`LocalProcessTerminalView` backed by a Unix pseudo-terminal. It preserves the
native application structure and avoids adding Chromium, Node, private APIs,
or a second process-hosting service. It is the newest stable release before
SwiftTerm made its optional Metal renderer a mandatory build resource, so
Beacon does not require Xcode's separately installed Metal toolchain.

## Clarifications

- Command-J is fixed for version 1. A configurable key recorder is out of scope.
- The built-in terminal is the only Beacon-owned provider. Warp remains an
  explicit external option because no supported API can make it a Beacon view.
- The panel stays inside the current Beacon dashboard window frame, follows
  dashboard moves and resizes, and clips that frame to its visible screen.
- A quiet login launch may materialize the retained dashboard controller to
  resolve its saved bounds without showing the dashboard.
- Settings offers Top and Bottom edges plus Compact (30%), Balanced (45%), and
  Spacious (60%) heights. Top and Balanced are the defaults.
- The second Command-J press hides the panel. Losing focus does not hide it.
- One shell session is retained while hidden. If the shell exits, the
  next show starts a fresh session rather than preserving a dead process.
- The shell uses the executable `SHELL` environment value when it is absolute
  and executable, otherwise `/bin/zsh`; it starts as a login shell in the user
  home directory with `TERM=xterm-256color` and `COLORTERM=truecolor`.
- Beacon never logs or persists terminal input, output, scrollback, or history.
  Ordinary shell-managed history remains the shell's responsibility.
- Custom slide animation is 180 milliseconds and is disabled when macOS Reduce
  Motion is enabled.
- Application termination sends the terminal child process `SIGTERM` through
  SwiftTerm and removes the application-local event monitor.

## Requirements

1. Handle Command-J through an application-local AppKit event monitor without
   requesting Accessibility or Input Monitoring permission or reserving the
   shortcut system-wide.
2. Install the monitor idempotently, remove it during shutdown, consume an exact
   Command-J only inside Beacon, and pass every other key event through.
3. Own one retained terminal window and one local pseudo-terminal session for
   the application lifetime, restarting only after the child process exits.
4. Show and hide the terminal on the main actor, focus it for immediate typing,
   and prevent terminal activation from opening the ordinary dashboard.
5. Calculate deterministic visible and collapsed panel frames within the
   dashboard bounds for both edges, every height, and non-zero origins.
6. Persist edge and height selections with stable user-default keys and apply
   changes to the visible panel immediately.
7. Render the terminal through SwiftTerm v1.11.2, using Beacon's selected code
   font plus a complete semantic default, cursor, selection, ANSI-16, and
   derived 256-color palette. Apply theme changes to a visible session and keep
   default input, the cursor, and every foreground-capable ANSI-16 entry at
   4.5:1 or better against the terminal canvas.
8. Resolve the login shell and environment safely without invoking a shell to
   discover configuration or interpolating command strings.
9. Expose Show Terminal, Position, Height, shortcut health, and supported Warp
   guidance through the existing Settings surface.
10. Keep the Go model, cached evidence, background agent, CLI behavior, and
    versioned protocol unchanged.
11. Document behavior, defaults, Warp boundaries, lifecycle, and troubleshooting.

## Assumptions

- Beacon remains outside the App Sandbox, matching the existing target and the
  SwiftTerm local-process requirement.
- Command-J remains available to the active application; Beacon receives it
  only while Beacon or its terminal panel is active.
- A single local shell is sufficient for version 1; tabs, panes, SSH profiles,
  per-project working directories, and session restoration are non-goals.
- The dashboard's retained current or restored frame is the terminal container;
  it remains authoritative while the dashboard itself is closed.
- SwiftTerm remains pinned to v1.11.2 for reproducible builds without an
  optional Xcode component prerequisite.

## Acceptance Criteria

- [x] AC1: While Beacon is active, Command-J opens a focused terminal inside the
  current dashboard bounds and a second press hides it; while another app is
  active, that app receives Command-J normally.
- [x] AC2: Top and Bottom plus all three heights persist and produce exact
  visible and collapsed frames inside dashboard bounds with non-zero origins.
- [x] AC3: Hiding and reopening preserve the same live shell process;
  an exited shell restarts on the next show, and app termination ends it.
- [x] AC4: Duplicate start and stop calls are safe, Settings reports shortcut
  health, and shutdown removes the local monitor exactly once.
- [x] AC5: The shell starts from the safe resolved executable in the user home
  with a login argument and explicit true-color terminal environment.
- [x] AC6: The terminal uses the selected Beacon code font and complete readable
  theme palette, updates a visible session when the theme changes, respects
  Reduce Motion, and stays usable on multiple displays and full-screen Spaces.
- [x] AC7: Settings contains terminal show, edge, height, hotkey status, and
  installed-Warp guidance without changing Warp preferences or adding a Beacon
  Accessibility permission.
- [x] AC8: Documentation, focused XCTest, complete macOS and repository gates,
  diff/secret review, and a manual application smoke test pass.

## Implementation Plan

1. Add the pinned SwiftTerm package product to the Beacon application target.
2. Add terminal preference, frame, shell-environment, Warp-detection, and
   application-local shortcut types with deterministic seams for focused tests.
3. Add the retained AppKit terminal panel and local-process session wrapper.
4. Integrate terminal ownership with `BeaconApplicationModel`, AppDelegate
   activation, shutdown, and both shared Settings surfaces.
5. Add focused unit tests for presentation math, preferences, lifecycle,
   registration, shell resolution, and Warp availability behavior.
6. Update README, Constitution, project summary, and this specification.
7. Run focused and complete validation, perform live smoke checks, reflect on
   the final diff, then deliver issue #43 on branch `GH-43` as a ready PR.

Rollback removes the Swift package reference, terminal-specific Swift files,
application lifecycle wiring, Settings entries, and documentation. No Go state,
protocol migration, or user-data conversion is required.

## Task Checklist

- [x] T1: Record clarified requirements, public integration boundaries, and
  validation mapping for AC1-AC8.
- [x] T2: Add SwiftTerm and implement preferences, shortcut, panel, and session
  ownership for AC1-AC6.
- [x] T3: Add Settings and Warp guidance for AC4 and AC7.
- [x] T4: Add focused tests and user/canonical documentation for AC1-AC8.
- [x] T5: Run full validation and manual smoke checks, repair defects, and
  record reflection and evidence for AC8.
- [x] T6: Explicitly stage, commit, push, open a ready PR closing #43, and verify
  exact hosted check state.

## Validation Map

| Criterion | Verification |
| --- | --- |
| AC1 | exact shortcut matcher/controller tests plus live Beacon and cross-application Command-J smoke |
| AC2 | frame and preference tests for both edges, three heights, and offset dashboard bounds |
| AC3 | singleton/session lifecycle tests plus live shell identity across hide and reopen |
| AC4 | stub registrar start/stop/failure tests and Settings inspection |
| AC5 | shell resolver and normalized environment tests plus live `echo` smoke |
| AC6 | complete ANSI palette and 4.5:1 contrast assertions across all five themes, live theme refresh, Reduce Motion behavior, dashboard move/resize frame refresh, visible-screen clipping, and panel collection-behavior review |
| AC7 | Settings source assertions, Warp-installed and unavailable tests, permission review |
| AC8 | focused XCTest, `make fmt-check vet test test-race build release-test macos-test macos-build`, Linux amd64/arm64 builds, `kit check --all`, `git diff --check`, secret scan, and final app smoke |

## Reflection Notes

Beacon now owns one retained SwiftTerm-backed login shell and an
application-local Command-J event monitor. Other applications retain their own
Command-J behavior because Beacon never registers the chord system-wide. Warp
remains external because it has no supported
embedding or preference-control API. SwiftTerm is pinned to v1.11.2: later
versions make an optional Metal toolchain a build prerequisite, while v1.11.2
provides the required local-process terminal without that contributor and CI
dependency. Generic macOS destinations are used for universal application and
release builds; XCTest remains active-architecture for deterministic package
compilation.

A live application smoke test opened the panel, executed commands from the
user home, retained the same shell PID across hide and reopen, and applied the
bottom/compact setting before restoring top/balanced. The available macOS
automation can target Command-J to Beacon and another active application because
the shortcut is application-local. A fresh build opened and hid Beacon's panel
with the chord, while the same chord targeted to another frontmost app left the
Beacon panel closed. Focused callback and exact-modifier tests cover the
controller seam.

After the dashboard-bounds follow-up, a fresh isolated build showed the
Balanced terminal at the dashboard's exact width and 45% of its current height.
The collapsed animation frames remain inside the same bounds, and dashboard
move and resize notifications refresh a visible terminal immediately.

The terminal now installs all 16 ANSI entries before deriving its 256-color
palette instead of inheriting SwiftTerm's unrelated defaults. Default text,
ANSI-16 foreground colors, and the cursor meet 4.5:1 against every theme
canvas, including Selenized Dark, and a Settings theme change refreshes an
already-visible panel.

## Documentation Updates

- [x] README macOS terminal usage, defaults, Warp boundary, and troubleshooting.
- [x] Constitution native terminal ownership, permission, dependency, and
  presentation-only boundary.
- [x] Project progress summary feature row and durable feature summary.
- [x] This specification reflection, completion state, delivery decision, and evidence.

## Delivery Decision

Deliver as a new ready-for-review pull request from exact branch `GH-43`, based
on refreshed `origin/main`, assigned to `jamesonstone`, and closing issue #43.
Use explicit staging, verified Jameson Stone author and committer identity, the
repository PR template, and literal hosted-check reporting.

## Evidence

- Issue #43 is open and assigned to `jamesonstone`.
- Branch `GH-43` was created at `b72ae96836d923fce18f5cd6b0d2ee371ccad9bf`,
  exactly matching refreshed `origin/main` before the first edit.
- Official Warp documentation and SwiftTerm v1.11.2 package/source were read to
  resolve the supported integration boundary and local-process API.
- Focused terminal/theme XCTest passed with 21 tests and zero failures; the
  complete XCTest suite passed with 124 tests and zero failures.
- `make fmt-check vet test test-race build release-test` passed, as did Linux
  amd64 and arm64 Go builds.
- Universal `make macos-build` passed and produced both application and helper
  binaries containing x86_64 and arm64 slices.
- A complete synthetic release package passed build, helper metadata, ad-hoc
  signature verification, all four CLI archives, universal app archive, and
  checksum generation.
- Live Settings inspection showed the application-local Command-J status and
  detected Warp. The terminal executed from `/Users/jamesonstone`, and shell
  PID 23827 remained identical across hide and reopen.
- A fresh isolated build opened and hid the Beacon terminal through targeted
  Command-J input. Targeting Command-J to another frontmost application left
  Beacon's terminal closed, confirming that Beacon no longer intercepts the
  chord outside its own process.
- Dashboard-bound frame tests cover both edges, all three heights, non-zero
  origins, collapsed animation frames, and live move/resize refresh. A fresh
  isolated application smoke confirmed the terminal's width and Balanced 45%
  height matched the current dashboard bounds.
- A live retained-shell smoke changed through all five themes and confirmed
  that typed command text and the cursor remained visible while the open
  terminal adopted each new canvas and palette without restarting its shell.
