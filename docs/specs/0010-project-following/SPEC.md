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
  id: "0010"
  slug: project-following
  dir: 0010-project-following
relationships:
  - type: builds_on
    target: 0004-project-tracking
  - type: builds_on
    target: 0009-beacon-working-set-radar
references:
  - id: issue-9
    name: Refocus project following and animate Beacon wordmark
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/9
    relation: implements
    read_policy: must
    used_for: requirements and delivery lane
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
    used_for: age-independent followed-issue visibility, lane-card identity, and merged-checkout warnings
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: shared authority, conservative collection, and macOS boundaries
    status: active
skills: []
---

# Project Following and Neon Wordmark

## Thesis

Beacon should distinguish repositories the user deliberately follows from
outside repositories that merely changed. Following is a stable manual choice;
recent outside activity is an inbox, not permission for Beacon to change that
choice. The macOS wordmark may be whimsical without sacrificing accessibility
or wasting resources.

## Context

Beacon currently models every discovered repository as tracked unless it has an
inverse untracked record. Material evidence automatically removes that record,
which silently restores the repository to the primary set. This makes a large
source tree drift back toward dozens of tracked projects and prevents the user
from maintaining a small intentional focus set. The current Active, Parking
Lot, Quiet, and Untracked tabs also mix lane attention with repository
membership.

The durable scanner, conservative muted-project probes, background protocol,
optimistic mutation queue, and lane working set remain useful. The change is to
make repository membership explicit and show outside activity separately.

## Clarifications

- `Following` is the macOS and documentation term. Existing CLI `track` and
  `untrack` commands remain compatibility aliases.
- Following membership changes only through an explicit user action.
- Material local or GitHub evidence in a non-followed repository records an
  activity timestamp and factual reason without following the repository.
- `Recently Updated` contains non-followed projects whose last material
  activity falls within `settings.stale_after`, currently 24 hours by default.
- `Quiet` contains every other discovered non-followed project.
- New repositories discovered after state initialization begin Quiet.
- Existing v1 project-selection state migrates to the explicit model without
  dropping its current followed/unfollowed choices.
- Lane attention remains independent. Following renders every open in-scope PR
  and issue lane plus the existing local lane working set for followed
  repositories; outside projects remain repository cards until followed.
- Open PR and issue identity takes precedence over stale local-worktree
  auto-parking. Reconciliation restores older non-explicit parked records while
  preserving Parking Lot entries created by an explicit Ignore.
- Dirty followed work remains visible from its last materially changed durable
  observation even when HEAD is old; only unchanged dirty state may age into
  Parking Lot, and the card shows that observation age and freshness rather
  than commit age. Startup cache snapshots that omit the private status hash do
  not refresh the durable observation.
- The neon wordmark animation is presentation-only and becomes static when the
  user enables Reduce Motion.

## Requirements

1. Replace inverse default-tracking persistence with a strict version-2 state
   that records every known project and its explicit followed membership.
2. Migrate version-1 state atomically on the first complete reconciliation:
   existing untracked entries remain non-followed and existing visible tracked
   projects become followed.
3. Add newly discovered projects as non-followed without changing any existing
   membership.
4. Preserve material evidence baselines for non-followed projects and persist
   `last_activity_at` plus a factual activity reason when evidence changes.
5. Remove automatic project reactivation. Neither local nor GitHub evidence may
   follow a project or remove its non-followed state.
6. Categorize projects as `following`, `recent`, or `quiet` in the shared Go
   snapshot. Recent classification uses the configured stale duration; Quiet is
   the non-followed fallback.
7. Keep legacy tracked/untracked summary and command compatibility for one
   schema generation while adding explicit following/recent/quiet counts and
   project state.
8. Add `beacon follow` and `beacon unfollow`; preserve `track` and `untrack` as
   aliases over the same authority. Update the interactive selector language to
   Following.
9. Replace the macOS dashboard tabs with `Following`, `Recently Updated`, and
   `Quiet`; Following is the default and each tab shows its project count.
10. Following renders the existing focused lane layouts and keeps every open
    in-scope PR and issue for a followed project visible regardless of age
    or stale local state unless its lane is explicitly parked. Reconciliation
    must repair legacy automatic parked state without overriding explicit
    Ignore. Dirty local lanes use their last materially changed observation for
    activity age rather than the HEAD commit time, preserving the durable hash
    when startup cache evidence omits it. Recently Updated and Quiet render
    searchable project cards with a nonblocking Follow action and visible queued
    state.
11. Keep project-selection management in Settings, relabeled for Following,
    with explicit Follow and Stop Following actions.
12. Remove auto-reactivation banners and terminology from the current user
    experience while retaining safe migration of legacy state.
13. Animate the `Beacon` wordmark with a horizontally traveling neon/pastel
    gradient at a modest frame rate. Use the existing palette, keep text
    contrast readable, and render a static gradient under Reduce Motion.
14. Use the same Go snapshot and mutation authority in the menu surface and
    detachable dashboard. Swift must not infer material activity.
15. Preserve conservative probe cadence, GitHub request batching, rate-budget
    reserves, and the read-only scanning boundary while collecting all open
    in-scope PRs and issues for followed projects.
16. Reserve `Quiet` for non-followed projects without recent activity. Label
    hidden idle lanes inside followed repositories as `Idle Following Projects`.
17. Every macOS Following lane card exposes an `Ignore` action at its far-right
    edge. Ignore parks only that lane through the shared Go authority so it
    leaves Following and appears in Parking Lot without changing project
    membership.
18. When a followed project's previously observed open pull request disappears
    while its local worktree remains on the same non-default branch, perform at
    most three exact, cached GitHub confirmations per completed refresh. Expose
    a read-only warning only when `gh` confirms the pull request merged and Git
    confirms the remote head branch no longer exists.

## Assumptions

- The existing `settings.stale_after` duration is the least surprising recent
  project window and avoids adding another configuration field.
- Keeping compatibility aliases is safer than removing established CLI scripts
  while the product vocabulary changes.
- Current followed membership is the only lossless migration choice because
  version-1 state cannot distinguish default inclusion from an old explicit
  Track action.
- A small timeline-driven gradient over one short text label is sufficiently
  cheap when capped and disabled under Reduce Motion.

## Acceptance Criteria

- [x] AC1: A fresh state initializes every discovered repository as Quiet and
  requires an explicit action to enter Following.
- [x] AC2: Version-1 state migrates atomically while preserving current project
  membership and valid evidence baselines.
- [x] AC3: Material local or GitHub evidence moves a non-followed project from
  Quiet to Recently Updated without changing its membership.
- [x] AC4: Recent projects return to Quiet after `settings.stale_after` and
  expose their last activity time and reason while recent.
- [x] AC5: Follow and Stop Following changes are optimistic, nonblocking, and
  shared by CLI, menu, and detachable dashboard.
- [x] AC6: The macOS dashboard defaults to Following and directly selects
  Following, Recently Updated, and Quiet with accurate counts.
- [x] AC7: Recently Updated and Quiet are searchable, show project identity and
  evidence age, and can Follow a project without navigating elsewhere.
- [x] AC8: Existing focused lane layouts remain available inside Following,
  every open in-scope PR and issue for a followed project remains visible
  regardless of age or stale local state, legacy non-explicit parking is
  repaired, and outside project activity does not enter until followed.
- [x] AC9: `beacon follow` / `unfollow` work and existing `track` / `untrack`
  aliases retain behavior without duplicating authority.
- [x] AC10: The Beacon wordmark visibly cycles through the existing neon
  palette and is static when Reduce Motion is enabled.
- [x] AC11: Opening or subscribing to either macOS surface does not add scans,
  fetches, GitHub calls, or duplicate animation authorities.
- [x] AC12: Go, race, Kit, Swift, Xcode, migration, output, and release gates
  pass with no schema or cache regression.
- [x] AC13: Following excludes every non-followed repository lane without
  deleting durable lane state, keeps followed open PR and issue lanes visible
  until closed or explicitly parked, keeps freshly changed dirty work visible
  even on an old HEAD without treating cache-only hash omission as activity,
  and never labels idle followed inventory Quiet.
- [x] AC14: Ignore is available at the far right of Following cards in every
  macOS layout and moves only the selected lane into Parking Lot through the
  shared attention mutation.
- [x] AC15: A previously observed merged pull request whose deleted remote head
  remains checked out locally produces one Go-owned warning in both macOS
  surfaces; unrelated, unconfirmed, non-followed, and over-budget projects do
  not produce warnings or broad merged-PR polling.

## Implementation Plan

1. Introduce the version-2 explicit-following state and v1 migration.
2. Preserve non-followed evidence, record material activity, and derive shared
   project categories and counts.
3. Add CLI vocabulary aliases and update terminal/project selection copy.
4. Update Swift decoding, optimistic presentation state, and the three-tab
   project-following navigation.
5. Add the accessible neon wordmark component and use one animation per visible
   dashboard surface.
6. Reconcile documentation, validate, independently verify, and deliver on
   issue #9 / branch `GH-9` / a ready PR targeting `main`.
7. Add a semantic Ignore-to-park action to the shared macOS card renderer and
   verify the identical behavior in the menu and detachable window.
8. Persist transition-scoped merged-checkout confirmation, cap exact `gh`
   requests at three followed candidates per refresh, and render the confirmed
   advisory through the shared lane card without mutating the repository.

## Agent Team Plan

- The supervisor owns the spec, model, migration, integration, validation,
  documentation, and delivery state.
- Go authority/schema and Swift presentation are logical lanes but run serially
  because the Swift contract depends directly on the final Go snapshot.
- No implementation specialist is spawned because the migration, compatibility
  layer, and UI model require continuous shared design judgment.
- One read-only verification agent reviews the completed diff, acceptance map,
  and validation evidence before delivery. Maximum concurrency is two.

## Task Checklist

- [x] T1: Complete live recon and create issue #9 plus branch `GH-9` from
  current `origin/main`.
- [x] T2: Implement explicit following state, migration, and recent evidence.
- [x] T3: Update snapshot/output/CLI compatibility contracts.
- [x] T4: Implement Following, Recently Updated, and Quiet macOS views.
- [x] T5: Add the accessible animated neon wordmark.
- [x] T6: Add Go and Swift tests plus migration fixtures.
- [x] T7: Reconcile README, constitution, and progress summary.
- [x] T8: Run full validation and read-only verification.
- [x] T9: Commit, push, open the ready PR, and record hosted evidence.
- [x] T10: Add and test the far-right Following-card Ignore action.
- [x] T11: Add and test conservative merged-checkout confirmation and the
  shared read-only macOS warning.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC4 | tracking-state unit tests, complete and partial v1 migration fixtures, cache-fallback regression, configured-window assembly test, selection-preservation test, and clock-controlled reconciliation tests |
| AC5 | agent batch mutation tests and Swift optimistic queue tests |
| AC6-AC8, AC13 | shared-authority outside-lane filtering/restoration test, terminal terminology test, Swift presentation/grouping tests, Xcode build, and manual compact/detachable smoke tests |
| AC9 | Cobra command/alias tests and terminal golden coverage |
| AC10-AC11 | deterministic seamless animation phase tests, Reduce Motion inspection, agent call-counter regressions, and follow/unfollow fast-cadence gating through the real mutation path |
| AC12 | complete Go, race, Kit, Swift, Xcode, release, and diff-hygiene gates |
| AC14 | Swift presentation and AppState parking regressions, Xcode build, and compact/detached visual smoke tests |
| AC15 | Go transition, budget, persistence, remote-ref, and exact-PR confirmation tests; additive Swift decoding and shared-card presentation tests; live menu/window smoke tests |

## Reflection Notes

- Treating repository selection as explicit Following membership removes the
  surprising coupling between outside activity and user intent. Recent is an
  evidence category, not a mutation.
- The background agent's project-at-a-time scan path required a separate
  complete-inventory initialization boundary. Finalizing version-1 migration
  from a partial snapshot would have quietly moved later legacy repositories
  to Quiet.
- Keeping tracked/untracked protocol and JSON fields for one compatibility
  generation lets existing clients and scripts upgrade without preserving the
  old user-facing abstraction.
- Following is durable user intent, so an open PR or issue in a followed
  project cannot disappear merely because its last GitHub update crossed the
  recent-activity window. Explicit parking remains the lane-level way to hide
  it.
- Open remote work must outrank stale-dirty local classification. The persisted
  `Explicit` flag separates user Ignore intent from automatic parking, allowing
  reconciliation to repair affected lanes without reopening intentionally
  parked work.
- Dirty-work recency belongs to the durable material observation, not the last
  commit. This keeps current unstaged and untracked changes in Following while
  retaining bounded automatic parking for unchanged dirty work.
- The macOS Ignore label is deliberately lane-scoped: it reuses parking and
  cannot mutate the independent Following membership of the repository.
- Remembering the exact open PR before it disappears avoids broad merged-PR
  discovery. Git can reject still-published branches before the scarce GitHub
  confirmation, while a persisted negative outcome prevents repeated polling.

## Documentation Updates

- Update the README, constitution, and project progress summary for explicit
  Following membership, the outside-activity inbox, Quiet inventory, CLI
  compatibility aliases, and accessible wordmark motion.

## Delivery Decision

The user explicitly requested a new issue, branch, and pull request. Issue #9
is assigned to Jameson Stone. Branch `GH-9` starts exactly at current
`origin/main`; ready PR #10 targets `main`.

The July 15 open-PR visibility correction is delivered as a focused bug-fix
commit on assigned issue #27 and exact branch `GH-27`, within the user-approved
multi-focus ready pull request to `main`.

The followed-issue visibility and distinct lane-card identity follow-up is
delivered on assigned issue #31 and exact branch `GH-31` as a ready pull request
targeting `main`.

The bounded merged-checkout warning is added to the same user-approved issue
#31 / branch `GH-31` / ready PR #32 lane because it extends the shared Following
card and evidence refresh behavior already under review.

The July 16 dirty-worktree activity correction is assigned to issue #33 and
exact branch `GH-33`, created from current `origin/main`, for a ready pull
request targeting `main`.

## Evidence

- Pre-implementation recon found a clean `main` at `7e9d4c4`, exactly matching
  `origin/main` and the remote default branch.
- PR #8 was merged with passing Go and macOS checks; no open issue or PR lane
  remained.
- Issue #9 was created and assigned to Jameson Stone before source edits.
- Branch `GH-9` was created exactly from `origin/main` before source edits.
- Fresh-state, complete and partial version-1 migration, cache-fallback,
  configured recent-window, selection-preservation, no-reactivation,
  outside-lane filtering, follow-cadence, optimistic-queue, CLI alias, output,
  and agent probe tests cover the shared behavior.
- `make fmt-check vet test test-race build release-test` passes.
- `make macos-test macos-build` passes with 39 Swift tests and the bundled Go
  helper.
- `kit check --all` passes all 10 feature specifications.
- `git diff --check` passes.
- An independent read-only verifier found no remaining actionable issues after
  three repair passes, including ten consecutive cadence-regression runs.
- Ready PR [#10](https://github.com/jamesonstone/beacon/pull/10) is assigned to
  Jameson Stone, closes issue #9, and records the hosted CI evidence.
- A July 15 live reproduction confirmed `labcore-ui` remained one of 30
  followed projects while its open PR #31 was absent after crossing the
  six-hour recent window.
- Background batch and working-set regression tests now keep followed open PRs
  visible regardless of age and remove them after closure. The complete
  Go/race/macOS gate, Linux build, all 15 Kit specs, and diff hygiene pass.
- A rebuilt-agent refresh then rendered PR #31 as one Active `labcore-ui` lane
  in the same shared snapshot consumed by the CLI and both macOS surfaces.
- A later live `kit` reproduction found assigned issue #50 in the diagnostic
  snapshot but outside Following after its recent window elapsed. The shared
  working-set candidate rule now retains scoped open issue lanes just like open
  PR lanes, with regression coverage for visibility and closure.
- Swift presentation and shared-authority regressions verify that Ignore exists
  only in Following and sends the selected lane ID with state `parked`. The
  returned snapshot removes the lane from Active and exposes it in Parking Lot.
- All 75 Swift tests and the universal macOS build pass. Live checks at both the
  580-point detached width and 430-point compact width show Ignore at the
  far-right card edge; Parking Lot contains no Ignore actions.
- Transition, exact-command, three-candidate budget, cache migration,
  cold-start, non-followed, remote-present, non-merged, failure, and severity
  tests cover the merged-checkout advisory without broad GitHub discovery.
- The complete Go/race/release matrix, Linux amd64 build, all 76 Swift tests,
  universal macOS build, all 15 Kit feature checks, and diff hygiene pass.
- A July 16 live reproduction found six followed `lsmc-bio` PR lanes parked
  automatically with `explicit: false` because stale dirty local evidence won
  candidate precedence. The repaired candidate order and reconciliation restore
  those lanes while retaining explicit Ignore state.
- The rebuilt helper changes the live working set from 3 Active / 24 Parked to
  10 Active / 18 Parked, with all six affected PRs present in Active. The full
  Go/race/release/macOS gate, Linux amd64 build, all 16 Kit specifications, and
  diff hygiene pass; the macOS suite executes 81 tests with zero failures.
- The July 16 `terrarium` reproduction confirms a followed dirty base worktree
  with six unstaged files and one untracked file now appears Active `now` after
  the rebuilt helper starts. Regression coverage ties its card timestamp and
  freshness to durable material observation while preserving omitted private
  status hashes from cached startup snapshots.
- Issue #33 is assigned to Jameson Stone, and branch `GH-33` starts exactly at
  the freshly fetched `origin/main` head for ready pull-request delivery.
