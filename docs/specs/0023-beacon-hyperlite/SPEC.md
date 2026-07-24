---
kit_metadata_version: 1
artifact: spec
workflow_version: 3
phase: deliver
delivery_intent: ready_pull_request
feature:
  id: 0023
  slug: beacon-hyperlite
  dir: 0023-beacon-hyperlite
relationships:
  - type: builds_on
    target: 0009-beacon-working-set-radar
  - type: builds_on
    target: 0018-following-workspace
references:
  - id: issue-80
    name: Prototype Hyperlite compact attention popover
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/80
    relation: implements
    read_policy: must
    used_for: compact popover scope and acceptance criteria
    status: active
---

# Hyperlite

## PURPOSE

Hyperlite is the compact menu-bar presentation for answering “what needs my
attention?” in under two seconds while retaining every currently active work
lane in one glanceable list.

## CONTEXT

The existing menu-bar surface renders the full dashboard, which makes the
first-glance question compete with Notes, settings, and other workspace
controls. Beacon already has cached working-set evidence and a low-CPU
event-driven background agent, so Hyperlite can remain a presentation-only
change. Existing timestamps describe evidence freshness, not task start time;
external activity hooks are the only optional source for an observed working
duration.

## REQUIREMENTS

- Show an explicit attention count and attention-first lane list.
- Retain active, waiting, and recently active lanes in the same popover.
- Show project, work-item identity, next action, and factual age.
- Show “working for” only when external activity provides an observed start
  timestamp; otherwise label evidence age as “updated”.
- Keep the surface read-only and event-driven with no continuous animation or
  timer-based invalidation.
- Provide direct actions to open a lane, refresh evidence, and open the full
  dashboard.

Non-goals are changing the Go snapshot schema, adding a task timer, or
replacing the existing dashboard/window surface.

## ACCEPTED PLAN

1. Add a pure Hyperlite presentation model over the existing snapshot and
   external-activity records.
2. Add a compact SwiftUI popover view as the default menu-bar content.
3. Keep the full dashboard available through one explicit button.
4. Cover ordering, deduplication, age formatting, and activity-duration truth
   with focused macOS tests.

## DECISIONS

- The menu-bar surface is Hyperlite; the detachable dashboard remains the
  detailed workspace.
- “Needs attention” is derived from review readiness, blockers, warnings, and
  action-oriented next-action values. Waiting and recently active work remain
  visible but are not promoted without evidence.
- Relative ages are computed when the popover renders and do not run a timer.

## DISCOVERIES

The current working-set groups already provide stable active, waiting, and
recent lane identifiers, so no schema or scanner change is required. External
activity records can be matched to an exact lane and provide an observed
timestamp for working sessions; ordinary lane updates must remain labeled as
evidence age.

## VALIDATION

- `make macos-test` passed all 160 tests.
- `make fmt-check vet test test-race release-test build` passed.
- Focused tests cover attention-first ordering, duplicate lane removal,
  compact age formatting, and observed working activity.

## OUTCOME

The menu-bar extra now opens Hyperlite by default. It presents attention-first
work, retains the remaining active lanes, allows explicit refresh, opens a
selected lane, and links to the existing full dashboard. No Go schema or
continuous UI timer was added.

## REPOSITORY MEMORY

Decision: created.

Rationale: Hyperlite introduces a durable product surface, ordering policy,
and truthful elapsed-time boundary that future UI work should preserve.

Artifacts: `docs/specs/0023-beacon-hyperlite/SPEC.md`, `README.md`,
`docs/USER_GUIDE.md`, and `docs/PROJECT_PROGRESS_SUMMARY.md`.
