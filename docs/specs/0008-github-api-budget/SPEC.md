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
- A process-local cache is sufficient; v1 does not add a persistent GitHub
  response database.
- The cache may serve a stale successful response when GitHub capacity is below
  Beacon's reserve. Staleness remains bounded to the lifetime of the agent.
- The default GraphQL reserve is 1,000 points, the Search reserve is five
  requests, and the REST Core reserve is 500 requests.
- Rate state is refreshed at most every 30 seconds and after at most 20
  cache-miss commands so concurrent work cannot rely indefinitely on an old
  allowance reading.
- Repository metadata is stable enough to cache for 24 hours. Other GitHub
  evidence uses the configured five-minute remote refresh interval.
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
9. Keep only `lsmc-bio/labcore` tracked in the user's current state and set the
   user's muted-project probe interval to one hour.

## Assumptions

- GitHub's `/rate_limit` REST response is the authoritative bucket snapshot.
- One locally reserved point per cache-miss command is a conservative estimate
  between authoritative refreshes; the 20-command refresh bound corrects drift.
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
- [x] AC6: The user's tracking state contains every discovered project except
  `lsmc-bio/labcore`, and the agent snapshot reports only that project tracked.
- [x] AC7: Existing automatic reactivation, Go, race, Kit, Swift, and packaging
  validation remains green.

## Implementation Plan

1. Add the API-budget runner and table-driven cache/budget tests.
2. Wire one runner through agent discovery, full scans, and muted probes.
3. Add batch tracking protocol and engine support without synchronous probing.
4. Make CLI project inventory agent/cache-first and use batch requests.
5. Update the user's tracking state and probe interval, restart the agent, and
   verify the resulting snapshot.
6. Update project docs and deliver on the active PR.

## Task Checklist

- [x] T1: Implement guarded cached GitHub execution.
- [x] T2: Implement atomic batch tracking and cached project inventory.
- [x] T3: Apply and verify the user's tracking selection.
- [x] T4: Update documentation and complete validation.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC3 | guarded-runner unit tests with deterministic quota fixtures and stale cache |
| AC4-AC5 | agent protocol/engine and CLI project-command tests with call counters |
| AC6 | local agent snapshot plus managed tracking-state inspection |
| AC7 | complete Go/race, Xcode, Kit, and release validation |

## Reflection Notes

The largest avoidable cost was not PR detail itself but repeated source
discovery: each scheduler cycle re-ran GraphQL-backed repository metadata for
every source checkout. A single shared runner makes the 24-hour repository
cache effective across discovery, scans, and probes. Refreshing the baseline on
an explicit repeated untrack was also necessary; otherwise changed projects
could correctly auto-reactivate immediately after the user reaffirmed the
untracked selection.

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
- The live batch command muted 78 projects in under one second. Agent schema-v2
  state reports exactly one tracked project, `lsmc-bio/labcore`, at
  `/Users/jamesonstone/go/src/github.com/lsmc-bio/labcore`.
- `/Users/jamesonstone/.config/beacon/config.yaml` now uses
  `untracked_probe_interval: 1h0m0s`; the managed state contains 78 muted
  projects and excludes `lsmc-bio/labcore`.
- `make fmt-check vet test test-race build release-test` passed.
- `kit check --all` passed for all eight features.
- `make macos-test` passed 31 tests.
- Release packaging passed for four CLI archives and the universal macOS app,
  including embedded helper, login item, icon, version metadata, and signing
  checks.
