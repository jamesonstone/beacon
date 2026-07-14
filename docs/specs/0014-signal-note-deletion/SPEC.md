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
  id: "0014"
  slug: signal-note-deletion
  dir: 0014-signal-note-deletion
relationships:
  - type: builds_on
    target: 0013-signal-note-tabs
references:
  - id: issue-21
    name: Add confirmed Signal Note deletion
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/21
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
    used_for: Go persistence authority, additive protocol evolution, and shared macOS state
    status: active
skills:
  - name: figma:figma-swiftui
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/figma/2.0.14/skills/figma-swiftui/SKILL.md
    trigger: native SwiftUI destructive actions and confirmation presentation
    required: true
---

# Signal Note Deletion

## Thesis

Beacon should let users permanently remove obsolete detail notes from every
place those notes are managed, without making the existing tab-close action
destructive. One shared confirmation flow and the existing Go persistence
authority keep deletion deliberate and consistent across both macOS surfaces.

## Context

Feature 0013 added persistent detail-note files, a shared tab workspace, New
Tab history, and Command-K/Command-P switchers. Closing a tab intentionally
preserves its file. Users now need an explicit permanent lifecycle action, and
the note switcher needs stronger visual contrast as its result set grows.

## Clarifications

- General and the singleton New Tab picker can never be deleted.
- Closing a detail tab remains non-destructive and continues to use its X.
- Deleting a detail note removes its Markdown file and workspace metadata.
- Deleting the active note selects its left neighbor, falling back to General.
- Every macOS delete entry point routes through one destructive confirmation
  alert; canceling leaves the workspace unchanged.
- Delete actions appear on detail tabs, at the far right of New Tab history
  rows, and at the far right of detail-note switcher results.
- The creation action is labeled **Create New Note from Highlighted Text in
  General**. Its existing current-General-line copy behavior is unchanged.
- CLI deletion is an explicit command rather than an interactive modal and
  retains current deterministic JSON and agent-fallback behavior.

## Requirements

1. Add a traversal-safe permanent detail-file deletion operation to the Go
   store, preserving directory locking, regular-file enforcement, user-only
   permissions, manifest recovery, and active-tab fallback.
2. Extend agent protocol v1 additively with `delete_note`, publish the resulting
   workspace to all subscribers, and retain direct-store fallback for an older
   running agent that rejects the new request.
3. Add `beacon notes delete <id-or-exact-title> [--json]` with the same stable-ID,
   unique-title, ambiguity, human-output, and escape-free JSON conventions as
   the existing lifecycle commands.
4. Add deletion to the shared Swift `AppState`, agent client, bundled CLI
   fallback, and test doubles so menu and dashboard state cannot diverge.
5. Route tab, New Tab history, and switcher delete controls through one shared
   confirmation alert that names the note and explains permanence.
6. Keep the close X distinct from the destructive delete control and make all
   destructive controls keyboard-focusable and accessibly labeled.
7. Give the quick switcher an opaque, darker background and stronger backdrop
   so its text and row boundaries remain readable over dashboard content.
8. Reconcile README, constitution, project progress, tests, and delivery
   evidence with the permanent-delete contract.

## Assumptions

- The requested word “ote” is a typo; the user-facing label uses “Note.”
- A failed macOS deletion keeps the current workspace and exposes the error.
- A successful deletion cannot be undone inside Beacon.
- No bulk deletion, trash/recovery folder, or General-note deletion is needed.
- The supplied screenshots are current-state context; no Figma artifact is
  required.

## Acceptance Criteria

- [x] AC1: The Go store permanently deletes open or closed detail files, removes
  their metadata, selects the correct fallback, and rejects General/New Tab,
  ambiguous titles, traversal, and symlinks.
- [x] AC2: CLI deletion works by ID or unique exact title with deterministic
  human/JSON output through current-agent, unavailable-agent, and unsupported-
  older-agent fallback paths.
- [x] AC3: Agent deletion broadcasts a typed workspace update to subscribers
  and does not load unrelated note bodies.
- [x] AC4: Shared Swift state deletes through the agent authority or bundled
  fallback and both macOS surfaces reflect the same resulting workspace.
- [x] AC5: Detail tabs, New Tab history rows, and note switcher results expose
  permanent deletion only through the shared confirmation alert.
- [x] AC6: New Tab delete controls sit at each row's far right, General/New Tab
  have no delete action, and the close X remains non-destructive.
- [x] AC7: The creation label uses the approved text and the quick switcher has
  materially darker, more legible contrast.
- [x] AC8: Focused Go and Swift tests, full Make validation, Linux build, Kit,
  diff hygiene, and hosted checks are reported exactly.

## Implementation Plan

1. Add the specification, Go store operation, protocol request, CLI command,
   and focused Go coverage.
2. Extend Swift clients and shared state with deletion and fallback coverage.
3. Add one shared confirmation flow to all three UI surfaces, update the label,
   and darken the switcher treatment.
4. Reconcile documentation, run complete validation, self-review, and deliver
   issue #21 on `GH-21` as a ready pull request.

## Agent Team Plan

- The primary agent owns specification, implementation, validation,
  documentation, and delivery.
- Work remains serial because storage, protocol, CLI, shared state, and UI all
  evolve one destructive lifecycle contract.
- No subagents are used.

## Task Checklist

- [x] T1: Create issue #21, branch `GH-21`, and this canonical specification.
- [x] T2: Implement Go store, protocol, CLI, and focused tests.
- [x] T3: Implement shared Swift deletion, confirmation UI, and contrast.
- [x] T4: Reconcile README, constitution, and project progress summary.
- [x] T5: Run complete validation and self-review.
- [x] T6: Commit, push, open the ready PR, and report hosted check evidence.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1 | Store tests for active/open/closed deletion, fallback, protected identities, ambiguity, file removal, and symlink safety |
| AC2 | CLI lifecycle tests for human/JSON output, selectors, ambiguity, agent routing, and fallback |
| AC3 | Protocol server/client subscription test for `delete_note` workspace broadcasts |
| AC4 | AppState agent and bundled-fallback deletion tests |
| AC5-AC7 | Swift model/UI tests plus macOS build and manual source-level wiring review |
| AC8 | Full Make gate, Linux build, `kit check --all`, `git diff --check`, and hosted PR checks |

## Reflection Notes

- Keeping deletion separate from close preserves the original tab-history
  promise while making the irreversible operation visually and semantically
  explicit.
- Manifest replacement happens before file removal. If removal fails, normal
  discovery safely recovers the still-present detail file rather than hiding
  user data behind stale metadata.
- One `MenuView` deletion request owns the native alert for both macOS surfaces;
  tab, history, and switcher controls cannot bypass confirmation.
- Deleting the active note intentionally discards its dirty draft only after
  confirmation. A failed delete keeps the draft and reschedules autosave.
- The SwiftUI skill reinforced native `Alert` roles, keyboard-focusable plain
  buttons, trailing destructive row actions, and accessible labels. The supplied
  screenshots were sufficient, so no Figma artifact was introduced.

## Documentation Updates

- Extend README Signal Notes lifecycle and macOS usage.
- Extend the constitution's storage, protocol, CLI, and confirmation boundary.
- Refresh the project progress summary after Kit validation.

## Delivery Decision

The user explicitly requested a new issue, branch, and pull request. The lane is
issue #21, exact branch `GH-21`, and a ready PR targeting `main` with the
repository template and `Closes #21`.

## Evidence

- Pre-mutation recon found a clean `main` matching `origin/main`, no branch or
  PR collision, authenticated human GitHub user `jamesonstone`, and human git
  identity `Jameson Stone <jameson@stone.tc>`.
- Issue #21 was created and assigned to `jamesonstone`.
- `origin/main` was fetched and `GH-21` was created from its exact head.
- Store tests cover active, open, and closed deletion, left-neighbor fallback,
  permanent file removal, protected identities, ambiguous titles, traversal,
  and symlink replacement without touching the symlink target.
- CLI and protocol tests cover human and JSON results, current-agent routing,
  unsupported-older-agent fallback, and subscriber workspace broadcasts.
- The Swift suite passes 65 tests, including active dirty-draft deletion,
  shared fallback state, and the exact creation-label presentation contract.
- The complete local gate passed: `make fmt-check vet test test-race build
  release-test macos-test macos-build`, the Linux amd64 cross-build,
  `kit check --all` across all 14 feature specifications, and
  `git diff --check`.
