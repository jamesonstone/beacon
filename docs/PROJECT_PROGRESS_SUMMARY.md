# PROJECT PROGRESS SUMMARY

## FEATURE PROGRESS TABLE

| ID | FEATURE | PATH | PHASE | PAUSED | CREATED | SUMMARY |
| -- | ------- | ---- | ----- | ------ | ------- | ------- |
| 0001 | beacon-v1 | `docs/specs/0001-beacon-v1` | deliver | no | 2026-07-09 | Build a read-only agent work-lane review radar as a Go CLI and native macOS menu application backed by the same versioned snapshot. |

## PROJECT INTENT

Beacon provides a dependable local signal layer for supervising coding-agent
work lanes. It derives review readiness and the next useful action from durable
Git and GitHub evidence without relying on chat history, synthetic progress, or
agent-private task state.

## GLOBAL CONSTRAINTS

See `docs/CONSTITUTION.md` for project-wide constraints and principles.

The project progress table and summaries must always reflect the highest
completed evidence-backed artifact or workflow-v2 phase for each feature. The
canonical feature artifact wins whenever this index disagrees with it.

## FEATURE SUMMARIES

### beacon-v1

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Identify which active Git worktree and pull-request lanes are ready for human review, need action, are waiting, or are idle.
- **APPROACH**: Keep configuration, Git and GitHub scanning, lane correlation, policy, deterministic ordering, and schema-v1 output in the Go CLI; keep the SwiftUI menu application a thin viewer over the bundled CLI.
- **OPEN ITEMS**: No implementation items remain. Issue [#1](https://github.com/jamesonstone/beacon/issues/1) is represented by ready-for-review PR [#2](https://github.com/jamesonstone/beacon/pull/2); AC1-AC12 and T1-T8 are complete, and the spec records the required Go, race, CLI, macOS, CI, and read-only-boundary evidence.
- **POINTERS**: `docs/specs/0001-beacon-v1/SPEC.md`

## LAST UPDATED

2026-07-09 18:10:15 EDT
