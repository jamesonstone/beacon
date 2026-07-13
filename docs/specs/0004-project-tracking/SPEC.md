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
  id: "0004"
  slug: project-tracking
  dir: 0004-project-tracking
relationships:
  - type: builds_on
    target: 0002-beacon-init-dashboard
references:
  - id: issue-3
    name: Add managed project tracking and automatic reactivation
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/3
    relation: implements
    read_policy: must
    used_for: delivery lane and acceptance scope
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: CLI authority, snapshot, state, and macOS boundaries
    status: active
skills: []
---

# Managed Project Tracking

## Thesis

Beacon should let users curate the projects competing for attention without losing the ability to notice resumed work. A deselected project becomes untracked inventory, retains a baseline of its current durable evidence, and automatically returns to tracked views when later Git or GitHub evidence differs from that baseline.

## Context

Persistent source roots intentionally rediscover every GitHub repository beneath them, so removing one repository from the declarative configuration is not a durable exclusion. The active-first dashboard reduces idle noise but does not let the user intentionally set aside a stale project. This feature adds a Beacon-managed tracking state separate from source configuration, project-management controls in both clients, and automatic reactivation owned by the Go CLI.

Post-delivery evolution: `0005-beacon-background-agent` migrates the sibling YAML state described here to user-scoped JSON at `$HOME/.local/state/beacon/tracking.json`, retains the same baseline semantics, and adds muted-project probes and reactivation reasons. This artifact remains the historical source for the original project-tracking behavior.

## Clarifications

- Declarative discovery remains in `config.yaml`; user tracking choices live in an atomic sibling `tracking.yaml`. With the default config, the path is `$HOME/.config/beacon/tracking.yaml`.
- A custom `--config` or `BEACON_CONFIG` keeps its tracking state beside that resolved configuration file.
- Any discovered project may be untracked, including one with current activity. Its current evidence becomes the baseline, so unchanged pre-existing activity does not immediately reverse the choice.
- Every scan still collects Git and scoped GitHub evidence for untracked projects. V1 does not optimize this into a reduced-cost probe.
- Local evidence includes lane/worktree identity, head and upstream state, publication/base comparisons, and staged, unstaged, untracked, and conflict counts. GitHub evidence includes issues, PR head/state/update time, checks, reviews, feedback, merge state, and linked issues.
- Time-derived freshness and recommended actions are excluded from fingerprints so the passage of time alone cannot reactivate a project.
- If any baseline evidence changes, Beacon atomically removes the untracked entry before publishing the snapshot. The project remains tracked after the triggering work later finishes.
- Manual tracking removes the untracked entry. Manual untracking records a new baseline. Repeating either operation is idempotent.
- Automatic state-file writes are the only new scan-time mutation. Git repositories, branches, remotes, issues, PRs, and `config.yaml` remain untouched.
- The Go CLI owns persistence, fingerprints, reconciliation, and mutations. The Swift app invokes CLI commands and renders snapshot state without duplicating policy.

## Requirements

1. Add strict tracking-state schema version 1 with deterministic atomic loading/writing, duplicate rejection, path resolution beside the active config, and no file creation until a project is explicitly untracked.
2. Add deterministic project evidence fingerprints that ignore time-derived policy while detecting local lane/status/head changes and GitHub issue, PR, check, review, feedback, and merge changes.
3. Reconcile tracking state after every completed collection: unchanged baselines remain untracked; changed baselines are removed and reported as automatically reactivated.
4. Keep every project and lane in schema-v2 JSON while preserving the existing total `projects` count and adding project tracking state, untracked lane grouping, explicit tracked/untracked project counts, tracking-file metadata, and reactivation evidence.
5. Exclude untracked projects and their repository-scoped diagnostics from ready, action, waiting, idle, top-item, and human summary views while preserving all evidence in JSON. Human dashboards show one compact untracked-project count and a command hint; global diagnostics remain visible.
6. Add `beacon projects` as an interactive multi-select management mode plus deterministic non-interactive `track`, `untrack`, and untracked-list behavior using stable `owner/repo` identity with unique configured-name aliases.
7. Add a macOS project-management mode with Tracked and Untracked tabs, search, explicit track/untrack controls, automatic refresh after mutations, and clear mutation errors/loading state.
8. Preserve strict config v1/v2 compatibility, existing discovery semantics, bounded command execution, partial scan results, JSON stdout purity, and the macOS 14 deployment target.
9. Document state ownership, automatic reactivation, CLI usage, app behavior, failure handling, and the expanded mutation boundary.

## Assumptions

- GitHub `owner/repo` is the canonical persistent project identity.
- Repository names are accepted by CLI mutation commands only when they resolve uniquely in the current snapshot.
- Untracked entries whose repositories are temporarily undiscoverable remain in `tracking.yaml` and reappear in management views once discovery resumes.
- A scan that detects reactivation but cannot persist the state change fails rather than publishing a non-durable tracked result.
- State-file edits are local user actions and never trigger GitHub writes.

## Acceptance Criteria

- [x] AC1: Missing tracking state behaves as an empty version-1 state without creating a file; valid state round-trips atomically and strict invalid/duplicate input fails clearly.
- [x] AC2: Evidence fingerprints are deterministic across ordering and clock changes, remain stable for unchanged evidence, and change for every supported Git or GitHub activity class.
- [x] AC3: Manual untracking records the current baseline and remains untracked on the next unchanged scan; manual tracking is persistent and both operations are idempotent.
- [x] AC4: New local or GitHub evidence automatically and permanently restores an untracked project, with the project ID recorded in snapshot reactivation metadata.
- [x] AC5: JSON retains all discovered projects and lanes while groups and summary counts separate tracked and untracked work without null collections or ANSI output.
- [x] AC6: Bare and human scan output omit untracked work, show a compact count and management hint, and top-item selection never opens an untracked lane.
- [x] AC7: `beacon projects` supports interactive selection, searchable untracked viewing, explicit track/untrack commands, non-TTY guidance, ambiguous/not-found errors, and stdout-safe JSON where offered.
- [x] AC8: The macOS management mode moves a deselected project from Tracked to Untracked, restores it manually, reflects automatic reactivation, preserves last-good scans, and delegates every mutation to the helper CLI.
- [x] AC9: Existing config, discovery, scan, dashboard, loader, release, and menu-bar-count tests remain green; the menu-bar count excludes untracked work.
- [x] AC10: Documentation and live isolated-HOME evidence demonstrate that `config.yaml` is unchanged while `tracking.yaml` is created, updated, and automatically reconciled.

## Implementation Plan

1. Add the tracking store, evidence fingerprint, model additions, reconciliation service, and focused unit tests.
2. Wrap the shared scan path with tracking reconciliation and update human/JSON output without moving policy into clients.
3. Add CLI project-management commands and Huh multi-selection behind injectable prompt/store boundaries.
4. Add Swift models, CLI mutation client, AppState transitions, and a separate searchable project-management view.
5. Update constitution, progress summary, README, command documentation, and doctor checks.
6. Run focused, race, integration, isolated-HOME, CLI, Swift, workflow, Kit, and live smoke validation before delivery.

## Task Checklist

- [x] T1: Implement and test tracking-state storage and path resolution.
- [x] T2: Implement and test evidence fingerprints and automatic reconciliation.
- [x] T3: Implement and test snapshot grouping, counts, terminal output, and top-item filtering.
- [x] T4: Implement and test interactive and explicit CLI management.
- [x] T5: Implement and test macOS tracked/untracked management.
- [x] T6: Update public and canonical documentation.
- [x] T7: Complete validation, reflection, evidence, and prepare the ready PR delivery.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1 | Strict loader/writer/path table tests and atomic-write failure tests |
| AC2 | Table-driven fingerprint tests with reordered lanes and individual evidence changes |
| AC3-AC4 | Reconciliation and temporary-repository integration tests with persisted reloads |
| AC5-AC6 | JSON and terminal goldens, summary/group assertions, and top-item tests |
| AC7 | Cobra and prompt tests covering TTY, non-TTY, selection, aliases, and errors |
| AC8-AC9 | Swift decoding, client, AppState, grouping, mutation, and Xcode tests |
| AC10 | Isolated-HOME CLI smoke test plus config/tracking file comparison |

## Reflection Notes

- Keeping attention state separate from discovery preserves source roots as a declarative inventory while allowing the active views to stay small.
- The fingerprint must be evidence-focused: freshness, derived actions, and scan clocks would otherwise reactivate projects without real work.
- Missing evidence is not evidence of change. Global and repository-scoped collection failures therefore suspend baseline creation and comparison instead of causing false reactivation.
- Preserving the existing schema-v2 `projects` total and adding explicit `tracked_projects` and `untracked_projects` counts avoids silently changing an established field's meaning.
- Swift only selects and invokes helper commands; serializing mutations in `AppState` prevents concurrent tracking-file updates without moving persistence rules into the UI.

## Documentation Updates

- Update `docs/CONSTITUTION.md` for the managed tracking state and limited scan-time mutation boundary.
- Update `docs/PROJECT_PROGRESS_SUMMARY.md` for feature 0004.
- Update `README.md` with CLI and macOS project-management workflows and automatic reactivation semantics.

## Delivery Decision

Deliver through assigned issue `#3`, branch `GH-3`, and a new ready pull request targeting `main`, as explicitly requested by the user.

## Evidence

- `make fmt-check` passed.
- `go vet ./...` passed.
- `go test ./... -count=1` passed across CLI, configuration, discovery, Git, GitHub, output, policy, progress, scan, and tracking packages.
- `go test -race ./... -count=1` passed.
- `go build -o bin/beacon ./cmd/beacon` passed.
- `xcodebuild -project macos/Beacon/Beacon.xcodeproj -scheme Beacon -configuration Debug -destination 'platform=macOS' -derivedDataPath /tmp/beacon-gh3-derived CODE_SIGNING_ALLOWED=NO test -quiet` passed.
- `make release-test` passed with `semantic version tests passed`.
- `kit check --project` passed with a coherent project contract.
- `git diff --check` passed.
- An isolated-HOME smoke test initialized a temporary configuration, untracked `jamesonstone/beacon`, retained that state on an unchanged scan, created a new local commit in a temporary clone, and observed `tracking.auto_reactivated: ["jamesonstone/beacon"]`, `tracking_state: tracked`, and an empty persisted `untracked` list on the next scan.
- The isolated smoke test recorded the same SHA-256 for `config.yaml` before untracking and after automatic reactivation: `c2a507921b8a3d5b24bc0a125437e9b47cff4b23421ccfdb13b7441745e2e095`.
