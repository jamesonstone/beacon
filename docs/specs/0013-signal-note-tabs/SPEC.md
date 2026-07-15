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
  id: "0013"
  slug: signal-note-tabs
  dir: 0013-signal-note-tabs
relationships:
  - type: builds_on
    target: 0011-working-notes-refresh
references:
  - id: issue-13
    name: Add Signal Note tabs and quick switchers
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/13
    relation: implements
    read_policy: must
    used_for: user-approved delivery lane and acceptance contract
    status: active
  - id: issue-27
    name: Improve Beacon visibility and Signal Notes editing
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/27
    relation: supports
    read_policy: must
    used_for: native spelling-underlines follow-up
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: Go persistence authority, additive protocol evolution, and shared macOS state
    status: active
skills:
  - name: figma:figma-swiftui
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/figma/2.0.14/skills/figma-swiftui/SKILL.md
    trigger: shared SwiftUI menu-extra and dashboard implementation
    required: true
---

# Signal Note Tabs

## Thesis

Beacon should let one brief Signal Notes line grow into a durable detail note
without losing the low-friction general scratchpad. The CLI, menu extra, and
detachable dashboard should share one persistent tab workspace whose files and
open state remain local, explicit, and Go-owned.

## Context

Feature 0011 added one atomic Markdown document and one shared SwiftUI editor.
Longer ideas currently make that document noisy and cannot remain independently
open. Both macOS surfaces already share `AppState`, and the agent protocol
already publishes notes updates, so the extension should preserve those
authorities rather than add Swift-side files or another synchronization path.

## Clarifications

- General remains the existing `$XDG_DATA_HOME/beacon/notes.md` document and is
  pinned, always open, and never closable.
- Detail files live under `$XDG_DATA_HOME/beacon/notes/`; a versioned workspace
  manifest in that directory stores stable IDs, open order, active selection,
  creation time, and last-opened time.
- The trimmed literal first line is the tab title. Empty first lines display as
  `Untitled`; editing the title never renames the stable file.
- Creating from the current General line copies that line into the detail file
  and leaves General unchanged.
- Open order is stable insertion order. Reopening a closed file appends it;
  selecting an open file activates it without duplication or reordering.
- Closing removes only open-state metadata. It never deletes the Markdown file.
- New Tab is a singleton persistent picker, not a Markdown file. It lists all
  detail notes most-recently-opened first and can create from the current line
  or a separately entered title.
- There is no hard tab-count limit. The existing 256 KiB limit remains per
  Markdown document, and aggregate lists load metadata rather than all content.
- Existing no-selector CLI commands continue to address General.
- Both macOS surfaces share one draft and autosave authority. Dirty content is
  flushed before switching or closing; a failed save keeps the tab active.
- The native editor enables macOS continuous spelling underlines using the
  user's preferred languages and learned words. Grammar checking, automatic
  correction, and custom dictionaries remain out of scope for the lean pass.

## Requirements

1. Add a traversal-safe, user-only, atomic detail-note and workspace store with
   stable timestamp-plus-random IDs, regular-file enforcement, and recovery by
   discovering valid Markdown files when workspace metadata is absent or stale.
2. Represent General, New Tab, open order, active selection, note timestamps,
   and most-recently-opened history with deterministic typed metadata.
3. Add CLI list/new/open/close workflows and `--note` selectors to show, set,
   append, edit, and path while preserving every existing General workflow.
4. Accept stable IDs or unique exact titles, reject ambiguous titles with their
   IDs, reserve `general` and `new`, and support `new --from-line N`.
5. Extend protocol v1 additively with workspace/create/open/close requests,
   optional note selectors, typed workspace payloads, and subscriber updates.
6. Keep direct CLI fallback and agent-backed mutations behaviorally equivalent.
7. Replace per-surface notes drafts with one shared `AppState` workspace and
   autosave authority used by the menu extra and detachable dashboard.
8. Add a horizontally scrollable tab strip, live first-line titles, a New Tab
   button, and hover/focus close controls with accessible labels.
9. Add current-line promotion from the General editor context menu, New Tab
   picker, and quick switcher without adding a persistent footer action.
10. Add a searchable command registry. Command-K exposes all app-wide,
    dashboard, note, and applicable lane/project actions; Command-P exposes
    only General, New Tab, and open or closed detail notes.
11. Support keyboard result navigation, dismissal, next/previous tab cycling,
    bracket cycling, and Command-1 through Command-9 direct selection on both
    macOS surfaces.
12. Reconcile the README, constitution, and project progress summary with the
    new storage, CLI, protocol, and macOS contracts.
13. Enable native spelling underlines in every General and detail-note editor
    without changing stored Markdown, enabling grammar, or applying automatic
    corrections.

## Assumptions

- Most-recently-opened ordering is descending, with creation time and stable ID
  as deterministic tie-breakers.
- Duplicate titles are valid; only selector resolution requires disambiguation.
- New Tab remains open until explicitly closed and is restored across relaunch.
- Closing the active detail selects its left neighbor, then General.
- No delete command or destructive note lifecycle is part of this feature.
- The supplied screenshot is current-state context; no Figma artifact is
  required.

## Acceptance Criteria

- [x] AC1: Existing General storage, no-selector CLI behavior, and the exact
  selector-free agent request shape remain compatible without migration or
  repository writes, including while an older strict agent is still running;
  tab workspace operations use the bundled Go fallback until that agent is
  restarted.
- [x] AC2: Detail files and workspace state are atomic, user-only, bounded per
  document, traversal-safe, stable across relaunch, and unlimited in count.
- [x] AC3: First-line titles, copy-from-line creation, open ordering, duplicate
  prevention, close fallback, and most-recent history are deterministic.
- [x] AC4: CLI lifecycle commands, selectors, stdin, JSON, edit, and path
  workflows behave deterministically through current-agent, unavailable-agent,
  and unsupported-older-agent fallback paths.
- [x] AC5: Protocol v1 publishes additive workspace updates that keep two
  clients synchronized without loading every note body.
- [x] AC6: Menu extra and dashboard display the same open and active tabs, use
  one draft/autosave authority, and preserve dirty content on save failure.
- [x] AC7: New Tab lists and reopens all detail files and creates detail notes
  from either the current General line or an entered first-line title.
- [x] AC8: Command-K, Command-P, cycling, numeric shortcuts, arrows, Return, and
  Escape work from both macOS surfaces without duplicating tabs.
- [x] AC9: Focused Go, race, Linux, Swift, Xcode, Kit, and diff-hygiene checks
  pass and the ready PR reports hosted check state exactly.
- [x] AC10: General and detail-note editors enable native continuous spelling
  underlines while grammar and automatic spelling correction remain disabled,
  and Markdown restyling preserves AppKit's temporary spelling attributes.

## Implementation Plan

1. Add the spec, workspace store, typed models, and focused store tests.
2. Extend the agent protocol and CLI with selectors and lifecycle commands.
3. Move notes draft/autosave authority into shared Swift state and add clients.
4. Build the tab strip, New Tab history, current-line promotion, and switchers.
5. Add regression coverage, reconcile documentation, run validation, and
   deliver issue #13 on branch `GH-13` as a ready pull request.
6. Configure the shared native editor for spelling-only checking and verify
   that Markdown styling does not remove temporary spelling indicators.

## Agent Team Plan

- The supervisor owns specification, Go/Swift implementation, integration,
  validation, documentation, and delivery.
- Work remains serial because storage, protocol, CLI, and Swift state share one
  evolving contract and overlapping files.
- No subagents are used; one accountable lane avoids contract drift.

## Task Checklist

- [x] T1: Create issue #13, branch `GH-13`, and this canonical specification.
- [x] T2: Implement the Go workspace store and storage tests.
- [x] T3: Implement agent protocol and CLI lifecycle support with tests.
- [x] T4: Implement shared Swift state, clients, tabs, and New Tab behavior.
- [x] T5: Implement command and tab switchers plus keyboard navigation.
- [x] T6: Reconcile README, constitution, and project progress summary.
- [x] T7: Run complete validation and self-review.
- [x] T8: Commit, push, open the ready PR, and record hosted check evidence.
- [x] T9: Enable lean native spell checking, add focused Swift coverage, and
  reconcile the user-facing Notes documentation.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC3 | notes store tests for migration, IDs, titles, state, ordering, discovery, security, concurrency, and bounds, plus Swift request-shape coverage for older strict agents |
| AC4 | CLI table tests for lifecycle, selectors, stdin, JSON, ambiguity, editor, current-agent routing, unavailable-agent fallback, and unsupported-older-agent fallback |
| AC5 | protocol server/client subscription and additive JSON decoding tests |
| AC6-AC8 | Swift model, shared-state, autosave, palette, shortcut, and two-surface tests plus manual smoke |
| AC9 | `make fmt-check vet test test-race build release-test macos-test macos-build`, Linux build, `kit check --all`, and `git diff --check` |
| AC10 | Swift editor-configuration assertions, temporary spelling-attribute preservation regression, `make macos-test`, and `make macos-build` |

## Reflection Notes

- Stable IDs decouple file identity from the live first-line title, which keeps
  duplicate titles valid and avoids filesystem churn while editing.
- The manifest stores only workspace state and timestamps. First-line metadata
  is discovered from bounded reads, and only the active document body is loaded.
- General and New Tab are reserved workspace identities. General is always the
  first pinned tab; New Tab is a closable singleton picker rather than a file.
- One directory lock spans the General file, detail files, and manifest so CLI
  and agent mutations share the same cross-process serialization boundary.
- The existing additive protocol version remains sufficient: older General-only
  clients ignore the new fields while current clients receive typed workspace
  snapshots after every create, open, close, set, and append operation.
- Additive request fields are not safe in the opposite direction because older
  agents strictly reject unknown JSON keys. General reads and writes therefore
  keep the original selector-free wire shape even though current clients model
  the document internally with the reserved `general` ID; a workspace lookup
  can then fall back without blanking the editor during an app/agent upgrade.
- LaunchAgent installation is intentionally explicit, so replacing the app does
  not imply that its already-running agent process has restarted. An unknown
  workspace command is therefore treated like agent unavailability for notes
  only: the bundled Go helper uses the same locked store directly, while a
  capable running agent remains the primary mutation and broadcast authority.
- Moving the draft and debounce timer into `AppState` eliminated surface-local
  save races. Switch and close operations now share one flush-or-stay contract.
- Native SwiftUI shortcuts keep dispatch in the frontmost shared dashboard view.
  A deferred focus handoff is required so a newly presented palette receives
  arrows, Return, and Escape instead of leaving focus in the Markdown editor.
- AppKit spelling indicators are temporary layout attributes, so permanent
  Markdown font and color restyling can coexist with native underlines without
  changing the stored text or introducing a second text-processing authority.

## Documentation Updates

- Extend README Signal Notes storage, CLI, and macOS usage.
- Document native spelling underlines and the intentional grammar/autocorrection
  exclusions.
- Extend the constitution's notes ownership, CLI, protocol, and macOS boundary.
- Refresh the project progress summary after Kit validation.

## Delivery Decision

The user explicitly requested a new issue, branch, and pull request. The lane is
issue #13, branch `GH-13`, and a ready PR targeting `main` with the repository
template and `Closes #13`.

After pull request #14 merged, the reported General-loading regression required
a direct completion fix. That follow-up uses issue #15 and branch `GH-15`, with
a separate ready pull request because the original feature lane is already
merged.

After pull request #16 merged, the remaining older-agent failure on detail-note
commands required issue #17 and branch `GH-17`. This follow-up completes the
same upgrade boundary without changing the explicit agent lifecycle contract.

After pull request #18 merged, user feedback found the persistent General-editor
footer action confusing. Issue #19 and branch `GH-19` remove that single button
while preserving current-line promotion in contextual and search-driven entry
points.

The lean native spell-checking refinement is delivered as a focused feature
commit on assigned issue #27 and exact branch `GH-27`, within the user-approved
multi-focus ready pull request to `main`.

## Evidence

- Pre-mutation recon found a clean `main` matching `origin/main`, no matching
  open issue or PR, authenticated human GitHub user `jamesonstone`, and human
  git author/committer `Jameson Stone <jameson@stone.tc>`.
- Issue #13 was created and assigned to `jamesonstone`.
- `origin/main` was fetched, local and remote main had zero divergence, and
  `GH-13` was created at the exact `origin/main` commit with no existing PR.
- Store tests cover migration, stable IDs and titles, open order, MRU history,
  duplicate prevention, close fallback, external discovery, corrupt manifests,
  permissions, traversal and symlink rejection, concurrent writes, unlimited
  tab creation, per-document size bounds, and NUL rejection.
- CLI and agent tests cover General compatibility, lifecycle commands, title and
  ID selectors, ambiguity, stdin, JSON, paths, agent routing and fallback, typed
  workspace broadcasts, active-only content, and subscriber synchronization.
- The Swift suite passes 61 tests covering shared drafts and autosave, switch and
  close flushing, failure retention, live titles, duplicate activation, cycling,
  current-line creation, workspace decoding, and switcher filtering.
- The full local gate passed: `make fmt-check vet test test-race build
  release-test macos-test macos-build`, the Linux amd64 cross-build,
  `kit check --all` across all 13 feature specifications, and
  `git diff --check`.
- An isolated CLI smoke created the supplied `[labcore]` detail from General,
  closed it without deletion, listed it as closed, reopened it after a separate
  process invocation, and confirmed General remained unchanged.
- Live inspection of the built macOS dashboard confirmed persistent tabs,
  closed-note search in Command-P, the complete Command-K registry, Control-Tab
  cycling, filtered Return activation, and Escape dismissal. The first pass
  exposed a palette-focus race; a deferred focus handoff fixed it, after which
  the 61-test Swift suite and universal Debug build passed again.
- Commit `e30b7e2` published the complete feature and ready pull request #14
  targets `main`, is assigned to `jamesonstone`, and remains open for review.
- The first hosted macOS run failed under Xcode 15.4 because the palette
  selection dispatcher called a main-actor closure from a nonisolated method.
  Commit `e38a52c` added the required actor annotation; the focused 61-test
  macOS suite, universal build, Kit, and diff gates passed before it was pushed.
- On PR head `e38a52c`, hosted `go` passed in 38 seconds and hosted `macos`
  passed in 1 minute 38 seconds. The PR is ready, cleanly mergeable, and was not
  merged.
- Pull request #14 subsequently merged as `4a4709c`. Follow-up recon for the
  reported blank General editor found a clean `main` at `origin/main`, no
  matching issue or branch, and created issue #15 plus branch `GH-15` from that
  exact remote head.
- The regression was the inverse compatibility direction: the current app sent
  `note_id` for General after its workspace request fell back, while the older
  running agent strictly rejected that unknown request key. The new Swift
  request-shape test verifies selector-free General reads and writes while
  retaining stable IDs for detail requests; all 62 macOS tests passed.
- The complete local follow-up gate passed: `make fmt-check vet test test-race
  build release-test macos-test macos-build` and the Linux amd64 cross-build.
- After pull request #16 merged as `a3c8bc6`, follow-up recon created issue #17
  and exact branch `GH-17` from the synchronized `origin/main` head. The detail
  failure came from the same older running agent rejecting the new
  `get_notes_workspace` command before it could create a tab.
- CLI capability tests now prove current-agent routing, safe direct-store
  fallback for unavailable and unsupported older agents, selector-free General
  writes, detail writes without unsupported mutation attempts, and preservation
  of supported-agent failures. The Swift regression exercises General loading,
  the supplied `[labcore]` current-line creation, detail saving, switching, and
  closing entirely through the bundled fallback after one failed capability
  request; all 63 macOS tests passed.
- The complete issue #17 local gate passed: `make fmt-check vet test test-race
  build release-test macos-test macos-build`, the Linux amd64 cross-build,
  `kit check --all` across all 13 feature specifications, and
  `git diff --check`.
- Pull request #18 subsequently merged as `1b64ea6`. Follow-up recon found a
  clean, synchronized `main`, no matching open issue, and created assigned issue
  #19 plus exact branch `GH-19` from `origin/main`.
- The persistent `Detail from Line` footer label is absent from the macOS source;
  current-line creation remains available from the General editor context menu,
  New Tab picker, and quick switcher. Revert, Save, and autosave status retain
  their existing implementation.
- The complete issue #19 local gate passed: `make fmt-check vet test test-race
  build release-test macos-test macos-build`, including all 63 macOS tests and
  the universal Debug build, plus the Linux amd64 cross-build.
- The local lean spell-checking follow-up enables AppKit continuous spelling
  while explicitly disabling grammar and automatic correction. The regression
  verifies Markdown restyling preserves temporary spelling indicators; all 70
  macOS tests, the universal Debug build, Go formatting/vet/tests, all 15 Kit
  feature checks, and final diff hygiene passed on the combined dirty tree.
