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
  id: "0008"
  slug: github-api-budget
  dir: 0008-github-api-budget
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
    used_for: read-only collection, tracking authority, and background-agent boundaries
    status: active
skills: []
---

# GitHub API Budget

## Thesis

Beacon must not exhaust the authenticated user's GitHub API allowance while it
collects background evidence. It should reuse recent successful `gh` results,
reserve API capacity for interactive work, and make bulk tracking changes
without triggering network scans.

## Context

The background agent refreshes repositories independently. Source discovery and
each project refresh can issue GraphQL-backed `gh repo`, `gh pr`, `gh issue`,
and review-thread commands. With many configured source repositories, the same
metadata is requested repeatedly and can consume the user's shared GraphQL
quota. Tracking changes also perform unnecessary synchronous probes even though
the cached full snapshot already supplies a durable reactivation baseline.

## Clarifications

- Beacon continues to authenticate exclusively through `gh` and stores no
  credentials.
- Successful GitHub responses are persisted in the user-only Beacon cache so
  agent restarts do not trigger a complete repository-metadata warm-up.
- The cache may serve a stale successful response when GitHub capacity is below
  Beacon's reserve. Activity entries expire from stale fallback after 24 hours;
  repository metadata expires after 30 days.
- The default GraphQL reserve is 2,500 points, the Search reserve is 15
  requests, and the REST Core reserve is 1,500 requests.
- GitHub commands are serialized per rate bucket, conservatively debit 25
  GraphQL points before execution, and refresh authoritative rate state at most
  every 30 seconds and after at most five cache-miss commands.
- Repository metadata is stable enough to cache for seven days. Other GitHub
  evidence uses the configured remote refresh interval, set to 15 minutes for
  the user's installation.
- Discovery derives repository identity and default branch entirely from local
  Git remotes, remote HEAD, or conventional local base branches. GitHub evidence
  collection later verifies remote accessibility.
- Untracking uses the last complete cached snapshot. The next scheduled muted
  probe establishes its compact probe baseline without blocking the selection.

## Requirements

1. Share one guarded `gh` runner across discovery, full GitHub collection, and
   muted-project probes in the background agent.
2. Cache successful read-only `gh` responses by exact argument array and return
   copies so callers cannot mutate cached bytes.
3. Check the appropriate GraphQL, Search, or Core bucket before a cache miss;
   stop Beacon calls at the reserve and report the bucket and reset time.
4. Recheck rate state after observed rate-limit failures and avoid repeated
   calls until the reported reset.
5. Use stale successful cache entries while a bucket is protected instead of
   discarding otherwise healthy project evidence.
6. Support one atomic agent request for multiple Track or Untrack targets.
7. Remove synchronous compact probes from tracking acknowledgements; preserve
   automatic reactivation through the cached full-evidence baseline and the
   next scheduled probe.
8. Read project-management inventory from the running agent or its local cache
   before falling back to a direct scan.
9. Clear the user's current tracked selection and set the user's muted-project
   probe interval to one hour.
10. Persist successful response entries atomically under the user-only Beacon
    cache with no credentials, tokens, or authorization headers.
11. Serialize cache-miss execution per rate bucket and use conservative local
    cost estimates so concurrent collection cannot overshoot the reserve.
12. Use only local Git remote identity and branch metadata during source
    discovery; source walking must issue no GitHub API call.
13. A tracking mutation must advance the cached project revision so an older
    in-flight scan cannot overwrite the user's new selection.
14. Explicit untracking with incomplete evidence must persist an untracked
    entry with a pending baseline; the first later complete collection records
    that baseline without reactivating the project.
15. Add top-level `beacon select` as the colored searchable interactive project
    selector. Existing tracked projects start highlighted, Space toggles the
    selection, confirmation writes through the agent, and macOS receives the
    same tracking-change snapshot.
16. Apply durable tracking state to cached records when the agent starts, before
    its scheduler decides which projects need full refreshes.

## Assumptions

- GitHub's `/rate_limit` REST response is the authoritative bucket snapshot.
- A 25-point GraphQL debit is intentionally conservative for Beacon's current
  queries; the five-command refresh bound corrects remaining drift.
- A one-hour muted probe interval is appropriate for the user's current set of
  many intentionally quiet projects while still allowing automatic revival.

## Acceptance Criteria

- [x] AC1: Repeated identical `gh` evidence calls within the cache interval run
  the delegate once.
- [x] AC2: Beacon declines new GraphQL/Search/Core work at its reserve and the
  error includes the reset time.
- [x] AC3: A stale successful entry is returned when a protected bucket cannot
  accept a refresh.
- [x] AC4: Twenty or more projects can be untracked with one agent mutation and
  no GitHub command or synchronous probe.
- [x] AC5: Project-management commands prefer agent/cache inventory and retain
  the direct-scan fallback only when no cached inventory exists.
- [x] AC6: The user's tracking state contains every discovered project and the
  live agent snapshot reports zero tracked projects.
- [x] AC7: Existing automatic reactivation, Go, race, Kit, Swift, and packaging
  validation remains green.
- [x] AC8: Successful cache entries survive a new runner process, remain
  user-only, and can provide bounded stale fallback without executing `gh`.
- [x] AC9: Concurrent GraphQL cache misses execute serially and cannot cross the
  2,500-point reserve based on Beacon's conservative local budget.
- [x] AC10: Source discovery with a local remote HEAD performs no GitHub command.
- [x] AC11: Untracking during an active scan remains untracked after that older
  scan completes.
- [x] AC12: A project with collection errors can be explicitly untracked,
  remains untracked, and initializes its baseline without reactivation after
  evidence recovers.
- [x] AC13: `beacon select` requires a TTY, starts from current tracked state,
  applies the confirmed selection through shared tracking authority, and keeps
  `beacon projects` compatibility.
- [x] AC14: An agent restart immediately reports and schedules from the durable
  tracked/untracked selection without requiring a fresh scan.

## Implementation Plan

1. Replace process-only cache storage with atomic user-only persistent entries
   and bounded stale fallback.
2. Serialize budgeted GitHub cache misses, strengthen reserves, and apply
   conservative local cost estimates with frequent authoritative refreshes.
3. Use local Git metadata exclusively during source discovery; GitHub
   collection verifies repository access later.
4. Increase the user's remote cache interval, restart the agent, and verify
   quota stability plus the existing tracking selection.
5. Reject scan results superseded by newer tracking revisions.
6. Add the top-level colored interactive selector and pending-baseline
   semantics for explicit choices during evidence errors.
7. Reconcile durable tracking state into cache records before scheduling.
8. Update project docs and deliver on the active PR.

## Task Checklist

- [x] T1: Implement atomic persistent caching and bounded stale fallback.
- [x] T2: Serialize and conservatively budget GitHub cache misses.
- [x] T3: Implement local-first source discovery.
- [x] T4: Apply the user's longer cache interval and verify live behavior.
- [x] T5: Preserve tracking mutations against older in-flight scans.
- [x] T6: Add `beacon select` and pending-baseline reconciliation.
- [x] T7: Apply durable tracking state during agent startup.
- [x] T8: Update documentation and complete validation.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC3 | guarded-runner unit tests with deterministic quota fixtures and stale cache |
| AC4-AC5 | agent protocol/engine and CLI project-command tests with call counters |
| AC6 | local agent snapshot plus managed tracking-state inspection |
| AC7 | complete Go/race, Xcode, Kit, and release validation |
| AC8 | restart-style runner test using one shared temporary disk cache |
| AC9 | concurrent guarded-runner test with delegate concurrency and reserve assertions |
| AC10 | discovery fixture with local remote HEAD and a runner that rejects `gh` |
| AC11 | blocked-scan engine test that mutates tracking before scan release |
| AC12 | tracking-manager recovery test with incomplete then complete evidence |
| AC13 | Cobra selector tests for shared persistence and non-TTY guidance |
| AC14 | engine-construction test with tracked cache and durable untracked state |

## Reflection Notes

The largest avoidable cost was not PR detail itself but repeated source
discovery: each scheduler cycle re-ran GraphQL-backed repository metadata for
every source checkout. Local-only discovery removes that cost completely, while
a shared persistent runner makes the seven-day repository cache effective
across full scans and muted probes. A tracking revision guard was also required:
without it, an older scan could finish after an explicit selection and restore
the stale tracking value.

## Documentation Updates

- Document the shared API reserve/cache behavior and muted probe tuning in the
  README, constitution, and project progress summary.

## Delivery Decision

Continue on issue #3, branch `GH-3`, and ready PR #4 as requested.

## Evidence

- Guarded-runner tests verified exact-command caching, copied result bytes,
  GraphQL reserve enforcement with reset context, stale-success fallback, and
  non-GitHub pass-through.
- A 25-project engine test verified one selection persisted 24 muted projects,
  issued zero probes, and deferred their first compact probe.
- CLI coverage verified project mutation uses local cached inventory when the
  agent socket is unavailable and never invokes an external scanner.
- The live agent schema-v2 state reports zero tracked and 80 untracked projects.
  Three projects with incomplete cached evidence retain pending baselines and
  remain untracked until a later complete probe initializes those baselines.
- `/Users/jamesonstone/.config/beacon/config.yaml` now uses
  `remote_refresh_interval: 15m0s` and `untracked_probe_interval: 1h0m0s`; the
  managed state contains all 80 discovered projects as untracked.
- `beacon select` is covered as a TTY-only colored multi-select whose confirmed
  choice is written through the background agent and immediately reflected in
  the shared snapshot consumed by macOS.
- `make fmt-check vet test test-race build release-test` passed.
- `kit check --all` passed for all eight features.
- `make macos-test` passed 31 tests.
- Release packaging passed for four CLI archives and the universal macOS app,
  including embedded helper, login item, icon, version metadata, and signing
  checks.
