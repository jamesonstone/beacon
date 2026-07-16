---
kit_metadata_version: 1
artifact: spec
workflow_version: 2
phase: implement
delivery_intent: existing_ready_pull_request
clarification:
  status: ready
  confidence: 100
  unresolved_questions: 0
feature:
  id: "0016"
  slug: external-task-activity
  dir: 0016-external-task-activity
relationships:
  - type: builds_on
    target: 0005-beacon-background-agent
  - type: builds_on
    target: 0010-project-following
references:
  - id: issue-31
    name: Improve Following visibility and live task awareness
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/31
    relation: implements
    read_policy: must
    used_for: user-selected delivery lane and acceptance contract
    status: active
  - id: pr-32
    name: Improve Following task awareness
    type: github-pr
    target: https://github.com/jamesonstone/beacon/pull/32
    relation: supports
    read_policy: must
    used_for: existing ready pull request
    status: active
  - id: codex-hooks
    name: Codex hooks documentation
    type: external-doc
    target: https://learn.chatgpt.com/docs/hooks
    relation: constrains
    read_policy: must
    used_for: documented Codex lifecycle events and payload fields
    status: active
  - id: claude-code-hooks
    name: Claude Code hooks documentation
    type: external-doc
    target: https://code.claude.com/docs/en/hooks
    relation: constrains
    read_policy: must
    used_for: documented Claude Code lifecycle and notification events
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: transient context, Go authority, Following, and macOS boundaries
    status: active
skills:
  - name: openai-docs
    source: codex
    path: /Users/jamesonstone/.codex/skills/.system/openai-docs/SKILL.md
    trigger: current Codex hook contract verification
    required: true
---

# External Task Activity

## Thesis

Beacon should show small, live presence and attention cues from structured
Codex and Claude Code lifecycle hooks without turning agent self-report into
evidence. Go owns normalization, exact project/worktree mapping, expiry, and
refresh coalescing; Swift renders the resulting transient overlay beside the
unchanged schema-v3 snapshot.

## Context

Features 0005 and 0010 established one background-agent snapshot, explicit
Following membership, and a shared menu/window presentation. They answer what
durable Git and GitHub evidence says, but they cannot indicate that a local
agent is currently working or has most recently asked for attention before
repository evidence changes.

This feature is a hooks-only validation experiment. It asks whether `working`,
`needs attention`, and `turn finished` context helps users return to the right
lane faster. It does not capture macOS notifications, inspect prompts, or add a
general activity-management product.

## Clarifications

- Version 1 supports documented local lifecycle hooks from Codex and Claude
  Code only.
- ChatGPT, ordinary Claude chat/Cowork, Warp, generic VS Code activity, macOS
  Notification Center, Accessibility capture, unstructured notification text,
  and remote sessions that do not execute local hooks are unsupported.
- `Stop` means the latest provider turn stopped. It does not mean that the
  project or task completed.
- `StopFailure` clears current Claude Code activity without creating a failure
  badge because no failure state exists in this MVP.
- `needs_attention` means the latest observed attention request. Providers do
  not consistently emit permission-resolved events, so it remains until a new
  prompt supersedes it with `working`, `Stop` supersedes it with
  `turn_finished`, `SessionEnd` removes it, or it expires.
- Codex has no documented `SessionEnd` hook, so Codex sessions rely on
  supersession and expiry.
- Activity never changes Following membership, lane attention, readiness,
  next action, ordering, evidence policy, or menu-bar lane counts.
- Agent protocol version 1 and the schema-v3 evidence snapshot remain
  unchanged. Mixed old/new agent and app combinations retain their prior
  transport behavior; an app paired with a helper that lacks activity commands
  simply has no overlay.
- An `active` integration proves only that Beacon observed the exact current
  callback fingerprint after installation. Codex may still require hook trust,
  and Claude Code may later be blocked by managed policy.

## Public Contracts

### Integration Commands

```text
beacon integrations install <codex|claude-code>
beacon integrations status <codex|claude-code>
beacon integrations uninstall <codex|claude-code>
```

Install and uninstall preview every exact handler addition/removal and adjacent
backup path before confirmation. `--yes` explicitly confirms a preview for
non-interactive callers. Settings are decoded before mutation; malformed hook
structures fail closed. Existing provider settings and unrelated hooks are
preserved. Every changed existing file receives an adjacent `0600` backup, and
every replacement is atomic and `0600`.

Each handler command includes provider timeout `2` and the shell-level guard:

```sh
'/absolute/path/to/beacon-cli' activity ingest --hook --provider codex \
  >/dev/null 2>&1 || true # beacon-activity-v1
```

The versioned marker identifies only Beacon handlers. Uninstall removes marked
handlers precisely and leaves unrelated event groups and commands intact. The
shell itself returns success when the executable moved, is missing, or exits
with an ingestion error.

### Normalized Activity

| Provider event | Result |
| --- | --- |
| `UserPromptSubmit` | `working` |
| `PermissionRequest` | `needs_attention` |
| Claude `Notification` with `permission_prompt`, `idle_prompt`, `elicitation_dialog`, or `agent_needs_input` | `needs_attention` |
| `Stop` | `turn_finished` |
| Claude `StopFailure` | remove the session's current activity |
| Claude `SessionEnd` | remove the session immediately |

Other Claude notification types are ignored. Beacon never derives `failed`
because neither supported normalization contract supplies an explicit MVP
failure state.

### Integration Health

| State | Meaning |
| --- | --- |
| `not_installed` | No Beacon-marked handler exists. |
| `installed` | Exact handlers and executable exist, but no callback was observed for their fingerprint. |
| `active` | A well-formed callback was observed for the exact current fingerprint. |
| `stale` | The command, marker, handler set, or executable does not match. |
| `error` | Provider settings or Beacon health state cannot be decoded safely. |

Health storage contains only each provider's handler fingerprint and observed
flag. It contains no activity, session, prompt, or notification content.

## Requirements

1. Add strict Codex and Claude Code hook installers for the documented event
   sets, with exact marked commands, timeout two, fail-open shell guards,
   previews, confirmation, adjacent restrictive backups, idempotence, malformed
   settings refusal, preservation, and precise uninstall.
2. Add a hidden fail-open hook ingestion command with a 32 KiB stdin limit and
   a total internal deadline below 500 milliseconds. It must never start the
   agent, write payloads to logs, or return failure to a provider hook.
3. Decode only provider event name, opaque session ID, `cwd`, supported
   notification type, and timestamp. Use receive time when no timestamp is
   supplied, hash the session ID before retention, and ignore prompts,
   transcripts, tool inputs, assistant content, titles, and free text.
4. Mark a well-formed current callback observed before contacting the agent.
   If the agent is unavailable, drop the activity event after that health
   update without starting or scanning anything.
5. Request the existing agent snapshot and map `cwd` only to followed projects:
   first choose the unique longest containing canonical worktree path and its
   lane; otherwise require exactly one containing canonical configured
   repository root and attach to the project header. Refuse missing paths,
   equal-length ambiguity, path-prefix collisions, unmatched paths, and
   non-followed projects.
6. Persist only current normalized records in
   `${XDG_CACHE_HOME:-~/.cache}/beacon/activity.json`: provider, state, hashed
   session key, project/lane target, observation time, expiry, and bounded
   project refresh-coalescing time. Use an atomic `0600` replacement and a
   short nonblocking lock.
7. Expire `working` after two hours, `needs_attention` after 24 hours or
   supersession, and `turn_finished` after one hour. Keep no activity history.
8. Go must physically remove every overdue record after every load or mutation
   and return the next expiry. Swift schedules that exact deadline and invokes
   a hidden Go prune command; delayed wake removes all overdue records before
   the next deadline is scheduled. Swift is never the expiry authority.
9. Request the existing `refresh_project` message only for a matched
   `turn_finished` event. Coalesce project refreshes for at least ten seconds.
   Never refresh for working, attention, failure clearing, session removal,
   unmatched, ambiguous, or non-followed events.
10. Add one shared compact chip renderer to exact lane cards and project
    headers, reused by the menu extra and detached dashboard. Priority is
    `needs_attention`, then `working`, then `turn_finished`; concurrent sessions
    render one count and mixed providers render `Agents`.
11. Add Codex and Claude Code integration-health rows to existing Settings,
    including the trust/managed-policy caveats, without a new destination or
    activity-management UI.
12. Keep activity outside schema v3, project and lane policy packages, agent
    protocol events, durable attention state, and menu-bar counts.

## Assumptions

- Supported hooks run locally and supply a canonicalizable existing `cwd`.
- Provider settings continue to use their documented JSON hook objects and
  command handlers.
- A missing provider resolution callback is normal; the badge is an observed
  cue, not a live permission lock.
- More providers should be considered only after this experiment demonstrates
  value and those providers offer structured lifecycle APIs.

## Acceptance Criteria

- [x] AC1: Every supported provider event normalizes to the documented state,
  attention-only Claude notification filtering works, and `Stop` is never
  labeled task completion.
- [x] AC2: `StopFailure` clears current activity without a failure badge and
  `SessionEnd` removes the session immediately.
- [x] AC3: Exact worktree-first and repository-fallback mapping handles path
  boundaries and longest paths while refusing ambiguity, missing paths, and
  non-followed projects.
- [x] AC4: Concurrent sessions supersede independently and render one
  provider/mixed-provider chip with the documented state priority and count.
- [x] AC5: Go physically prunes expired cache records, returns the next expiry,
  and the macOS timer calls Go again after wake or delay without Swift hiding
  records independently.
- [x] AC6: Duplicate matched Stops request exactly one project refresh per
  ten-second window; every excluded event or mapping requests none.
- [x] AC7: Hook execution exits quickly and successfully for missing/moved
  executables, malformed/oversized input, agent unavailability, lock
  contention, timeout, and write failure.
- [x] AC8: Provider payloads, prompts, transcripts, assistant text, tool inputs,
  notification text, and unhashed session IDs never enter cache, health state,
  or logs.
- [x] AC9: Install/uninstall preserve unrelated hooks, are idempotent, create
  `0600` backups, refuse malformed files, and remove only marked handlers.
- [x] AC10: Health transitions distinguish not installed, installed, active,
  stale, and error for the exact handler fingerprint.
- [x] AC11: The menu and detached dashboard use the same lane/header chip and
  Settings health presentation, with no new destination, inbox, alias editor,
  timeline, or permissions.
- [x] AC12: Agent protocol v1, schema-v3 evidence JSON, Following membership,
  lane attention, readiness, next action, ordering, and menu counts are
  unchanged.
- [x] AC13: README and constitution document the transient product boundary,
  support matrix, commands, health caveats, semantics, storage, and unsupported
  sources.
- [ ] AC14: Full Go, race, release, macOS, Linux, Kit, diff, and hosted gates
  pass and PR #32 receives updated evidence without being merged.

## Implementation Plan

1. Add Go event normalization, exact path mapping, versioned transient storage,
   expiry pruning, and targeted-refresh coalescing.
2. Add strict provider-settings editing, health state, fail-open hook commands,
   and explicit integration CLI commands.
3. Add shared Swift models, cache watching, Go-backed expiry scheduling,
   lane/header chip presentation, and Settings health rows.
4. Reconcile specification, constitution, README support matrix, and progress.
5. Run focused and complete validation, self-review, push the existing GH-31
   lane, and update ready PR #32 without merging it.

## Agent Team Plan

- The primary agent owns specification, implementation, validation,
  documentation, and delivery.
- Work remains serial because the Go cache authority, helper invocation, and
  shared AppState presentation cross one safety boundary.
- No subagents are used.

## Task Checklist

- [x] T1: Preserve issue #31 criteria and update ready PR #32 title/scope.
- [x] T2: Implement and focus-test Go activity normalization, mapping, storage,
  pruning, coalescing, and harmless ingestion.
- [x] T3: Implement and focus-test hook installation and integration health.
- [x] T4: Implement shared macOS activity chips, cache watching, Go pruning,
  and Settings health.
- [x] T5: Reconcile README, constitution, progress, and this specification.
- [x] T6: Complete full validation and self-review.
- [ ] T7: Commit, push, update PR #32 evidence, and leave it unmerged.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC2 | Go normalization tables, notification filters, supersession/removal tests, and chip copy assertions |
| AC3 | temporary-directory canonical path tables for longest worktree, repository fallback, ambiguity, path boundary, missing, and non-followed cases |
| AC4-AC5 | concurrent-session store tests, physical file inspection, next-expiry tests, Swift timer/client tests, and directory-watcher tests |
| AC6 | service fake-agent request recording with duplicate Stop and excluded-event cases |
| AC7-AC10 | bounded hook/lock tests, `/bin/sh` missing-executable smoke, payload leak assertions, installer/backup/idempotence tests, and health transition tables |
| AC11-AC12 | shared Swift renderer/AppState tests, unchanged protocol/schema assertions, full existing Go/macOS suites, and mixed-version fallback tests |
| AC13-AC14 | docs review, complete Make gate, Linux cross-build, Kit validation, diff/secret checks, and hosted PR checks |

## Reflection Notes

- Keeping the activity transport outside protocol v1 and schema v3 let a new
  bundled helper read an existing agent snapshot and send the existing
  `refresh_project` request without creating a mixed-version wire contract.
- The harmless-hook boundary is layered: the installed shell ends with
  `|| true`, the hidden command suppresses every ingestion error, and its work
  has a 475-millisecond outer deadline. Health is marked before the local agent
  request, but no activity is retained when that authority is unavailable.
- Canonical filesystem containment is stricter than project-name matching.
  Choosing one unique longest worktree first and exactly one repository root
  second preserves worktree lanes while refusing nested-repository ambiguity
  and textual path-prefix collisions.
- The integration editor treats the marker as Beacon-owned while comparing the
  expected handler set independently from unrelated hook ordering. It also
  rechecks settings after preview, so confirmation cannot overwrite a provider
  or user change made in the meantime.
- Go writes and prunes the only activity cache. Swift watches the cache
  directory, renders normalized records unchanged, and calls the helper at the
  returned expiry. The focused watcher test initially received more than one
  legitimate directory event; allowing over-fulfillment made that asynchronous
  contract explicit without weakening the assertion that a write is observed.

## Documentation Updates

- Add a README support matrix and integration setup/troubleshooting contract.
- Define external activity as transient, non-authoritative context in the
  constitution and keep it outside schema-v3 evidence policy.
- Add feature 0016 to the project progress summary.

## Delivery Decision

The user explicitly selected the existing issue #31, exact branch `GH-31`, and
ready PR #32. This feature preserves that lane and every prior acceptance
criterion, updates the PR's scope and evidence, and must not merge it.

## Evidence

- Event tables cover every supported Codex/Claude Code event, all four Claude
  attention notification types, ignored notification types, honest Stop
  semantics, StopFailure/SessionEnd removal, out-of-order supersession, input
  bounds, and session hashing.
- Temporary-directory tests cover longest-worktree mapping, repository
  fallback, equal-path ambiguity, nested-repository ambiguity, missing paths,
  path-boundary safety, and rejection outside Following.
- Store/service tests inspect physical cache pruning, `next_expiry`, concurrent
  sessions, ten-second Stop coalescing, excluded refreshes, bounded lock
  contention, unavailable-agent drops, and absence of raw prompt/transcript or
  session content.
- Integration tests cover exact commands, shell-level missing-executable
  success, provider preservation, unrelated-hook ordering, idempotence,
  restrictive backups, malformed/symlink refusal, post-preview change refusal,
  health transitions, stale commands/executables, and precise uninstall.
- An isolated HOME smoke exercised Codex status, install, status, and uninstall
  with exact previews and confirmed `0600` settings and health files.
- The Swift suite passes 81 tests, including normalized-cache decoding, chip
  priority/count aggregation, exact lane/header targeting, unchanged menu
  counts, cache-directory watching, and Go-backed delayed pruning.
- `make fmt-check vet test test-race build release-test macos-test macos-build`
  passed. Linux amd64 and arm64 CGO-disabled cross-builds passed.
- `kit check --all` passed all 16 feature specifications, `git diff --check`
  passed, the secret scan found no credential material, and protocol/model
  diffs are empty.
- Hosted PR evidence remains pending until the implementation commit is pushed.
