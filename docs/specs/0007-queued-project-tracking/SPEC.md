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
  id: "0007"
  slug: queued-project-tracking
  dir: 0007-queued-project-tracking
relationships:
  - type: builds_on
    target: 0004-project-tracking
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
    used_for: tracking authority, background agent, and client boundaries
    status: active
skills: []
---

# Queued Project Tracking

## Thesis

Track and Untrack choices should update the macOS inventory immediately and
complete through an ordered background queue so the user can curate many
projects without waiting for each durable baseline probe.

## Context

The original Swift client disabled every project action while one tracking
request ran. The first implementation kept the Go agent's compact local and
GitHub probe synchronous, which could still make each queued acknowledgement
take tens of seconds. Feature 0008 replaces that probe with the already durable
full cached-evidence baseline and defers compact probing to the schedule.

## Clarifications

- The Go agent remains the durable tracking authority. It records the complete
  cached-evidence baseline before acknowledgement; compact probes are deferred.
- The Swift client optimistically moves a selected project between Tracked and
  Untracked and serializes submitted mutations in click order.
- Each queued project shows an individual progress indicator. Other project
  actions, navigation, scanning, and dashboard use remain available.
- One failed mutation rolls back only that project's optimistic state, reports
  a project-specific error, and does not stop later queued mutations.
- The queue is process-local and intentionally does not survive app quit.
- Repeated action on the same project is unavailable while its mutation is
  pending; the user may act again after acknowledgement or rollback.

## Requirements

1. Replace the global project-mutation lock with a serial background queue and
   per-project pending state.
2. Apply optimistic project membership immediately and keep the rest of the UI
   interactive while the queue drains.
3. Consume the tracking-change response snapshot directly so acknowledgement
   does not require another full client scan.
4. Continue processing after individual failures and restore failed projects
   to their last acknowledged tracking state.
5. Show the number of queued/in-flight project changes in the tracking view.
6. Preserve durable baseline probing, automatic reactivation, protocol version,
   snapshot schema, ordering, and Go read-only boundaries.

## Assumptions

- Serial processing is preferable to parallel tracking-file writes and avoids
  races between baseline updates.
- A queue of 10-20 items may take time to drain, but selection latency should be
  effectively immediate and the app must remain usable throughout.

## Acceptance Criteria

- [x] AC1: Ten or more project actions can be selected without waiting for any
  previous network or probe operation.
- [x] AC2: Selected rows move immediately and show per-project pending state.
- [x] AC3: Requests reach the agent in selection order with at most one active
  tracking mutation.
- [x] AC4: Navigation and scanning remain enabled while mutations drain.
- [x] AC5: A failed item rolls back and reports an error while the next item
  still completes.
- [x] AC6: Existing tracking, reactivation, Go, Swift, and packaging validation
  remains green.

## Implementation Plan

1. Add pending-state projection and a serial mutation queue to `AppState`.
2. Return the agent tracking event to AppState and apply its snapshot directly.
3. Remove global button disabling and expose queue state in project tracking UI.
4. Add ordered, nonblocking, optimistic, rollback, continuation, and scan
   interactivity tests.
5. Update project docs and deliver with the active PR.

## Task Checklist

- [x] T1: Implement the optimistic serial queue and response application.
- [x] T2: Update project tracking UI for per-project and queue state.
- [x] T3: Add Swift queue and failure-continuation tests.
- [x] T4: Update documentation and complete full validation.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC4 | delayed-agent Swift tests asserting immediate projection, ordered calls, and one active request |
| AC5 | first-request failure fixture followed by successful queued request |
| AC6 | complete Go/race, Xcode, Kit, and release validation |

## Reflection Notes

Keeping baseline creation in the Go authority preserves reactivation
correctness. Optimistic client projection plus a single ordered task provides
instant selection without introducing concurrent tracking-file writes. Feature
0008 makes the acknowledgement itself network-free and adds atomic batch
selection to protocol version 1.

## Documentation Updates

- Update project progress and README macOS project-curation behavior.

## Delivery Decision

Continue on issue #3, branch `GH-3`, and ready PR #4 with the other active
Beacon client improvements.

## Evidence

- A focused 20-project XCTest queued every selection immediately, projected all
  rows into Untracked, preserved click order, and observed maximum concurrency
  of one.
- Failure-continuation coverage confirmed one rejected mutation rolls back and
  later queued work still completes.
- `make macos-test` passed 31 tests, including queue, lifecycle, protocol, and
  schema coverage.
- Full Go/race/build, Kit, and release packaging validation remained green.
