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
  id: "0012"
  slug: repository-sync-ui-refresh
  dir: 0012-repository-sync-ui-refresh
relationships:
  - type: builds_on
    target: 0008-github-api-budget
  - type: builds_on
    target: 0009-beacon-working-set-radar
  - type: builds_on
    target: 0011-working-notes-refresh
references:
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: shared Go authority, safe process execution, and explicit mutation boundaries
    status: active
  - id: user-ui-request
    name: Repository sync and dashboard refinement request
    type: user-request
    target: conversation
    relation: implements
    read_policy: must
    used_for: repository sync behavior and seven attached UI changes
    status: active
  - id: user-ui-followup
    name: Empty-state and live Markdown follow-up
    type: user-request
    target: conversation
    relation: implements
    read_policy: must
    used_for: whimsical no-work presentation and single-surface rich Markdown editing
    status: active
skills:
  - name: figma:figma-swiftui
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/figma/2.0.14/skills/figma-swiftui/SKILL.md
    trigger: shared SwiftUI dashboard design and implementation
    required: true
---

# Repository Sync and Dashboard Refresh

## Thesis

Beacon should warn when a configured repository is still based on stale default-
branch evidence and should offer one explicit, conservative way to bring one or
many repositories current. The same change should finish the requested dashboard
cleanup by making notes prominent and Markdown-readable, increasing typography,
separating parked work, compacting the header, and simplifying Settings.
When no work lane is in progress, the dashboard should replace its blank body
with a lightweight celebratory state. Signal Notes should also behave as one
live Markdown surface instead of asking the user to switch modes.

## Context

Beacon often observes several repositories participating in one change. A merged
pull request in a backend repository can leave that local checkout on an old
feature branch or an out-of-date `main` while work starts in a dependent UI
repository. Existing Beacon refreshes update evidence, but they do not clearly
identify this local default-branch mismatch or provide an intentionally bounded
way to resolve it.

The menu extra and detachable dashboard share `MenuView` and one `AppState`.
Collection and mutation authority remains in Go. Swift should request typed
repository-sync reports and render the same behavior in both surfaces.

## Clarifications

- Repository sync uses configured or source-discovered repository `base` and
  `remote` values; `main` and `origin` remain the defaults.
- Passive macOS detection performs local Git inspection only. It never fetches,
  invokes `gh`, or calls the GitHub API.
- `Check for Updates` and interactive `beacon sync` are explicit user actions
  that may run bounded `git fetch --prune --no-tags` commands.
- A repository needs attention when its checked-out branch is behind the fetched
  remote default branch or its local default branch is behind that remote branch.
- Beacon may automatically complete only a fast-forward-safe update. It may
  fast-forward a clean checked-out default branch, or switch a clean fully merged
  branch back to the local default branch and then fast-forward it.
- Dirty worktrees, detached heads, missing default branches, checked-out default
  branches in another worktree, and branches with commits not contained in the
  remote default branch remain manual. Beacon reports the reason.
- Beacon never rebases, hard-resets, force-updates, deletes branches, stashes,
  commits, pushes, or changes GitHub state during repository sync.
- `beacon sync` is interactive by default. Scripted and Swift fallback clients
  use explicit `check` and `apply` subcommands with JSON output.
- The repository-sync UI supports selecting one or many safe candidates and
  keeps blocked repositories visible with manual instructions.
- Signal Notes consumes approximately half of the available dashboard height
  while expanded on both macOS surfaces.
- Markdown editing stays lossless and saves the original Markdown source. One
  native editor applies heading, emphasis, list, quote, link, code, and divider
  styling in place as the user types; there is no Edit/Preview mode switch.
- Dashboard typography offers System, Rounded, Monospaced, and Serif system
  designs plus base sizes 11, 12, 13, 14, and 16. The default base size is 12.
- Parking Lot becomes the tab immediately after Following and is removed from
  the Following layouts.
- Recently Updated and Quiet remain primary tabs and are removed from Settings.
- When Following has no in-progress lanes and no projects are loading, both
  macOS surfaces show an adaptive, whimsical all-caught-up backsplash. The copy
  describes lane state only and does not claim repositories are Git-current.

## Requirements

1. Add a shared Go repository-sync service that resolves configured repositories,
   optionally refreshes only their default-branch remote-tracking refs with Git,
   compares checked-out and local default branches to the remote default branch,
   and returns deterministic typed reports with per-repository failures.
2. Use explicit argument arrays, five-second local Git timeouts, 30-second fetch
   timeouts, bounded concurrency, and stable project ordering.
3. Implement fast-forward-only updates with dirty, detached, divergence,
   multi-worktree, missing-ref, and concurrent-change guards.
4. Add `beacon sync`, `beacon sync check`, and `beacon sync apply`. The default
   command must scan, present safe candidates in a TTY multi-select, describe the
   planned Git action, confirm once, and update the selected repositories.
5. Provide deterministic `--json`, `--no-fetch`, and non-interactive apply paths
   for tests and the bundled macOS helper without weakening confirmation defaults.
6. Extend protocol v1 additively with repository-sync check/apply requests and a
   typed report event. Do not add scheduled network work or any GitHub command.
7. Load a local-only repository-sync report when the shared macOS state starts,
   show the current attention count on an equal-sized header button, and open a
   shared selection panel from either the menu extra or detachable dashboard.
8. Require an explicit button click before fetching or applying. Support per-row,
   selected, and all-safe updates while displaying blocked reasons and outcomes.
9. Resize expanded Signal Notes to approximately half of the shared surface and
   provide native live Markdown styling while preserving three-second autosave,
   Save/Revert, and the single Go-owned Markdown document.
10. Increase the default dashboard text scale, expose system font and base-size
    menus in Settings, persist those choices, and keep icons and accessibility
    labels intact.
11. Move Parking Lot into a direct tab immediately after Following and remove
    parked lanes from Following in stacked, tile, and kanban modes.
12. Compact the header by moving lane count and update time to the right of the
    Beacon wordmark and keep refresh, sync, view, and settings controls equal-sized.
13. Remove Recently Updated and Quiet Projects from Settings while retaining
    Manage Following, Open Config, badge restoration, login, agent, and quit
    controls.
14. Update README, constitution, project progress, CLI help, Go tests, Swift
    model/state tests, and visual/manual evidence for the changed behavior.
15. Replace the blank Following body at zero in-progress lanes with a responsive
    SwiftUI backsplash using project palette tokens and SF Symbols, with a
    compact menu-extra presentation and a fuller window presentation.
16. Replace Edit/Preview with one AppKit-backed live Markdown editor that keeps
    plain Markdown as the binding and saved document, updates supported styling
    after each edit, preserves selection and undo, and respects the selected
    dashboard font settings.

## Assumptions

- A fetched remote-tracking default branch is sufficient Git evidence; this
  feature does not need merge-status or pull-request queries.
- A checked-out branch whose HEAD is an ancestor of the remote default branch is
  already merged for the purpose of a safe return to the default branch.
- Refusing ambiguous or destructive updates is more useful than attempting to
  automate merge or rebase conflict resolution.
- Both macOS surfaces should share one repository-sync presentation rather than
  duplicate state or controls.
- A focused live syntax presentation is the simplest viable Notion-like behavior;
  Markdown markers remain editable source rather than introducing a second
  structured document model or lossy source conversion.

## Acceptance Criteria

- [x] AC1: Local-only sync checks detect checked-out and local-default branches
  behind their configured remote default branch without any network or `gh` call.
- [x] AC2: Explicit refresh uses only bounded `git fetch --prune --no-tags`, and
  one repository failure does not discard healthy results.
- [x] AC3: Safe updates fast-forward clean default branches and clean fully merged
  branches, while every risky state is refused with a specific manual reason.
- [x] AC4: Interactive CLI selection updates one or many repositories, JSON paths
  are deterministic, and non-TTY mutation requires explicit targets and approval.
- [x] AC5: Agent and direct-helper paths return the same repository-sync report,
  and no scheduled background GitHub or repository-sync network work is added.
- [x] AC6: The menu extra and detachable dashboard share an equal-sized sync
  button, attention badge, explicit fetch control, multi-select, and update actions.
- [x] AC7: Signal Notes occupies about half the expanded surface and preserves
  saved Markdown, autosave behavior, and immediate rich presentation.
- [x] AC8: System font and base-size choices persist, default to size 12, and
  increase legibility across dashboard, inventory, lane, note, and status text.
- [x] AC9: Parking Lot is the second direct tab and no parked lane appears in the
  Following stacked, tile, or kanban content.
- [x] AC10: The compact header and simplified Settings match the requested control
  placement and remove duplicate Recently Updated and Quiet navigation.
- [x] AC11: Go, race, Swift, Xcode, Kit, release, and diff-hygiene gates pass.
- [x] AC12: With zero in-progress lanes, Following shows an adaptive celebratory
  backsplash in menu and window layouts; loading and non-empty states do not.
- [x] AC13: Signal Notes has no Edit/Preview control and styles Markdown live
  while retaining exact Markdown source, autosave, manual Save/Revert, selection,
  undo, accessibility, and configured font-family/base-size behavior.

## Implementation Plan

1. Add the Git-only repository-sync domain/service and focused integration tests.
2. Add CLI check/apply/interactive workflows and protocol/agent wiring.
3. Add Swift models, clients, shared state, repository-sync panel, and tests.
4. Apply the notes, typography, tab, header, and Settings refinements.
5. Reconcile durable docs, validate, visually inspect both macOS surfaces, and
   record evidence without performing delivery mutations.
6. Add the adaptive zero-lane backsplash and replace mode-based notes with a
   single live Markdown editor, then repeat Swift, Xcode, Kit, and visual gates.

## Agent Team Plan

- The supervisor owns specification, Go/Swift implementation, integration,
  validation, documentation, and final reporting.
- Go, protocol, and Swift work remain serial because each surface depends on the
  same final report and mutation contract.
- No subagents are used because the user did not request delegation and the active
  repository instructions do not require it.

## Task Checklist

- [x] T1: Implement and test the Git-only repository-sync service.
- [x] T2: Implement CLI and protocol/agent repository-sync paths.
- [x] T3: Implement and test shared Swift state and repository-sync UI.
- [x] T4: Implement and test the seven requested dashboard refinements.
- [x] T5: Reconcile README, constitution, and project progress.
- [x] T6: Run complete validation and visual inspection.
- [x] T7: Implement and test the adaptive zero-lane backsplash.
- [x] T8: Implement and test live in-place Markdown styling without mode controls.
- [x] T9: Re-run validation and visually inspect the follow-up behavior.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC3 | temporary multi-repository Git fixtures covering stale refs, fetch, fast-forward, merged branch switch, dirty, detached, divergent, missing, and multi-worktree states |
| AC4 | Cobra and prompt tests for TTY selection, explicit targets, non-TTY approval, JSON, no-fetch, and stable ordering |
| AC5 | protocol round trips, server request tests, Swift agent/direct-helper fallback tests, and recorded command inspection proving no `gh` invocation |
| AC6 | Swift AppState and presentation tests plus menu/window manual inspection |
| AC7-AC10 | Swift model/presentation tests, Xcode Debug build, and live inspection of the default dashboard, sync panel, Parking Lot, and Settings options |
| AC11 | `make fmt-check vet test test-race build release-test macos-test macos-build`, `kit check --all`, and `git diff --check` |
| AC12 | Swift presentation predicates, adaptive compact/window layout review, and live detached-window inspection with zero in-progress lanes |
| AC13 | Markdown style-range and exact-source unit tests, editor binding/focus behavior tests, Swift tests, Xcode build, and live rendered-source inspection |

## Reflection Notes

- Default-branch freshness does not require GitHub merge APIs. Comparing local
  refs to an explicitly fetched remote default ref is both more conservative
  and more directly connected to the local problem.
- The safe mutation set stayed deliberately smaller than the technically
  possible set. Beacon does not advance an unchecked default ref or modify a
  second worktree while the active branch has unmerged commits.
- Apply re-inspects the repository and verifies exact refs immediately before
  mutation. This preserves a useful failure mode when a checkout changes after
  selection instead of applying an outdated plan.
- A new app can encounter an older strict protocol-v1 agent. Repository sync
  falls back to the bundled matching-version helper for unsupported requests,
  while existing snapshot and notes traffic remains on the running agent.
- Native SwiftUI controls and selectable system font designs kept the dashboard
  update dependency-free and consistent across the menu extra and window.
- The follow-up empty state uses native SwiftUI shapes and SF Symbols rather than
  a fixed raster, so it scales across both surfaces and remains theme-aware.
- Live Markdown styling deliberately keeps markers in the editable plain-text
  source. This preserves exact files and undo semantics while providing immediate
  visual hierarchy without a fragile parallel rich-document representation.

## Documentation Updates

- README documents the repository-sync command and safety boundaries.
- The constitution records the explicit Git mutation exception and CLI/protocol
  contracts.
- The project progress summary reflects the validated delivery state.

## Delivery Decision

Deliver the complete validated change through assigned GitHub issue #11 and the
exact issue branch `GH-11`, using a ready pull request to `main`. The requested
delivery does not authorize merge, force-push, branch deletion, or repository
configuration changes.

## Evidence

- Initial recon found a clean `main` checkout at v0.5.0 matching `origin/main`.
- The request was classified as spec-driven because it crosses Go, agent protocol,
  CLI, shared SwiftUI state, and both macOS presentation surfaces.
- Temporary real-Git fixtures prove local-only checks, explicit default-ref
  fetches, clean fast-forwards, merged-branch return, deterministic ordering,
  and refusal of dirty, detached, divergent, missing-default, multi-worktree,
  unmerged, and post-inspection-change cases. Recorded commands contain no `gh`.
- CLI tests prove JSON no-fetch output, non-TTY approval enforcement,
  unambiguous target resolution, and interactive safe-candidate selection and
  update. CLI help exposes `sync`, `sync check`, and `sync apply` as documented.
- Agent protocol round trips and Swift state tests prove typed check/apply
  reports, multi-selection state, and matching-version helper fallback when an
  older running agent rejects the new request.
- `go test ./...`, `make fmt-check vet test-race build release-test`, and a
  Linux amd64 cross-build passed.
- `make macos-test` passed all 48 Swift tests, and `make macos-build` produced a
  successful universal Debug build. The non-fatal local App Intents service
  diagnostics remained unchanged.
- `kit check --all` passed all 12 feature specifications, and
  `git diff --check` passed.
- Live native inspection confirmed the compact equal-sized header controls,
  local-only 20-repository attention report, manual dirty/unmerged rows,
  populated Parking Lot peer tab, the initial half-height notes panel, font and
  size menus, and removal of duplicate recent/quiet Settings items. No
  Check for Updates or repository update action was invoked during inspection.
- The follow-up Swift suite passed all 49 tests. Focused coverage proves the
  backsplash appears only when Following has zero in-progress lanes and zero
  loading projects, and that applying Markdown styles retains the exact source
  while giving headings greater visual weight than body text.
- The final universal Debug macOS build passed. Live inspection of the detached
  dashboard showed the responsive all-caught-up orbit, lane-specific copy, no
  Edit/Preview mode control, and the existing `## test` source rendered as a
  heading. The compact menu-extra branch uses the same predicate and view with
  its explicit compact layout.
- Direct inspection of the notes file after the live UI check confirmed the
  stored source still ended in the literal `## test`; the inspection did not
  edit or trigger a repository-sync network or mutation action.
- The full final gate passed: `make fmt-check vet test test-race build
  release-test macos-test macos-build`, `kit check --all`, and
  `git diff --check`.
- After local completion, the user explicitly requested GitHub delivery. Live
  recon found no matching open issue or existing branch/PR, so assigned issue
  #11 was created and `GH-11` was branched from a freshly fetched `origin/main`.
- Commit `c51c2cd` published the complete validated change set, and ready pull
  request #12 targets `main` with Jameson Stone assigned for human review.
