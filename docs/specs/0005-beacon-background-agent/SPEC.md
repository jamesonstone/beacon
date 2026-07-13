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
  id: "0005"
  slug: beacon-background-agent
  dir: 0005-beacon-background-agent
relationships:
  - type: builds_on
    target: 0004-project-tracking
references:
  - id: issue-3
    name: Add managed project tracking and automatic reactivation
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/3
    relation: implements
    read_policy: must
    used_for: user-selected delivery lane
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: evidence, read-only, CLI authority, and macOS boundaries
    status: active
skills: []
---

# Beacon Background Agent

## Thesis

Beacon should render durable cached project state immediately and move expensive Git and GitHub collection into one persistent, user-scoped background agent. The agent owns scheduling, caching, incremental events, and project reactivation; CLI and Swift clients remain presentation surfaces over the same Go authority.

## Context

Direct scans are correct but make every invocation pay discovery, Git, and GitHub latency before showing useful state. Project tracking already records durable evidence baselines, but untracked projects still receive full synchronous collection. A single bounded background process can preserve last-good state, coalesce refreshes, update projects independently, and probe muted projects less frequently without introducing one process per repository.

The attached plan requested feature ID `0004`; that ID already belongs to `0004-project-tracking`, so this canonical feature uses the next collision-free ID, `0005`.

## Clarifications

- The existing `beacon` executable hosts the agent; no `beacond` binary is introduced.
- The agent protocol is version 1 newline-delimited JSON over a user-only Unix-domain socket. Snapshot schema remains version 2.
- User-facing `Untracked` maps to internal state `muted`; `ignored` is accepted by the storage model but is not exposed as a user action in this feature.
- Operational state follows the attached paths: `$HOME/.local/state/beacon/tracking.json`, `$HOME/.cache/beacon/projects/`, and `$HOME/.cache/beacon/agent.sock`.
- Existing sibling `tracking.yaml` data is migrated once, atomically, without changing `config.yaml`.
- Cache corruption is isolated per file: the bad file is quarantined with a `.corrupt-<timestamp>` suffix and healthy cached projects remain usable.
- Warm clients never wait for repository collection before receiving a snapshot. A cold cache may initially contain no lanes; the agent then publishes discovery and stage events.
- `beacon scan` and `beacon scan --json` remain blocking, complete, deterministic direct-scan compatibility paths.
- Bare `beacon` requires the agent, renders cache immediately, requests a non-blocking refresh, and returns without waiting for discovery or collection. `--no-watch` renders cache only.
- Non-TTY bare output uses the same cache-first behavior and never emits cursor control or incremental frames.
- The scheduler runs logical per-project jobs with bounded concurrency, one active job per project, and duplicate request coalescing.
- Tracked jobs collect complete evidence. Muted jobs run a local and remote summary probe; a material delta queues a complete scan and durable reactivation.
- Fetch timestamps, scan timestamps, freshness, and derived actions are excluded from probe fingerprints.
- macOS connects through a Swift actor, publishes view state only on `@MainActor`, reconnects after disconnects, and retains the last good snapshot.
- LaunchAgent installation is explicit and user-scoped. `beacon init` offers installation only in an interactive terminal.

## Public Contracts

### Commands

```text
beacon [--no-watch]
beacon scan [--repo NAME] [--json] [--no-refresh]
beacon refresh [project]
beacon track <project>...
beacon untrack <project>...
beacon projects [--tracked|--untracked]
beacon agent install|serve|status|stop|uninstall
```

Existing `beacon projects track|untrack` commands remain compatibility aliases.

### Configuration

```yaml
settings:
  max_parallel: 4
  tracked_refresh_interval: 1m
  untracked_probe_interval: 10m
```

`tracked_refresh_interval` defaults to existing `scan_interval` when omitted. `untracked_probe_interval` defaults to ten minutes. Version-1 and existing version-2 files remain readable.

### Protocol

Each request and event contains `protocol_version: 1`. Requests contain a unique `request_id`, a `type`, and optional `project_id`. Events contain `type`, `scan_id`, `project_id`, monotonically increasing project `revision`, `stage`, and `generated_at`. Supported requests are `get_snapshot`, `subscribe`, `refresh_all`, `refresh_project`, `set_tracking_state`, `list_projects`, and `get_agent_status`. Supported events are `snapshot`, `project_discovered`, `project_queued`, `project_local_ready`, `project_updated`, `project_failed`, `tracking_changed`, `project_reactivated`, `scan_completed`, and `heartbeat`.

Clients discard project events older than their accepted revision and treat malformed or unsupported protocol envelopes as connection errors rather than corrupting last-good state.

### Stage Vocabulary

```text
cached
queued
local
github
ready
failed
```

No final action is derived from incomplete `queued`, `local`, or `github` evidence.

## Requirements

1. Add strict operational path resolution with user-only directories, a JSON tracking-state store, legacy YAML migration, and per-project atomic cache files with corruption quarantine.
2. Add protocol-v1 request/event models, NDJSON codec, Unix socket server/client, subscriptions, heartbeats, revision monotonicity, and malformed-message isolation.
3. Add a bounded scheduler with per-project exclusion, duplicate coalescing, cancellation, tracked refresh cadence, muted probe cadence, and last-good result preservation.
4. Add stable project IDs based on GitHub `owner/repo`, deterministic cache inventory, and project-scoped snapshot merging that preserves schema-v2 ordering and summary/group semantics.
5. Add lightweight muted probes covering local HEAD/status/upstream evidence and scoped GitHub PR/issue summaries, with material fingerprint comparison and reasoned durable reactivation.
6. Add `agent install|serve|status|stop|uninstall`, a user LaunchAgent plist, PID/socket locking, user-only socket/cache/state permissions, and bounded rotating logs.
7. Make bare CLI cache-first with non-blocking refresh requests, add `--no-watch` and `refresh`, add root `track`/`untrack` aliases, and preserve blocking deterministic scan/JSON behavior.
8. Replace macOS subprocess polling with a Swift actor agent client, cached startup, incremental project stages, reconnect behavior, and last-good retention.
9. Add primary Tracked and Untracked menu tabs, stage badges, muted timestamps/probe timestamps, search/multi-selection, manual restoration, and automatic-reactivation reasons.
10. Offer agent installation during interactive init, show an Enable Background Agent action when missing, and keep all repository and GitHub operations read-only.
11. Package agent lifecycle resources with release artifacts and document installation, troubleshooting, cache reset, logs, stop, and uninstall.
12. Preserve partial results, deterministic complete snapshots, non-TTY purity, source/config compatibility, release versioning, and the existing PR delivery lane.

## Assumptions

- Beacon runs one background agent per macOS user account.
- `git` and an authenticated `gh` remain available to the LaunchAgent without Beacon storing credentials.
- Cached evidence may be stale between refreshes, so clients retain and display the snapshot generation time and active scan stage.
- Missing or corrupt per-project cache entries can be rebuilt from configured repository discovery without changing repository or GitHub state.
- Operational state and cache files are local to one machine and are not synchronized between hosts.

## Acceptance Criteria

- [x] AC1: A warm CLI and menu launch display cached projects without waiting for discovery, Git, or GitHub.
- [x] AC2: Tracked projects update independently and scheduler concurrency never exceeds the configured limit.
- [x] AC3: Duplicate refreshes coalesce and no project has overlapping jobs.
- [x] AC4: Muting records a stable baseline, unchanged probes remain muted, and every supported material Git/GitHub delta durably reactivates with a reason.
- [x] AC5: Muted projects use the lightweight probe path and do not run complete collection until a delta is detected.
- [x] AC6: Protocol revisions and scan IDs prevent stale or out-of-order events from replacing newer client state.
- [x] AC7: One cache, project, socket, or collection failure preserves healthy cached/project results and last-good UI state.
- [x] AC8: `beacon scan --json` remains ANSI-free, deterministic, blocking, and complete without requiring the agent.
- [x] AC9: Bare execution renders cache and queues refresh without waiting; `--no-watch` returns cache without requesting work, and non-TTY output contains no incremental control output.
- [x] AC10: Agent install/status/stop/uninstall and duplicate-process prevention work with user-only state, socket, PID, plist, and rotated logs.
- [x] AC11: CLI and macOS clients consume the same agent snapshot/events; Swift transport runs in an actor and UI mutation remains on `@MainActor`.
- [x] AC12: No agent operation changes tracked files, branches, commits, remotes, pull requests, reviews, or issues.
- [x] AC13: Documentation, packaging, isolated-HOME smoke evidence, Go race tests, and Xcode tests cover the supported lifecycle.

## Implementation Plan

1. Add operational paths, JSON state/cache stores, legacy migration, and canonical models.
2. Add protocol codec, socket client/server, scheduler, event hub, and lifecycle lock.
3. Wire direct scan/probe functions into the agent and cache/project snapshot merger.
4. Add lifecycle, refresh, root tracking aliases, cache-first root behavior, and init offer.
5. Add Swift actor transport, incremental AppState handling, tab/stage UI, and enable-agent action.
6. Add LaunchAgent resources, packaging, docs, validation, reflection, and existing-PR delivery.

## Task Checklist

- [x] T1: Implement and test operational paths, state migration, and caches.
- [x] T2: Implement and test protocol, socket transport, scheduler, revisions, and coalescing.
- [x] T3: Implement and test tracked collection, muted probes, cache merging, and reactivation.
- [x] T4: Implement and test CLI lifecycle, refresh, aliases, cache-first, and compatibility behavior.
- [x] T5: Implement and test Swift actor transport, incremental state, tabs, and lifecycle UI.
- [x] T6: Update LaunchAgent packaging, README, constitution, progress summary, and PR metadata.
- [x] T7: Complete full validation, reflection, evidence, and delivery to PR #4.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1, AC7, AC9 | cache-first CLI integration tests, corrupt-cache fixtures, and isolated-HOME timing/order smoke |
| AC2, AC3 | scheduler table tests with blocking fake jobs and concurrency counters |
| AC4, AC5 | probe fingerprint tables and muted-to-tracked integration tests with full-scan call counters |
| AC6 | protocol codec/socket tests plus Swift out-of-order fixture tests |
| AC8 | existing and new JSON golden/stdout-purity tests |
| AC10 | lifecycle/plist/lock tests and isolated-HOME install/status/stop/uninstall smoke |
| AC11 | Swift actor/AppState tests and Xcode Debug test/build |
| AC12 | argument-array runner tests, temporary repositories, and mutation inspection |
| AC13 | full Go, race, build, release, Xcode, Kit, docs, and PR checks |

## Reflection Notes

The smallest reliable architecture remained one Go authority and two thin
clients. Per-project cache records make partial failure and immediate startup
straightforward, while one versioned NDJSON socket avoids coupling protocol
evolution to snapshot schema changes. Keeping muted probes separate from full
collection was also essential: the saved probe baseline prevents existing
dirty work from reactivating immediately, while a changed probe only promotes a
project after complete evidence confirms a material delta.

Two hardening details emerged during validation. Snapshot-bearing terminal
events must displace older buffered events so a slow subscriber cannot miss
`scan_completed`, and the Xcode build must sign the nested Go helper before the
containing app is signed. Both behaviors are now covered by tests or the signed
Xcode build.

## Documentation Updates

- Update `docs/CONSTITUTION.md` for agent ownership, operational writes, cache/protocol boundaries, and read-only execution.
- Update `docs/PROJECT_PROGRESS_SUMMARY.md` for feature 0005.
- Update `README.md` for lifecycle, cache-first usage, state paths, troubleshooting, and uninstall.

## Delivery Decision

Per the user's explicit instruction, extend assigned issue #3, branch `GH-3`, and ready PR #4 rather than creating a new delivery lane.

## Evidence

- `make fmt-check vet test test-race build release-test` passed after the final
  Go implementation and scheduler hardening.
- `xcodebuild -project macos/Beacon/Beacon.xcodeproj -scheme Beacon
  -configuration Debug test -quiet` passed with app signing enabled; the nested
  universal Go helper was signed successfully before the app bundle.
- `make macos-test` passed the unsigned CI-equivalent Debug build and Swift
  suite.
- An isolated `/tmp` home demonstrated foreground serve, status, immediate
  cache-only rendering, explicit refresh, direct scan/JSON parity, and graceful
  stop without touching the user's real configuration or runtime state.
- The isolated lifecycle rejected a duplicate agent, created state/cache/PID/
  socket files with user-only permissions, muted `jamesonstone/beacon` without
  immediate reactivation, then restored it with durable reason `new local
  changes` after a controlled worktree edit.
- Isolated `beacon doctor` passed `git`, `gh`, authentication, configuration,
  state, agent, repository, and GitHub checks. `beacon scan --json | jq` reported
  schema version 2, one project, and zero errors.
- Unit coverage includes strict config intervals, JSON state migration, cache
  quarantine, lifecycle files, duplicate PID locking, bounded scheduling,
  duplicate and distinct refresh handling, last-good preservation, muted
  probes, scoped GitHub summaries, protocol validation, slow subscribers,
  placeholder discovery, and monotonic retry revisions.
- Swift coverage includes protocol decoding, cache socket resolution,
  out-of-order revision and scan rejection, cold-cache placeholders,
  reconnect/error handling, and last-good snapshot retention.
