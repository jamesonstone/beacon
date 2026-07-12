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

- Authored open PRs enter the working set only when recently active or pinned.
- Design, research, and planning work without Git evidence uses a manually
  created lane; no Codex task API is required.
- GitHub evidence may be 30–60 minutes stale. Local evidence remains
  near-real-time and explicit Refresh remains available.
- Notes are short local memory cues. Evidence remains canonical, and the UI
  identifies notes written before a later evidence change.
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
   branches, recent local commits, recent authored PR activity, pins, and
   manual lanes. Old inactive authored PRs remain out unless pinned.
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
    assigned-issue search, filter to discovered repositories, and enrich only
    active/recent/pinned PRs. Batch PR enrichment when practical.
15. Default remote evidence age is 45 minutes. Exact cache age is exposed;
    existing generous rate reserves remain enforced.
16. Parked and inactive remote work receives no review-thread/check enrichment
    in the default path. Full enrichment remains available through explicit
    diagnostics or one-lane refresh.
17. Migrate existing project tracking intent without deleting config or state.
    Existing muted project lanes begin parked; explicit configuration remains
    intact and reversible.
18. Agents are never required to write Beacon, Kit, repository files, or an
    external task system for working-set correctness.

## Assumptions

- Strict JSON remains sufficient because Beacon stores only current and
  last-seen observations, not an event history.
- Seven days is the default recent-activity window unless durable product
  evidence establishes a better value.
- An additive protocol command set is preferable to a second local control
  channel.
- Stacking this branch on `GH-3` is required until PR #4 lands because this
  feature intentionally reuses its background-agent and API-budget work.

## Acceptance Criteria

- [x] AC1: A configuration with many repositories renders only the small lane
  working set by default.
- [x] AC2: Two worktrees in one repository have independent attention, note,
  seen, and park state.
- [x] AC3: A parked lane ignores unrelated repository activity and resumes only
  for its own material delta or explicit action.
- [x] AC4: A recent authored PR appears while an inactive authored PR remains
  absent unless pinned.
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

## Task Checklist

- [x] T1: Create issue #5, branch `GH-5`, and canonical feature spec.
- [x] T2: Implement lane state, migration, observations, and deltas.
- [x] T3: Implement additive agent protocol and CLI lane commands.
- [x] T4: Implement conservative local/remote working-set collection.
- [x] T5: Implement CLI and macOS primary working-set views.
- [x] T6: Reconcile docs and complete validation.
- [x] T7: Commit, push, and create the ready stacked PR.

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

## Reflection Notes

- Lane attention must override legacy project tracking for scheduler cadence;
  otherwise resuming a lane inside an untracked project inherits the old slow
  project probe interval.
- Automatic candidate entries are removed when they become inactive so hidden
  historical work cannot retain the fast cadence. Notes, pins, explicit
  resume, and parking make attention durable.
- Pinned inactive remote PRs remain represented from local lane state even when
  default recent-only GitHub enrichment omits them. Explicit lane refresh can
  restore current remote detail.
- Schema upgrades must migrate last-good per-project caches in memory. Rejecting
  schema-v2 cache files as corrupt forced an unnecessary fleet-wide startup
  rebuild and erased the cache-first experience this feature depends on.
- The simplest conservative remote policy is constant-cost discovery: the
  default scope always uses two global searches, filters locally, and enriches
  only recent matches. Explicit diagnostics opt into inactive PR enrichment.

## Documentation Updates

- Update the constitution, README, and project progress summary for lane-level
  attention, optional notes, factual deltas, conservative collection, and the
  explicit diagnostic boundary.

## Delivery Decision

Create a distinct issue/branch/PR lane as requested. Issue #5 is assigned to
Jameson Stone. Branch `GH-5` starts at `origin/GH-3`, and its PR targets `GH-3`
until prerequisite PR #4 lands.

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
