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
  id: "0011"
  slug: working-notes-refresh
  dir: 0011-working-notes-refresh
relationships:
  - type: builds_on
    target: 0005-beacon-background-agent
  - type: builds_on
    target: 0010-project-following
references:
  - id: issue-9
    name: Refocus project following and animate Beacon wordmark
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/9
    relation: implements
    read_policy: must
    used_for: user-selected delivery lane
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: shared authority, conservative collection, and macOS boundaries
    status: active
skills:
  - name: figma:figma-swiftui
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/figma/2.0.14/skills/figma-swiftui/SKILL.md
    trigger: shared SwiftUI dashboard design and implementation
    required: true
---

# Working Notes and Explicit Refresh

## Thesis

Beacon should offer one low-friction local Markdown scratchpad beside the
working set and one unmistakable manual refresh control. Notes are optional
operator memory, while refresh is an explicit request for current Git and
GitHub evidence; neither changes the observed repositories or GitHub state.

## Context

The working-set dashboard already exposes lane-specific notes and tags, but it
lacks a global place for transient cross-lane thoughts. The existing macOS Scan
Now action is hidden inside Settings, and bare `beacon` intentionally renders
only cached agent state. That behavior conflicts with the user's established
habit of invoking Beacon or pressing refresh immediately after merging one or
more pull requests.

Both macOS surfaces already share one `AppState`, and the background agent
already coalesces forced refresh requests. The implementation should extend
those authorities instead of adding Swift-side persistence or a second scan
path.

## Clarifications

- Working notes are one global Markdown document, distinct from short
  lane-specific notes and tags.
- The document lives at `$XDG_DATA_HOME/beacon/notes.md`, defaulting to
  `$HOME/.local/share/beacon/notes.md`.
- The CLI supports reading, replacing, appending, editing, and locating the
  document through `beacon notes` subcommands.
- The macOS panel edits the same document through the Go agent protocol; Swift
  never writes the file directly.
- The notes panel is collapsed by default and appears at the bottom of the
  shared dashboard surface used by both the menu extra and detachable window.
- The top-right refresh button is always visible, reports in-progress state,
  and coalesces repeated clicks through the existing agent queue.
- A manual refresh bypasses normal remote-evidence age checks while retaining
  GitHub rate-limit reserves and batched collection.
- Bare `beacon` always performs a forced manual refresh. If the agent is
  unavailable, it performs the existing foreground scan rather than returning
  stale data or failing solely because the agent is absent.
- Scheduled background observation remains cache-first and conservative.

## Requirements

1. Add a strict local Markdown notes store with XDG path resolution, user-only
   directory/file permissions, bounded content, and atomic same-directory
   replacement.
2. Add `beacon notes`, `beacon notes show`, `beacon notes set`, `beacon notes
   append`, `beacon notes edit`, and `beacon notes path` without changing the
   existing singular `beacon note <lane-id>` command.
3. Accept note text from arguments or standard input where replacement and
   append commands would otherwise receive no text; never emit ANSI in note
   content.
4. Extend protocol v1 additively with get/set/append global-notes requests and a typed
   notes payload containing content, path, and update time.
5. Keep all note persistence in Go. The Swift client and direct fallback use
   the same CLI/agent authority.
6. Add a whimsical collapsible Signal Notes panel at the bottom of the shared
   dashboard, with a Markdown editor, save state, saved timestamp, and clear
   error feedback.
7. Add a dedicated top-right Scan Now button beside the view and Settings
   controls. Disable it while a scan is active and show progress in place.
8. Remove the duplicate Scan Now item from Settings after the dedicated button
   is available.
9. Make bare `beacon` queue a forced all-project refresh, wait for completion,
   and render the resulting snapshot on every direct invocation.
10. Fall back to the existing foreground scanner when a configured background
    agent is unavailable. TTY output may use the existing loader; piped output
    remains stable and escape-free.
11. Ensure explicit manual refresh includes inactive authored open pull
    requests for followed repositories, uses batched GitHub collection, and
    honors the configured rate-limit reserve.
12. Preserve scheduled cadence, subscription behavior, cached launch state,
    partial failures, and the read-only observation boundary.

## Assumptions

- A single global scratchpad is the simplest useful complement to lane notes.
- Markdown is stored verbatim except that append operations ensure a clean
  newline boundary.
- A 256 KiB document limit is generous for a working scratchpad and prevents
  accidental unbounded protocol payloads.
- The existing agent refresh coalescing is the correct concurrency authority;
  the UI does not need another queue.

## Acceptance Criteria

- [x] AC1: CLI and both macOS surfaces read and edit the same local Markdown
  document without writing repository files.
- [x] AC2: Note writes are atomic, user-only, size-bounded, and preserve valid
  Markdown text across restart.
- [x] AC3: `beacon notes` provides show, set, append, edit, and path workflows
  with deterministic non-interactive behavior.
- [x] AC4: The collapsible Signal Notes panel is available at the bottom of the
  menu and detachable dashboards, shares state, and reports save failures.
- [x] AC5: A visible top-right refresh button triggers exactly one coalesced
  forced scan and accurately reflects agent scan state.
- [x] AC6: Bare `beacon` always performs a manual forced refresh and renders the
  completed evidence, using a foreground fallback when the agent is absent.
- [x] AC7: Explicit refresh sees inactive authored open PRs in followed
  repositories without introducing per-repository GitHub polling.
- [x] AC8: App launch/subscription and scheduled observation remain cache-first
  and do not force GitHub calls.
- [x] AC9: Go, race, Kit, Swift, Xcode, release, and diff-hygiene gates pass.

## Implementation Plan

1. Add the global notes store and CLI commands with focused unit tests.
2. Extend the agent protocol and direct Swift fallback with shared note reads
   and writes.
3. Make bare CLI execution wait for a forced agent refresh with a foreground
   fallback and preserve loader/output contracts.
4. Add the shared SwiftUI Signal Notes panel and dedicated refresh button.
5. Reconcile documentation, validate, independently verify, and update ready
   PR #10 on branch `GH-9`.

## Agent Team Plan

- The supervisor owns specification, Go/Swift implementation, integration,
  validation, documentation, and delivery.
- Go and Swift changes run serially because the Swift contract depends on the
  final protocol shape.
- No implementation specialist is spawned because the shared state and
  protocol changes are tightly coupled.
- The existing read-only verification agent may review the completed diff and
  evidence before delivery; it may not mutate files or GitHub state.

## Task Checklist

- [x] T1: Add the notes store, CLI surface, and tests.
- [x] T2: Extend the agent protocol and bare manual refresh path.
- [x] T3: Implement the macOS notes panel and header refresh action.
- [x] T4: Add Go and Swift regression tests.
- [x] T5: Reconcile README, constitution, and progress summary.
- [x] T6: Run complete validation and read-only verification.
- [ ] T7: Commit focused changes, push `GH-9`, and update PR #10.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC3 | notes store and CLI table tests using temporary HOME/XDG roots, permission checks, stdin/argument cases, and atomic-write failure seams |
| AC4-AC5 | Swift AppState/client/presentation tests plus Xcode build and shared-surface inspection |
| AC6-AC8 | bare dashboard agent/fallback tests, refresh coalescing tests, explicit inactive-PR fixture, and subscription no-refresh regression |
| AC9 | `make fmt-check vet test test-race build release-test macos-test macos-build`, `kit check --all`, and `git diff --check` |

## Reflection Notes

- Keeping one global Markdown document avoids introducing note ownership into
  the snapshot schema and lets the agent publish a small additive protocol
  event to both app surfaces.
- CLI writes must prefer the running agent so a live editor cannot retain stale
  state. Direct-file fallback is safe only when the agent socket is unavailable;
  append remains serialized within the file-store authority and across
  standalone processes through an advisory directory lock.
- A forced refresh should bypass the GitHub cache without expanding inactive-PR
  enrichment across the full discovery inventory. Global search remains
  batched, while old PR detail calls are limited to followed repositories.
- The bare CLI needs foreground fallback at every agent boundary, including
  status polling and final snapshot retrieval, not only initial connection.

## Documentation Updates

- Update README CLI examples and macOS usage.
- Update the constitution's CLI, storage, explicit-refresh, and macOS contracts.
- Refresh the project progress summary after Kit validation if it changes.

## Delivery Decision

The user explicitly requested that this extension continue on issue #9,
branch `GH-9`, and ready PR #10. It remains a separate feature specification
so the completed project-following contract is not rewritten.

## Evidence

- Live recon confirmed a clean `GH-9` working tree matching `origin/GH-9` and
  ready PR #10 before this specification was created.
- `make fmt-check vet test test-race build release-test macos-test macos-build`
  passed, including 41 Swift tests and a successful Xcode Debug build.
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build` passed for the standalone
  CLI's supported release target.
- `kit check --all` passed all 11 feature specifications.
- `git diff --check` passed.
- Focused regressions prove agent-published note set/append events, serialized
  concurrent in-process and cross-process appends, foreground fallback at each
  agent failure phase, and followed-only inactive pull-request enrichment in a
  mixed project batch.
