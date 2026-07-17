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
  id: "0017"
  slug: beacon-focus-notes
  dir: 0017-beacon-focus-notes
relationships:
  - type: builds_on
    target: 0013-signal-note-tabs
  - type: builds_on
    target: 0011-working-notes-refresh
  - type: related_to
    target: 0001-beacon-v1
references:
  - id: issue-35
    name: Refine Beacon focus and notes experience
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/35
    relation: implements
    read_policy: must
    used_for: user-approved requirements and ready pull request lane
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: Go-owned notes persistence, thin macOS clients, and output compatibility
    status: active
skills:
  - name: github:github
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/github/0.1.8-2841cf9749ae/skills/github/SKILL.md
    trigger: GitHub issue and repository delivery orientation
    required: true
  - name: github:yeet
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/github/0.1.8-2841cf9749ae/skills/yeet/SKILL.md
    trigger: user-requested branch, commit, push, and pull request delivery
    required: true
---

# Beacon Focus and Notes

## Thesis

Beacon should make the next useful action obvious while turning Notes into a
playful, flexible working surface. One Go-owned pinned order keeps both macOS
surfaces and the direct CLI fallback consistent, while restrained native space
animations and predictable panel sizing improve the experience without adding
assets, network work, or a second state authority.

## Context

Feature 0013 introduced a persistent local tab workspace with General fixed at
the front and open detail notes in stable insertion order. Its public tab model
already carries a pinned flag, but only General can be pinned and the workspace
manifest does not persist a pinned-detail order. The menu extra and detached
dashboard share one Swift `AppState`, while agent-backed and bundled-CLI paths
share one Go `FileStore`.

The current macOS presentation still says Signal Notes, uses a pencil mark and
static empty state, and exposes only expanded or collapsed sizing. The global
header stacks its refresh age under the focus count, and lane cards put a
labelled `+ Tag` pill before the existing tag chips. Separately, the authorized
starting worktree adds one deterministic `Next:` cue to human scan output and
leaves JSON unchanged.

## Clarifications

- General remains permanently pinned, open, first, non-closable, and
  non-deletable.
- Pinned detail tabs share the existing horizontal tab strip: pinned tabs appear
  on the left, followed by unpinned open tabs and the New Tab control.
- A pinned detail stays open and must be unpinned before it can be closed.
- Users pin or unpin a detail through its tab pill and drag pinned detail pills
  to reorder them. General cannot be dragged away from the first position.
- Pinned membership and order persist through the Go workspace manifest and are
  identical through the agent and bundled CLI fallback.
- Unpinned tabs retain their existing stable open order. Pinning appends a tab to
  the pinned group; unpinning returns it to the front of the unpinned group.
- Notes sizing is shared by the menu extra and detached dashboard. Double-click
  cycles 50 percent, 80 percent, minimized, then 50 percent; the chevron
  single-clicks between minimized and the most recent expanded size.
- Height percentages use the available Beacon surface height, not the physical
  display height.
- The pencil mark is replaced by the solar-system animation; the chevron remains
  the explicit collapse and restore control.
- Native animations use system symbols and drawing primitives and stop moving
  when Reduce Motion is enabled.
- The new-note placeholder is the literal lowercase string `title`.
- The existing local README and human-output changes are intentional parts of
  issue #35 and this pull request.

## Requirements

1. Persist an ordered pinned-detail ID list in the existing user-only atomic
   notes manifest, normalize missing or stale IDs safely, and expose General plus
   valid pinned details in typed workspace metadata.
2. Keep pinned details open and order `open_ids` as General, pinned details,
   then unpinned open tabs without changing unpinned relative order.
3. Add atomic pin-order mutation through the file store, agent protocol, CLI,
   bundled Swift fallback, and shared app state without weakening older-agent
   fallback behavior.
4. Extend the CLI with deterministic pin, unpin, and pinned-order workflows that
   accept stable IDs or unique exact titles and can emit workspace JSON.
5. Add accessible pin controls and native drag reordering to detail pills while
   preserving selection, deletion, keyboard switching, autosave, and General
   invariants.
6. Rename the macOS surface to Notes, use `title` as the creation placeholder,
   remove the expanded tagline, and keep the minimized preview concise.
7. Add a reduced-motion-aware animated rocket mark, Notes solar system, and
   space-themed empty state using native SwiftUI only.
8. Persist and share the three-state Notes presentation, implement the specified
   double-click cycle, and keep explicit chevron collapse and restore behavior.
9. Put refresh age immediately to the right of the focus count and render the
   add-tag affordance as a plus after existing tag pills.
10. Preserve deterministic JSON output while adding one highest-priority human
    `Next:` cue from Ready before Needs Action.
11. Reconcile README, constitution, progress summary, and this specification with
    the delivered storage, CLI, protocol, and macOS behavior.

## Assumptions

- New Tab is never pinnable and remains after all document tabs.
- A pin-order request supplies the complete detail-note pinned order; duplicate,
  reserved, missing, or incomplete reorder inputs fail without changing state.
- Unpinning does not close or activate a note.
- Dropping a pinned detail on General moves it to the first detail position;
  dropping on another pinned detail inserts it before that target.
- The existing macOS 14 minimum supports the native drag and drop APIs used.
- No image asset, schema migration command, or cross-repository change is needed.

## Acceptance Criteria

- [x] AC1: General remains first and pinned, pinned details persist in user order
  across reloads, stale IDs are discarded, and unpinned open order is stable.
- [x] AC2: Pin, unpin, reorder, close, delete, discovery, and concurrent mutation
  paths preserve atomicity, validation, permissions, and tab invariants.
- [x] AC3: Agent protocol, direct CLI, older-agent fallback, and bundled Swift
  fallback produce equivalent pin-order workspaces.
- [x] AC4: Both macOS surfaces show accessible pin controls, pinned-left ordering,
  and drag reordering without regressing autosave, selection, or shortcuts.
- [x] AC5: Notes copy, lowercase `title`, minimized preview, plus-only tag action,
  inline refresh age, and all requested native space animations are present.
- [x] AC6: Reduce Motion leaves every animated mark legible and stationary.
- [x] AC7: Header double-click cycles 50%, 80%, minimized, and 50% on both
  surfaces; chevron single-click restores the most recent expanded size.
- [x] AC8: Human scan output adds one deterministic `Next:` cue while JSON shape
  and actionable-lane ordering remain unchanged.
- [x] AC9: README, constitution, project progress, and spec evidence agree with
  the implementation and contain no stale user-facing contract.
- [x] AC10: Focused and full Go, race, Linux, Swift, Xcode, Kit, and diff-hygiene
  checks pass, and the ready PR reports hosted checks literally.

## Implementation Plan

1. Record issue #35, the clarified behavior, validation map, and ready-PR intent.
2. Extend the Go notes manifest/store and tests with normalized pinned order.
3. Extend protocol and CLI mutations plus older-agent and direct fallbacks.
4. Extend shared Swift models/state/clients and tab interactions.
5. Add presentation sizing, copy/layout changes, native animations, and tests.
6. Reconcile documentation, validate the entire authorized diff, self-review,
   commit, push, open the ready PR, and observe hosted checks.

## Agent Team Plan

- One supervisor owns specification, Go and Swift contracts, integration,
  validation, documentation, and delivery.
- No subagents are used because the store, protocol, fallback, and shared UI
  state evolve as one tightly coupled contract.

## Task Checklist

- [x] T1: Create assigned issue #35 and branch `GH-35` from refreshed
  `origin/main` while preserving the explicitly authorized starting diff.
- [x] T2: Create this ready v2 specification and progress entry.
- [x] T3: Implement and test Go pinned-order persistence and invariants.
- [x] T4: Implement and test protocol, CLI, and fallback parity.
- [x] T5: Implement and test macOS pinned interactions and presentation changes.
- [x] T6: Reconcile README, constitution, and project progress documentation.
- [x] T7: Run focused and full validation and review the complete diff.
- [x] T8: Commit, push, create the assigned ready PR, and record hosted evidence.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC2 | notes store tests for persistence, normalization, ordering, close/delete, discovery, invalid input, and concurrency |
| AC3 | agent protocol mutation tests, CLI routing/fallback tests, Swift request/model/client tests |
| AC4 | shared AppState pin/reorder tests, tab ordering assertions, macOS tests and build |
| AC5-AC7 | Swift presentation-state, copy, accessibility, and reduced-motion assertions plus macOS build |
| AC8 | focused output tests plus JSON regression and full Go tests |
| AC9 | documentation review and `kit check --all` |
| AC10 | `make fmt-check vet test test-race build release-test macos-test macos-build`, Linux build, `kit check --all`, and `git diff --check` |

## Reflection Notes

- Keeping pin order in the existing Go manifest let every writer reuse the
  workspace lock, atomic replacement, selector validation, and user-only file
  contract instead of introducing Swift-owned preference state.
- Normalization derives open order from General, persisted pins, then the prior
  unpinned order, so older manifests upgrade additively and stale detail IDs
  disappear without a migration command.
- One shared `AppState` presentation value makes the menu extra and detached
  window agree on the 50/80/minimized cycle. Native timelines provide the space
  motion and pause deterministically for Reduce Motion.
- The visual smoke initially reached a stale process whose debug binary had been
  replaced on disk. The repository's singleton-aware `macos-run` target closed
  that prior process and launched the fresh build before any UI judgment.

## Documentation Updates

- Extend README Notes workspace, CLI pinning, macOS presentation, and human
  `Next:` output behavior.
- Update the constitution's Go-owned workspace and shared macOS surface contract.
- Add feature 0017 to `docs/PROJECT_PROGRESS_SUMMARY.md`.

## Delivery Decision

- Deliver all issue #35 work, including the explicitly authorized starting diff,
  on branch `GH-35` in one ready pull request targeting `main`.
- Use explicit staging, human author and committer identity, the repository PR
  template, and issue/PR assignment to `jamesonstone`.

## Evidence

- Issue: https://github.com/jamesonstone/beacon/issues/35
- Branch: `GH-35`, created from remote `main` at
  `c73d4482659d70fe5996b39fbafda06131fcaf55`.
- Focused validation: `go test ./internal/notes ./internal/agent ./internal/cli`
  passed.
- Full local gate: `make fmt-check vet test test-race build release-test
  macos-test macos-build` passed; the macOS suite executed 85 tests with zero
  failures.
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...`, `kit check --all`,
  and `git diff --check` passed. `kit check --project` separately retains the
  repository's pre-existing instruction-migration findings for AGENTS, Claude,
  Copilot, and customized v2 artifacts; issue #35 does not alter those files.
- Fresh-build smoke evidence confirmed the Notes copy and space visuals, inline
  age and plus-only tag placement, empty-state `title`, and the visible 50% to
  80% to minimized to 50% header cycle on the shared detached surface. General
  was restored without creating or editing note content.
- Implementation commit:
  `736036d64727ae226ba554e4fbbe4219aa27a144`, authored and committed by
  Jameson Stone.
- Ready pull request: https://github.com/jamesonstone/beacon/pull/36, targeting
  `main`, assigned to `jamesonstone`, and closing issue #35.
- Hosted `go`, `macos`, and `Assign configured maintainers` checks pass on the
  final pull-request head.
