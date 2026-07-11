# PROJECT PROGRESS SUMMARY

## FEATURE PROGRESS TABLE

| ID | FEATURE | PATH | PHASE | PAUSED | CREATED | SUMMARY |
| -- | ------- | ---- | ----- | ------ | ------- | ------- |
| 0001 | beacon-v1 | `docs/specs/0001-beacon-v1` | deliver | no | 2026-07-09 | Build a read-only agent work-lane review radar as a Go CLI and native macOS menu application backed by the same versioned snapshot. |
| 0002 | beacon-init-dashboard | `docs/specs/0002-beacon-init-dashboard` | deliver | no | 2026-07-10 | Add guided initialization, persistent repository-source discovery, GitHub issue and feedback evidence, Kit progress inference, an active-first colorful dashboard, and schema-v2 macOS parity. |
| 0003 | beacon-github-releases | `docs/specs/0003-beacon-github-releases` | deliver | no | 2026-07-10 | Publish synchronized SemVer CLI and universal macOS artifacts with generated notes and checksums after accepted merges to main. |
| 0004 | project-tracking | `docs/specs/0004-project-tracking` | deliver | no | 2026-07-11 | Let users curate tracked projects while automatically restoring untracked projects when new Git or GitHub evidence appears. |

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

### beacon-init-dashboard

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Make Beacon immediately useful through guided setup and a project-grouped dashboard backed by durable Git, GitHub, and optional Kit evidence.
- **APPROACH**: Persist and rediscover source roots, enrich the shared snapshot with issues, feedback, checks, and progress, derive deterministic next actions, prioritize active work in both human views, and keep idle inventory searchable without removing it from schema-v2 JSON.
- **OPEN ITEMS**: No implementation items remain. Issue #1, branch `GH-1`, and ready PR #2 contain the delivery and validation evidence; final review and merge remain human decisions.
- **POINTERS**: `docs/specs/0002-beacon-init-dashboard/SPEC.md`

### beacon-github-releases

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Turn every accepted merge to `main` into one traceable, downloadable Beacon version for both the CLI and macOS menu application.
- **APPROACH**: Derive SemVer from Conventional Commit history, inject identical release metadata into both products, validate and package platform artifacts, and publish them with generated GitHub release notes and checksums.
- **OPEN ITEMS**: Local implementation and validation are complete on `GH-1` / PR #2. The first live release and same-commit rerun behavior remain post-merge evidence because release automation intentionally runs only after a human merges to `main`.
- **POINTERS**: `docs/specs/0003-beacon-github-releases/SPEC.md`

### project-tracking

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Keep stale projects out of active organizational views without losing visibility when work resumes.
- **APPROACH**: Persist user choices in a separate managed tracking state, baseline durable project evidence when deselected, reconcile changed evidence on every scan, and expose thin CLI and macOS management surfaces over the same Go authority.
- **OPEN ITEMS**: No implementation items remain. AC1-AC10 are complete on issue #3 and branch `GH-3`; final review and merge remain human decisions.
- **POINTERS**: `docs/specs/0004-project-tracking/SPEC.md`

## LAST UPDATED

2026-07-11 EDT
