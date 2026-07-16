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
  id: "0009"
  slug: beacon-working-set-radar
  dir: 0009-beacon-working-set-radar
relationships:
  - type: builds_on
    target: 0005-beacon-background-agent
  - type: builds_on
    target: 0008-github-api-budget
references:
  - id: issue-5
    name: Refocus Beacon on personal working-set lanes
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/5
    relation: implements
    read_policy: must
    used_for: dedicated stacked delivery lane
    status: active
  - id: prerequisite-pr-4
    name: Beacon project controls and API budget
    type: github-pull-request
    target: https://github.com/jamesonstone/beacon/pull/4
    relation: uses
    read_policy: must
    used_for: background protocol, cached snapshots, and rate-budget foundation
    status: active
  - id: pull-request-6
    name: Focus Beacon on working-set lanes
    type: github-pull-request
    target: https://github.com/jamesonstone/beacon/pull/6
    relation: verifies
    read_policy: evidence
    used_for: ready stacked delivery and hosted validation
    status: active
  - id: issue-7
    name: Split oversized files and refine dashboard navigation
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/7
    relation: implements
    read_policy: must
    used_for: direct activity-tab follow-up delivery
    status: active
  - id: issue-27
    name: Improve Beacon visibility and Signal Notes editing
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/27
    relation: supports
    read_policy: must
    used_for: age-independent followed pull-request visibility follow-up
    status: active
  - id: issue-31
    name: Keep followed issues visible and distinguish lane cards
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/31
    relation: supports
    read_policy: must
    used_for: age-independent followed-issue visibility and lane-card identity
    status: active
  - id: pull-request-8
    name: Refine Beacon source structure and dashboard navigation
    type: github-pull-request
    target: https://github.com/jamesonstone/beacon/pull/8
    relation: verifies
    read_policy: evidence
    used_for: follow-up implementation and hosted validation
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: evidence authority, client boundaries, and read-only behavior
    status: active
skills: []
---

# Beacon Working-Set Radar

## Thesis

Beacon should be a personal working-set memory for roughly three to eight
simultaneous AI work lanes, not a fleet-wide repository review dashboard. A
lane's durable Git and GitHub evidence determines what changed and what action
comes next; optional user notes provide context without becoming status truth.

## Context

Beacon currently discovers a broad repository inventory, classifies every open
issue and pull request, and uses project-level Track/Untrack as its primary
attention control. Even with caching and batching, this produces a large view
whose unit does not match the user's actual work: one repository may contain
several independent worktrees, PRs, or planning efforts. The background agent,
stable lane IDs, local Git scanner, PR correlation, cache, and deterministic
next-action policy are the correct foundation, but the default collection and
presentation path must become lane-centered and local-first.

## Clarifications

- Open PRs and issues allowed by the configured GitHub scope remain in the
  working set for followed projects regardless of age; explicit parking can
  still hide a lane. Remote-work identity takes precedence over automatic
  stale-dirty parking, and reconciliation repairs non-explicit parked state
  created by older candidate ordering.
- Dirty and conflicted local lanes age from the durable observation when their
  material status last changed, not from the HEAD commit timestamp. Freshly
  observed or changed dirty work is Active; unchanged dirty work may age into
  Parking Lot after the recent window. Working-set cards show the same material
  observation time and freshness so classification and presentation cannot
  disagree. A cached snapshot that omits the internal status hash preserves the
  last durable hash and does not count as a worktree change.
- Design, research, and planning work without Git evidence uses a manually
  created lane; no Codex task API is required.
- GitHub evidence may be 30–60 minutes stale. Local evidence remains
  near-real-time and explicit Refresh remains available.
- Notes are short local memory cues. Evidence remains canonical, and the UI
  identifies notes written before a later evidence change.
- Lane notation also supports short reusable tags. Tags are user context, not
  durable evidence, and never influence attention or next-action policy.
- Parking is explicit and lane-specific. Unrelated activity elsewhere in the
  repository cannot resume a parked lane.
- Project discovery remains configuration and inventory authority, but project
  Track/Untrack is compatibility management rather than primary attention.
- The rich global scanner remains available through explicit `beacon scan` and
  JSON diagnostics; it is not the automatic default dashboard.

## Requirements

1. Add a strict versioned user-only JSON state owned by the background agent at
   `$XDG_STATE_HOME/beacon/lanes.json` (or the equivalent default under the
   user's home directory).
2. Persist stable lane ID, attention state, pin, optional note and timestamp,
   last-seen time, previous/current durable observation, evidence delta,
   reactivation reason, and local/remote refresh timestamps.
3. Support attention states `active`, `waiting`, `recent`, and `parked`; pinning
   is an independent durable flag.
4. Add manual lanes with stable `manual:<id>` identities and no required Git or
   GitHub fields.
5. Candidate lanes include dirty/conflicted worktrees, unpublished/diverged
   branches, recent local commits, every open in-scope PR and issue for
   followed projects, pins, and manual lanes. Open PR and issue lanes remain
   active regardless of age or stale local state unless explicitly parked.
   Dirty/conflicted lane age must use the last materially changed durable
   observation rather than the HEAD commit timestamp, and missing ephemeral
   cache evidence must not refresh that observation.
6. Reconcile parked lanes only against their own observation. Material
   lane-specific change may reactivate with an explainable reason; unrelated
   project activity may not.
7. Derive factual deltas such as a new commit, publication transition, PR
   opening, CI transition, review-feedback transition, or no material change.
8. Mark note context stale when durable evidence changes after the note.
9. Add shared-authority commands for listing lanes, pinning, parking, resuming,
   editing/clearing notes, adding manual lanes, marking seen, and refreshing one
   lane. Exact Cobra shapes must preserve existing conventions.
10. Bare `beacon` and macOS group the primary view as Active, Waiting, Recently
    Active, and Parked. Repository inventories move to secondary management.
11. Each primary row shows lane title, repository/branch or PR identity, time
    since material activity, concise delta, optional note, and next action.
12. Increment the snapshot schema and update Go JSON output plus Swift decoding
    in one change. Protocol evolution may remain additive when compatible.
13. Frequent scheduled observation must not fetch. Git fetch occurs only for
    explicit refresh or a slow active-lane cadence.
14. Under `github_scope: mine`, use one global authored-PR search and at most one
    assigned-issue search, filter to discovered repositories, and enrich every
    open authored PR in followed projects plus recent/pinned outside work.
    Batch PR enrichment when practical.
15. Default remote evidence age is 45 minutes. Exact cache age is exposed;
    existing generous rate reserves remain enforced.
16. Inactive remote work outside Following receives no review-thread/check
    enrichment in the default path. Open in-scope PRs in followed projects are
    enriched regardless of age; full outside enrichment remains available
    through explicit diagnostics or one-lane refresh.
17. Migrate existing project tracking intent without deleting config or state.
    Existing muted project lanes begin parked; explicit configuration remains
    intact and reversible.
18. Agents are never required to write Beacon, Kit, repository files, or an
    external task system for working-set correctness.
19. Persist zero or more short, deduplicated user tags per lane through the Go
    working-set authority. Preserve existing notes for compatibility, while
    presenting new notation in macOS as removable tag chips.
20. The macOS dashboard moves secondary controls into a top-right Settings
    menu, retains a separate compact view-mode control, and removes the fixed
    action/footer region so lane evidence receives the available height.
21. The macOS dashboard offers persisted `stacked`, `tiles`, and experimental
    `kanban` presentation modes. These modes only rearrange the same ordered
    schema-v3 lanes and must not duplicate policy or alter attention state.
22. Use JetBrains Mono Nerd Font when installed, with the system monospaced
    design as a safe fallback. Typography, spacing, cards, and column widths
    must remain readable in both the 430-point menu surface and detachable
    window.
23. Evidence badges are locally dismissible without mutating their underlying
    Git/GitHub signal. Hover reveals a clickable trailing close control;
    dismissal is scoped to lane, evidence dimension, and exact value so a new
    signal value appears normally. Settings provides a single restore action.
24. Replace stacked secondary-navigation cards with four peer dashboard tabs:
    `Active`, `Parking Lot`, `Quiet`, and `Untracked`. `Active` is selected for
    every new dashboard surface, the tabs expose their current counts, and each
    tab reuses the existing lane/project data and actions without changing
    working-set policy. Tracked-project configuration remains in Settings.
25. Give local-only, pull-request-backed, and issue-backed macOS lane cards
    three distinct palette identities in every dashboard layout. This visual
    identity may style the card, work-item reference, and action text, but must
    not infer or change Go-owned attention or next-action policy.
26. Put an `Ignore` action at the far-right edge of every macOS Following lane
    card in stacked, tile, and kanban layouts. The action must use the existing
    Go-owned parking mutation, move the lane into Parking Lot in both macOS
    surfaces, and never unfollow its project or delete lane state.

## Assumptions

- Strict JSON remains sufficient because Beacon stores only current and
  last-seen observations, not an event history.
- Six hours is the default automatic recent-activity window for local and
  outside-project evidence, not an expiration for open PRs or issues in
  followed projects. A live 80-project
  rollout established that seven days admitted 49 lanes and 48 hours still
  admitted 12, while a work-session-sized window meets the three-to-eight lane
  objective. Pins and explicit resume preserve longer-lived attention.
- An additive protocol command set is preferable to a second local control
  channel.
- The Nerd Font is an optional local presentation enhancement in v1 of this
  design pass; Beacon must remain legible when the font is unavailable rather
  than making application startup depend on a separately installed font.
- Stacking this branch on `GH-3` is required until PR #4 lands because this
  feature intentionally reuses its background-agent and API-budget work.

## Acceptance Criteria

- [x] AC1: A configuration with many repositories renders only the small lane
  working set by default.
- [x] AC2: Two worktrees in one repository have independent attention, note,
  seen, and park state.
- [x] AC3: A parked lane ignores unrelated repository activity and resumes only
  for its own material delta or explicit action.
- [x] AC4: Every open in-scope PR or issue in a followed project appears
  regardless of age or stale local state until it closes or its lane is
  explicitly parked; legacy non-explicit parking is repaired automatically.
- [x] AC5: A manual non-Git lane can be created, noted, parked, resumed, seen,
  and removed from active attention without affecting repositories.
- [x] AC6: Evidence deltas are concrete and deterministic; notes visibly become
  stale when evidence changes after their timestamp.
- [x] AC7: Opening or subscribing to either client initiates no network refresh.
- [x] AC8: Background GitHub collection has a small bounded batched call count
  independent of configured repository count.
- [x] AC9: Frequent local observation performs zero network commands and no
  fetch; explicit refresh remains available.
- [x] AC10: CLI and macOS show identical attention, notes, deltas, and next
  action from one versioned snapshot.
- [x] AC11: Existing explicit full-scan diagnostics remain available.
- [x] AC12: Migration retains user configuration and converts prior muted intent
  without false lane reactivation.
- [x] AC13: Go, race, Kit, Swift, build, and release validation passes.
- [x] AC14: Lane tags persist through the shared Go authority, render as
  removable macOS chips, and do not affect evidence or next-action policy.
- [x] AC15: Settings actions live behind a top-right gear, the fixed footer is
  removed, and all three persisted view modes render the same working set.
- [x] AC16: The compact menu and detachable window remain readable with
  JetBrains Mono Nerd Font installed and with the system fallback.
- [x] AC17: Every evidence badge exposes a hover-only close control, exact-value
  dismissals persist across launches, changed evidence reappears, and Settings
  can restore all hidden badges without changing lane policy.
- [x] AC18: Active is the default dashboard tab; Parking Lot, Quiet, and
  Untracked are directly selectable peers with counts, and the former stacked
  navigation cards and back-button drill-ins are absent.
- [x] AC19: Local-only, pull-request-backed, and issue-backed lanes use three
  distinct macOS card colors in stacked, tile, and kanban layouts without
  changing shared lane policy.
- [x] AC20: Every Following card exposes a far-right Ignore action that parks
  the lane through the shared agent authority, removes it from Following, and
  makes it available in Parking Lot without changing project membership.
- [x] AC21: A dirty followed worktree on an old HEAD remains Active when first
  observed or materially changed, then ages into Parking Lot only after its
  durable observation remains unchanged for the recent window; its card age is
  derived from that same observation, including with a future-dated HEAD or a
  cached snapshot that omits the internal status hash.

## Implementation Plan

1. Add schema-v3 working-set models and strict lane-state storage with migration.
2. Reconcile durable observations, candidate rules, factual deltas, notes,
   parking, pinning, manual lanes, and last-seen state.
3. Extend agent protocol and CLI commands over the shared Go authority.
4. Separate cache-only/local observation from explicit fetch and broad scan,
   retaining `beacon scan` as the diagnostic path.
5. Render lane-centered CLI groups and update Swift models and interactions.
6. Update canonical docs, validate, and deliver on issue #5 / branch `GH-5` as
   a ready PR stacked on `GH-3`.
7. Add shared lane tags and refine the macOS dashboard with a compact Settings
   menu, improved spacing, Nerd Font typography, and three presentation modes.
8. Add reversible macOS-only evidence-badge dismissals keyed by lane,
   dimension, and exact signal value.
9. Replace secondary dashboard drill-ins with a compact four-tab activity
   selector while preserving Settings-based tracked-project management.
10. Keep scoped open issues in followed projects active regardless of age and
    add one shared macOS work-item identity mapping for distinct lane colors.
11. Add one semantic macOS Ignore action over the existing parking mutation and
    render it at the far right of every Following card across all layouts.
12. Age and present dirty local work from its durable material observation so
    old commit timestamps cannot hide or mislabel fresh worktree changes.

## Task Checklist

- [x] T1: Create issue #5, branch `GH-5`, and canonical feature spec.
- [x] T2: Implement lane state, migration, observations, and deltas.
- [x] T3: Implement additive agent protocol and CLI lane commands.
- [x] T4: Implement conservative local/remote working-set collection.
- [x] T5: Implement CLI and macOS primary working-set views.
- [x] T6: Reconcile docs and complete validation.
- [x] T7: Commit, push, and create the ready stacked PR.
- [x] T8: Add shared tags and the macOS settings/view-mode design pass.
- [x] T9: Add hover-to-dismiss evidence badges and restore controls.
- [x] T10: Add Active, Parking Lot, Quiet, and Untracked dashboard tabs.
- [x] T11: Retain open followed issues, add distinct lane identities, and cover
  the shared Go and macOS presentation behavior with focused regressions.
- [x] T12: Add the Following-card Ignore action and cover its visibility and
  shared-authority parking behavior with focused Swift regressions.
- [x] T13: Correct dirty-lane activity timing and cover activation, aging, and
  reactivation from a changed status hash without treating cache-only hash
  omission as activity.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC6 | table-driven working-set state, candidate, observation, delta, note, manual, and transition tests |
| AC7 | Swift scripted-agent and Go protocol call-counter tests |
| AC8-AC9 | 80-repository command-count fixtures and local-only scheduler tests |
| AC10 | schema-v3 JSON fixture decoded and grouped by both Go output and Swift models |
| AC11 | existing scan/golden tests plus explicit diagnostic smoke test |
| AC12 | temporary v1 tracking-state migration fixture |
| AC13 | complete repository validation commands |
| AC14 | workset mutation, protocol, CLI, Swift decoding, and tag interaction tests |
| AC15-AC16 | Swift view-model tests, XCTest, Xcode Debug build, and manual app smoke test |
| AC17 | dismissal-key unit tests, XCTest, Xcode build, and hover/click manual smoke test |
| AC18 | dashboard-tab unit tests, XCTest, Xcode build, and compact-menu manual smoke test |
| AC19 | followed-issue lifecycle test, Swift work-item identity test, XCTest, and Xcode build |
| AC20 | Swift Following-card eligibility and AppState parking tests, XCTest, Xcode build, and compact/detached visual smoke tests |
| AC21 | clock-controlled dirty-lane activation, aging, clock-skew presentation, cache-hash omission, and status-change reactivation regression plus live `terrarium` verification |

## Reflection Notes

- Lane attention must override legacy project tracking for scheduler cadence;
  otherwise resuming a lane inside an untracked project inherits the old slow
  project probe interval.
- Automatic local-only candidate entries are removed when they become inactive
  so hidden historical work cannot retain the fast cadence. Open PRs and issues
  in followed projects remain active until closed or explicitly parked; notes,
  pins, explicit resume, and parking keep the remaining attention durable.
- Pinned remote PRs remain represented from local lane state when enrichment is
  temporarily unavailable. The normal followed-project path now enriches open
  PRs regardless of age, so a pin is not required for visibility.
- Schema upgrades must migrate last-good per-project caches in memory. Rejecting
  schema-v2 cache files as corrupt forced an unnecessary fleet-wide startup
  rebuild and erased the cache-first experience this feature depends on.
- Explicit one-lane refresh must bypass otherwise-valid response-cache entries
  while retaining rate-reserve protection; otherwise a just-opened PR remains
  invisible until the scheduled 45-minute cache age expires.
- The simplest conservative remote policy is constant-cost discovery: the
  default scope always uses two global searches, filters locally, and enriches
  every open match in followed projects while retaining the recent cutoff for
  outside projects. Explicit diagnostics opt into all inactive enrichment.
- Live rollout evidence reduced the automatic recent window to six hours,
  excludes clean base branches, and parks stale dirty work. Open PRs in
  followed projects are the deliberate exception because they remain current
  work until closed or explicitly parked.
- Candidate precedence and persisted attention must agree: an open PR or issue
  outranks stale-dirty auto-parking, while only `Explicit` parking records the
  user's Ignore intent. Reconciliation can therefore repair old automatic
  parked records without undoing an explicit Ignore.
- Git status contains material state but no change timestamp. The durable
  observation timestamp already advances only when the status hash, HEAD, or
  other evidence changes, so it is the correct age authority for dirty work;
  the HEAD commit timestamp is only evidence of commit recency. Cached project
  snapshots omit the private status hash, so reconciliation must carry the last
  durable hash forward rather than inventing activity at helper startup.
- A shared gear menu recovers the fixed footer's height without hiding primary
  evidence. A separate persisted mode button keeps layout selection discoverable
  while stacked, horizontal-tile, and kanban views remain policy-free.
- Tags remain durable context without entering candidate or next-action policy;
  inactive tagged lanes retain their tags in local state without entering the
  visible working set.
- Evidence-badge dismissals use lane, dimension, and exact signal value as the
  presentation key. This keeps the action reversible, avoids hiding changed
  evidence, and leaves the Go snapshot and next-action policy untouched.
- Naming the macOS parking affordance Ignore makes the immediate focus action
  plain without adding a second policy path; AppState still sends the existing
  `parked` mutation and renders the agent's returned snapshot.

## Documentation Updates

- Update the constitution, README, and project progress summary for lane-level
  attention, optional notes, factual deltas, conservative collection, the
  explicit diagnostic boundary, distinct macOS work-item colors, and the
  Following-card Ignore-to-Parking-Lot action.

## Delivery Decision

Create a distinct issue/branch/PR lane as requested. Issue #5 is assigned to
Jameson Stone. Branch `GH-5` starts at `origin/GH-3`, and its PR targets `GH-3`
until prerequisite PR #4 lands.

The later direct activity-tab refinement continues on the explicitly approved
issue #7 / branch `GH-7` / ready PR #8 lane alongside the source-structure
cleanup, with a separate focused feature commit and updated delivery evidence.

The July 15 open-PR visibility correction is delivered as a focused bug-fix
commit on assigned issue #27 and exact branch `GH-27`, within the user-approved
multi-focus ready pull request to `main`.

The followed-issue visibility and distinct lane-card identity follow-up is
delivered on assigned issue #31 and exact branch `GH-31` as a ready pull request
targeting `main`.

The July 16 dirty-worktree activity correction is assigned to issue #33 and
exact branch `GH-33`, created from current `origin/main`, for a ready pull
request targeting `main`.

## Evidence

- Pre-implementation recon found no matching open issue, a clean pushed `GH-3`
  head at `8f11cbf`, passing PR #4 checks, and Jameson Stone as Git author and
  committer.
- Issue #5 was created and assigned to Jameson Stone.
- Branch `GH-5` was created exactly from `origin/GH-3` with no product-code
  changes before this specification.
- `make fmt-check vet test test-race build release-test` passed.
- `make macos-test` passed 33 Swift tests with zero failures.
- `make macos-build` completed with `** BUILD SUCCEEDED **`.
- `kit check --all` passed all nine feature specifications.
- `make fmt-check vet test test-race build release-test` passed after the
  settings, tags, typography, and view-mode design pass.
- `make macos-test macos-build` passed 34 Swift tests and produced the Debug
  application successfully.
- Manual inspection verified the compact gear menu, JetBrains Mono Nerd Font
  rendering, tag chips, and stacked, horizontal-tile, and kanban layouts; the
  persisted selection was restored to the default stacked view afterward.
- `make macos-test macos-build` passed with 35 Swift tests after adding
  deterministic evidence-badge dismissal keys and the hover-only close control.
- Manual inspection verified the hover affordance and the disabled/enabled
  `Restore Hidden Badges` Settings action without changing canonical evidence.
- `make fmt-check vet test test-race build release-test` passed after replacing
  the secondary navigation cards with direct activity tabs.
- `make macos-test macos-build` passed 37 Swift tests and produced the Debug
  application successfully.
- Manual inspection of the Debug application verified Active is the default,
  all four counted tabs select their existing content, Track controls remain
  available under Untracked, and the former navigation cards are absent.
- `kit check --all` passed all nine feature specifications after reconciling
  the README, constitution, progress summary, and this specification.
- `bin/beacon config validate` passed for the user's five-source configuration,
  whose remote refresh interval is now 45 minutes.
- `bin/beacon scan --repo beacon --no-refresh --json | jq ...` returned
  schema version 3, one project, two diagnostic lanes, and zero errors.
- Commit `28ca6c8` was pushed on `GH-5`, and ready PR
  [#6](https://github.com/jamesonstone/beacon/pull/6) targets prerequisite
  branch `GH-3` with issue #5 assigned to Jameson Stone.
- A live schema-v3 rollout exposed and verified the v2 cache migration path;
  `TestCacheLoadUpgradesSchemaTwoSnapshotWithoutQuarantine` prevents future
  startup rebuild regressions.
- A July 15 live reproduction confirmed `lsmc-bio/labcore-ui` was followed and
  PR #31 was open, yet its fourteen-hour age left the working set empty.
- Regression coverage now verifies background inactive-PR enrichment for
  followed projects, age-independent Active membership, and removal after the
  PR closes. The complete Go/race/macOS gate, Linux build, all 15 Kit specs,
  and diff hygiene pass.
- Restarting the rebuilt helper and refreshing `labcore-ui` produced one Active
  lane for PR #31 in the shared agent snapshot and terminal Following output.
- A July 15 live reproduction confirmed `jamesonstone/kit` was followed and
  assigned issue #50 was present in the explicit diagnostic snapshot but absent
  from Following because issue-only lanes were not automatic candidates after
  the six-hour recent window.
- Issue #31 is assigned to Jameson Stone, and branch `GH-31` starts exactly at
  the freshly fetched `origin/main` head.
- `TestReconcileKeepsOpenIssuesInFollowedWorkingSet` now verifies an old scoped
  issue remains Active until closure, while the Swift lane-identity regression
  fixes local-only cards to mint, pull-request-backed cards to cyan, and
  issue-backed cards to pink across all shared layouts.
- `make fmt-check vet test test-race build release-test macos-test macos-build`
  passes with 73 Swift tests and a successful universal macOS build. The Linux
  amd64 cross-build, all 15 Kit specifications, Go formatting, and diff hygiene
  also pass.
- The Following-card Ignore follow-up adds one shared presentation guard and
  one semantic AppState wrapper over the existing `parked` mutation. Focused
  Swift tests verify Following-only visibility and the exact lane/state request,
  including the returned Active-to-Parking-Lot snapshot transition.
- The complete repository gate passes with 75 Swift tests, a universal macOS
  build, the Linux amd64 cross-build, all 15 Kit specifications, and diff
  hygiene. Live 580-point and 430-point window checks keep the Ignore capsule at
  the far-right card edge, while Parking Lot exposes no Ignore controls.
- A July 16 live snapshot exposed six followed `lsmc-bio` pull requests in
  Parking Lot with `explicit: false`: stale dirty checkout classification ran
  before remote-work classification, and reconciliation then preserved the
  resulting automatic parked state.
- Candidate-order and persisted-state regressions now cover both a newly seen
  stale-dirty open PR and a legacy non-explicit parked PR, including preservation
  of an explicit park. After restarting the rebuilt helper, the live snapshot
  changed from 3 Active / 24 Parked to 10 Active / 18 Parked and restored all
  six affected PRs to Active.
- `make fmt-check vet test test-race build release-test macos-test macos-build`,
  the Linux amd64 cross-build, all 16 Kit specifications, and diff hygiene pass;
  the macOS suite executes 81 tests with zero failures.
- A July 16 live reproduction found followed `lsmc-bio/terrarium` parked with
  six unstaged files and one untracked file because its dirty lane inherited the
  older HEAD commit time. After restarting the rebuilt helper, `terrarium/main`
  is Active `now` with `explicit: false`; clock-controlled regressions also
  verify aging, status-change reactivation, clock skew, freshness alignment,
  and startup cache snapshots that omit the private status hash.
- Issue #33 is assigned to Jameson Stone, and branch `GH-33` starts exactly at
  the freshly fetched `origin/main` head for ready pull-request delivery.
