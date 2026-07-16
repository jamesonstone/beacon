# PROJECT PROGRESS SUMMARY

## FEATURE PROGRESS TABLE

| ID | FEATURE | PATH | PHASE | PAUSED | CREATED | SUMMARY |
| -- | ------- | ---- | ----- | ------ | ------- | ------- |
| 0001 | beacon-v1 | `docs/specs/0001-beacon-v1` | deliver | no | 2026-07-09 | Build a read-only agent work-lane review radar as a Go CLI and native macOS menu application backed by the same versioned snapshot. |
| 0002 | beacon-init-dashboard | `docs/specs/0002-beacon-init-dashboard` | deliver | no | 2026-07-10 | Add guided initialization, persistent repository-source discovery, GitHub issue and feedback evidence, Kit progress inference, an active-first colorful dashboard, and schema-v2 macOS parity. |
| 0003 | beacon-github-releases | `docs/specs/0003-beacon-github-releases` | deliver | no | 2026-07-10 | Publish synchronized SemVer CLI and universal macOS artifacts with generated notes and checksums after accepted merges to main. |
| 0004 | project-tracking | `docs/specs/0004-project-tracking` | deliver | no | 2026-07-11 | Introduce durable project curation and evidence baselines; feature 0010 supersedes its automatic-reactivation behavior with explicit Following. |
| 0005 | beacon-background-agent | `docs/specs/0005-beacon-background-agent` | deliver | no | 2026-07-11 | Render cached state immediately while a user-scoped background agent refreshes followed projects and probes outside inventory for material activity. |
| 0006 | beacon-detachable-dashboard | `docs/specs/0006-beacon-detachable-dashboard` | deliver | no | 2026-07-12 | Add a Dock- and Command-Tab-accessible singleton dashboard plus a quiet optional login item without duplicating Beacon evidence logic. |
| 0007 | queued-project-tracking | `docs/specs/0007-queued-project-tracking` | deliver | no | 2026-07-12 | Make macOS Track and Untrack selections optimistic and nonblocking through an ordered background queue. |
| 0008 | github-api-budget | `docs/specs/0008-github-api-budget` | deliver | no | 2026-07-12 | Preserve the user's GitHub API allowance with shared caching, rate-budget circuit breaking, and network-free batch tracking changes. |
| 0009 | beacon-working-set-radar | `docs/specs/0009-beacon-working-set-radar` | deliver | no | 2026-07-12 | Refocus Beacon on a small lane-level working set with durable attention, factual deltas, conservative enrichment, and direct activity tabs. |
| 0010 | project-following | `docs/specs/0010-project-following` | deliver | no | 2026-07-13 | Make repository Following explicit, retain a complete outside inventory, and conservatively warn when a merged PR's deleted branch remains checked out. |
| 0011 | working-notes-refresh | `docs/specs/0011-working-notes-refresh` | deliver | no | 2026-07-13 | Add one local Markdown signal log plus unmistakable manual refresh controls across the CLI, menu extra, and detachable dashboard. |
| 0012 | repository-sync-ui-refresh | `docs/specs/0012-repository-sync-ui-refresh` | deliver | no | 2026-07-14 | Add conservative Git-only sync, explicit dependency limits, shared dashboard refinements, and read-only merged-checkout advisories backed by bounded exact confirmation. |
| 0013 | signal-note-tabs | `docs/specs/0013-signal-note-tabs` | deliver | no | 2026-07-14 | Extend Signal Notes into a persistent Go-owned tab workspace shared by the CLI, menu extra, and dashboard, with detail history and native quick switchers. |
| 0014 | signal-note-deletion | `docs/specs/0014-signal-note-deletion` | deliver | no | 2026-07-14 | Add permanent detail-note deletion through the Go authority with shared macOS confirmation controls and a higher-contrast switcher. |
| 0015 | notes-agent-lifecycle | `docs/specs/0015-notes-agent-lifecycle` | deliver | no | 2026-07-14 | Restore native Signal Notes input and bind the background agent lifetime to direct CLI or macOS application activation. |
| 0016 | external-task-activity | `docs/specs/0016-external-task-activity` | implement | no | 2026-07-16 | Add transient Codex and Claude Code hook activity to exact followed projects and lanes without changing Beacon evidence or policy. |

## PROJECT INTENT

Beacon provides a dependable local working-set memory for the small set of
coding-agent lanes competing for attention. It derives factual change and the
next useful action from durable Git and GitHub evidence, plus optional local
notes, without relying on chat history, synthetic progress, or agent-private
task state.

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
- **APPROACH**: Persist user choices in separate managed state, baseline durable evidence when deselected, and expose thin CLI and macOS management surfaces over one Go authority. Feature 0010 replaces automatic restoration with explicit Following plus Recently Updated and Quiet categories.
- **OPEN ITEMS**: No implementation items remain. AC1-AC10 are complete on issue #3 and branch `GH-3`; final review and merge remain human decisions.
- **POINTERS**: `docs/specs/0004-project-tracking/SPEC.md`

### beacon-background-agent

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Remove synchronous scan latency from everyday CLI and menu use while preserving complete deterministic direct scans.
- **APPROACH**: Run one user-scoped agent with a versioned Unix-socket protocol, bounded project scheduler, durable per-project cache, full followed scans, lightweight outside-inventory probes, and thin CLI/Swift clients.
- **OPEN ITEMS**: No implementation items remain. AC1-AC13 are complete on issue #3 and branch `GH-3`; final PR #4 review and merge remain human decisions.
- **POINTERS**: `docs/specs/0005-beacon-background-agent/SPEC.md`

### beacon-detachable-dashboard

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Keep Beacon reachable when a crowded or notched menu bar obscures its menu-bar item.
- **APPROACH**: Present one shared SwiftUI dashboard in both the existing menu extra and a regular singleton macOS window, add a neon-space app icon, and use an embedded Service Management login helper for quiet optional startup.
- **OPEN ITEMS**: Implementation and validation are complete on issue #3, branch `GH-3`, and ready PR #4; final review and merge remain human decisions.
- **POINTERS**: `docs/specs/0006-beacon-detachable-dashboard/SPEC.md`

### queued-project-tracking

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Let users curate tens of projects quickly without waiting for each durable baseline probe.
- **APPROACH**: Project selections update optimistically, enter one serial background queue, consume the agent acknowledgement snapshot directly, and roll back individually on failure while later work continues.
- **OPEN ITEMS**: Implementation and validation are complete on issue #3, branch `GH-3`, and ready PR #4; final review and merge remain human decisions.
- **POINTERS**: `docs/specs/0007-queued-project-tracking/SPEC.md`

### github-api-budget

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Keep Beacon background collection from exhausting the GitHub API capacity needed for the user's daily interactive work.
- **APPROACH**: Persist user-only cached `gh` results across agent restarts, keep subscriptions cache-only, skip scheduler collection when no cached project is due, batch default-scope GitHub searches across every due project, enrich only matching PRs, serialize cache misses behind 50% rate-bucket reserves, and probe quiet inventory on a slower cadence.
- **OPEN ITEMS**: Implementation, local state migration, and validation are complete on issue #3, branch `GH-3`, and ready PR #4; final review and merge remain human decisions.
- **POINTERS**: `docs/specs/0008-github-api-budget/SPEC.md`

### beacon-working-set-radar

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Make Beacon a personal memory for the small set of Git, PR, issue, and manual lanes currently competing for attention.
- **APPROACH**: Persist lane-level attention, pins, notes, tags, last-seen observations, and factual deltas; observe local Git frequently without network work; discover GitHub activity globally, retain every open in-scope PR and issue for followed projects regardless of age, and keep the recent cutoff for outside activity; present lane attention inside the explicit Following repository set while Recently Updated and Quiet hold outside project inventory; use distinct mint, cyan, and pink card identities for local, PR, and issue work across stacked, horizontal-tile, and experimental kanban views; expose far-right Ignore actions that park individual Following lanes without changing project membership.
- **OPEN ITEMS**: The working-set implementation is complete on issue #5 / PR #6, and the direct activity-tab refinement is complete on issue #7 / PR #8. The followed-issue visibility, distinct lane-card identity, and far-right Ignore follow-up are delivered on issue #31 / branch `GH-31` in a ready PR targeting `main`; final review and merge remain human decisions.
- **POINTERS**: `docs/specs/0009-beacon-working-set-radar/SPEC.md`

### project-following

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Keep a deliberately selected set of repositories in focus without losing awareness of meaningful activity elsewhere.
- **APPROACH**: Persist explicit Following membership, preserve non-followed evidence baselines, keep every scoped open PR and issue in followed projects visible regardless of age, categorize outside projects without automatic reactivation, support lane-specific Ignore-to-Parking-Lot actions, and use a Git-first, exact-PR, three-candidate confirmation budget for read-only merged-checkout warnings.
- **OPEN ITEMS**: The original implementation is complete on issue #9 / PR #10. The followed-issue visibility, distinct lane-card identity, Ignore-to-Parking-Lot, and bounded merged-checkout warning follow-up are delivered on issue #31 / branch `GH-31` in ready PR #32; final human review and merge remain.
- **POINTERS**: `docs/specs/0010-project-following/SPEC.md`

### working-notes-refresh

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Keep transient cross-lane thoughts close to the working set and make post-merge evidence refresh an obvious explicit action.
- **APPROACH**: Store one atomic user-only Markdown signal log behind Go CLI/agent authority, expose a shared collapsible macOS editor, and make bare CLI plus the top-right app control perform a coalesced forced refresh with batched GitHub evidence.
- **OPEN ITEMS**: Implementation, full validation, and independent verification are complete on issue #9, branch `GH-9`, and ready PR #10; final human review and merge remain.
- **POINTERS**: `docs/specs/0011-working-notes-refresh/SPEC.md`

### repository-sync-ui-refresh

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Keep dependent local repositories current after merged pull requests without spending GitHub capacity or automating risky Git history changes.
- **APPROACH**: Compare local and remote default refs through one Go authority, keep Repository Sync network and mutation behind explicit actions, automate only guarded fast-forwards, and add a separate read-only advisory that confirms at most three previously observed PR transitions per refresh before routing both macOS surfaces to the existing local-only sync report.
- **OPEN ITEMS**: The original implementation is complete on issue #11 / PR #12. The bounded merged-checkout warning is validated on issue #31 / branch `GH-31` in ready PR #32; final human review and merge remain.
- **POINTERS**: `docs/specs/0012-repository-sync-ui-refresh/SPEC.md`

### signal-note-tabs

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Let a brief General Signal Notes line grow into a durable detail document without losing the low-friction shared scratchpad.
- **APPROACH**: Keep General pinned, persist stable-ID detail files plus open order and history through one Go store and additive agent protocol, expose equivalent CLI lifecycle commands, and share one macOS draft/autosave authority with tab and command switchers.
- **OPEN ITEMS**: Implementation, full local validation, live CLI/macOS smoke, and hosted Go/macOS checks are complete on issue #13, branch `GH-13`, and ready PR #14; final human review and merge remain.
- **POINTERS**: `docs/specs/0013-signal-note-tabs/SPEC.md`

### signal-note-deletion

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Permanently remove obsolete detail notes without making tab close destructive or allowing General to be deleted.
- **APPROACH**: Add a separate Go-owned delete lifecycle, route tab, New Tab, and switcher actions through one native confirmation alert, and strengthen switcher contrast.
- **OPEN ITEMS**: Implementation and the complete local gate are finished on issue #21 and branch `GH-21`; final human review and merge remain.
- **POINTERS**: `docs/specs/0014-signal-note-deletion/SPEC.md`

### notes-agent-lifecycle

- **STATUS**: deliver
- **PAUSED**: no
- **INTENT**: Keep Signal Notes directly editable and prevent the Beacon background agent from outliving every user-facing Beacon process.
- **APPROACH**: Reconcile native editor focus only across real focus transitions, add idempotent Go start/stop authority, activate it from direct CLI or macOS launch, and synchronously unload it on application termination.
- **OPEN ITEMS**: Implementation, the complete local gate, and ready-PR delivery are complete on issue #25 and branch `GH-25`; final human review and merge remain.
- **POINTERS**: `docs/specs/0015-notes-agent-lifecycle/SPEC.md`

### external-task-activity

- **STATUS**: implement
- **PAUSED**: no
- **INTENT**: Show transient structured Codex and Claude Code working, latest-attention, and turn-finished context on the exact followed project or lane.
- **APPROACH**: Normalize and map documented hooks in Go, keep current activity in a separately pruned user-only cache, request only coalesced matched Stop refreshes, and let both macOS surfaces render one shared non-authoritative chip.
- **OPEN ITEMS**: Complete documentation, full validation, ready-PR evidence, and hosted checks on issue #31, branch `GH-31`, and PR #32; final human review and merge remain.
- **POINTERS**: `docs/specs/0016-external-task-activity/SPEC.md`

## LAST UPDATED

2026-07-16 EDT
