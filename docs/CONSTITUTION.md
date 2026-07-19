# CONSTITUTION

This document is Beacon's canonical project contract. It records the durable
product, architecture, implementation, and delivery rules that apply across
features. Feature-specific requirements and evidence belong in the applicable
`docs/specs/<feature>/SPEC.md`.

## PRINCIPLES

### Durable Evidence Over Self-Reported Progress

Beacon infers agent-work state from Git worktrees, branches, commits, remotes,
GitHub pull requests and issues, automation results, review feedback, and
optional Kit feature documents. Chat history, agent assertions, percentages,
and private task state are not evidence of review readiness.

### The Work Lane Is the Unit of Attention

A repository can contain several independent agent efforts. Beacon therefore
tracks work lanes, not repositories. A work lane is a local Git worktree and
branch optionally correlated with a GitHub pull request and issue, a
remote-only scoped pull request, an unlinked scoped issue waiting to start, or
a manually named planning/research lane. Repository identity is a discovery
container, not the primary attention control.

### Attention Is Durable, Context Is Optional

Each working-set lane has an explicit attention state (`active`, `waiting`,
`recent`, or `parked`), an independent pin, a last-seen observation, and an
optional short local note. Notes are memory cues, never canonical progress.
Beacon reports factual evidence deltas and marks notes stale when evidence
changes after the note. Parking is lane-specific; unrelated repository
activity must not reactivate it.
The macOS Following view names this explicit lane action Ignore; it maps to
parking only and must never unfollow the repository or delete lane state.
Lanes may also carry short, deduplicated user tags. Tags and notes are optional
context only and must not alter evidence, attention, readiness, or next-action
policy.
The working set also owns one complete global user lane order. It is projected
into evidence-derived attention groups, persists independently from pins, and
must never let presentation override attention or next-action policy. New lanes
enter at the front of their derived group; stale identities are removed during
reconciliation.

Beacon also owns one global Markdown signal log for transient working notes
that span lanes. It is optional local context, never durable evidence or a
source of inferred progress.

Structured external task activity is also optional context, never evidence.
Documented local provider hooks may report that a turn is working, that Beacon
most recently observed an attention request, or that a turn stopped. These
cues must remain transient and must never change Following membership, lane
attention, readiness, next action, ordering, policy, or menu-bar lane counts.
They remain outside the schema-v3 evidence snapshot and are discarded after
bounded expiry rather than retained as history.

### One Domain Model, Multiple Surfaces

Go is the source of truth for collection, correlation, policy, ordering,
tracking, caching, and actions. The background agent, direct CLI scans,
terminal output, JSON output, and the macOS application must present the
same snapshot. A client must not reimplement Git, GitHub, correlation,
project-activity classification, or readiness rules.

### Read-Only by Default

Observation must not change the work being observed. Beacon may perform a
bounded `git fetch --prune --no-tags` to refresh remote-tracking metadata.
Scanning and background refresh must never edit files, switch branches, create commits, push, create
or update pull requests, submit reviews, or merge. Beacon may atomically update
its own managed following state and cache when explicit user choices change,
fresh evidence updates a non-followed project's factual activity record, or a
scan produces a new last-good result. Evidence must never change Following
membership.

Repository sync is a separate, explicit mutation boundary. A passive check is
local-only; an explicit check may fetch only configured default-branch refs.
After confirmation, Beacon may fast-forward a clean checked-out default branch,
or return a clean feature branch already contained in the remote default branch
to the local default branch and fast-forward it. Dirty, detached, diverged,
unmerged, missing-ref, and multi-worktree states must be refused. Repository
sync never invokes `gh` or the GitHub API and never rebases, hard-resets,
force-updates, stashes, deletes, commits, pushes, or changes GitHub state.

Read-only merged-checkout advisories are part of evidence collection, not
Repository Sync. They may use Git to verify that a previously observed pull
request head branch disappeared, then spend one exact `gh pr view` request to
confirm the pull request merged. Only followed projects crossing that observed
open-to-absent transition qualify, at most three may be confirmed per refresh,
and results must be cached. An advisory never switches, pulls, deletes, or
otherwise mutates Git or GitHub state.

Provider-hook installation is another explicit local-settings mutation
boundary. Beacon may preview and, after confirmation, atomically add or remove
only its exact version-marked command handlers in supported provider JSON
files. Existing settings and unrelated hooks must be preserved, malformed
settings must be refused, and an adjacent user-only backup must precede every
change to an existing file. Hook execution itself is observational and
fail-open; it must never mutate a repository or provider task.

### Independent Signals Before Conclusions

Worktree, publication, pull-request, issue, CI, review, merge, and freshness state are
independent evidence. Beacon preserves those signals and then derives a
`review_ready` decision and one deterministic `next_action`. It must not hide
important evidence behind a single opaque status.

### Conservative, Explainable Readiness

Every readiness decision and recommended action must be traceable to explicit
reasons, warnings, and blockers. Unknown evidence blocks readiness when it may
conceal a correctness problem. Pending or absent CI remains visible even when
policy permits review.

### Partial Results Over Global Failure

A failure in one configured repository or one evidence source must not erase
healthy results. Repository-scoped errors belong in the snapshot beside usable
local or remote evidence. Only configuration and startup failures prevent the
scan itself.

### Determinism Is a Product Feature

Given the same inputs, Beacon must produce the same lane identities, ordering,
groups, readiness decisions, actions, and JSON shape. Stable output allows
humans, the macOS application, scripts, and future integrations to trust the
CLI as a contract.

### Explicit, Minimal Implementation

Prefer small cohesive packages, typed states, direct control flow, and
interfaces only at external boundaries or test seams. Avoid hidden global
state, shell interpolation, speculative frameworks, premature abstraction,
and duplicate business logic.

### Documentation and Evidence Complete the Change

Requirements, implementation, tests, validation evidence, reflection, and
delivery state are part of the feature. A change is not complete when code
exists; it is complete when the canonical artifact and project progress index
truthfully describe the highest completed state.

## GOALS

- Answer which agent work lanes are ready for human review.
- Distinguish local changes, unpublished commits, missing pull requests,
  drafts, failed or pending checks, requested changes, conflicts, staleness,
  remote-only work, and idle base work.
- Recommend the next useful human or agent action without mutating work.
- Preserve situational awareness across multiple repositories and worktrees.
- Provide a useful standalone CLI and a native macOS application backed
  by the identical versioned snapshot.
- Remain predictable under partial failures, unusual Git paths, stale remote
  state, and concurrent repository scans.
- Be simple to configure locally without storing GitHub credentials.

## ARCHITECTURE

### Data Flow

```text
strict versioned YAML configuration
              |
              +-- persistent source discovery
              +-- local Git worktree and branch evidence
              +-- bounded remote-tracking refresh
              +-- GitHub pull-request, issue, check, and feedback evidence
              +-- optional Kit progress documents
                              |
                      lane correlation
                              |
                   independent signals
                              |
              readiness and next-action policy
                              |
                 atomic per-project cache
                              |
                versioned Unix socket agent
                       /              \
              terminal / JSON     SwiftUI menu

documented local Codex / Claude Code hooks
                       |
        bounded Go normalization + exact cwd mapping
                       |
       separate user-only transient activity cache
                       |
              shared SwiftUI overlay
```

Collection, normalization, correlation, policy, presentation, and platform UI
are separate responsibilities. Data flows toward presentation; presentation
must not feed new policy back into the scanner.

### Go Package Boundaries

- `cmd/beacon` is a thin executable entry point.
- `internal/cli` defines Cobra commands, flags, exit behavior, and dependency
  wiring.
- `internal/config` resolves and strictly validates schema-versioned YAML.
- `internal/discovery` recursively resolves configured sources into accessible
  GitHub repositories without following symlinks.
- `internal/command` is the only general external-process boundary and uses
  `exec.CommandContext` with argument arrays.
- `internal/gitscan` discovers and inspects worktrees using stable,
  NUL-delimited Git porcelain output and performs bounded refreshes.
- `internal/reposync` owns local default-branch comparison and the only guarded,
  explicit fast-forward mutation path.
- `internal/githubapi` owns shared authenticated `gh` caching, background
  reserves, and explicit dependency-limit snapshots.
- `internal/githubscan` queries scoped open pull requests and issues through
  authenticated `gh` and normalizes checks, comments, reviews, unresolved
  threads with bounded comment detail, bounded issue and pull-request bodies,
  linked issues, and merge state.
- `internal/progress` parses optional Kit project summaries and exact SPEC
  issue references as non-authoritative progress evidence.
- `internal/model` owns schema v3 types and typed signal/action enums.
- `internal/policy` correlates local and remote evidence and derives readiness,
  explanations, and the next action as pure domain logic.
- `internal/scan` coordinates bounded repository concurrency, preserves partial
  results, orders lanes, and creates groups and summary counts.
- `internal/tracking` owns the strict repository-following store, evidence
  fingerprints, migration, and recent/quiet classification without automatic
  reactivation.
- `internal/workset` owns strict lane attention, pins, notes, tags, last-seen
  observations, factual deltas, manual lanes, one normalized user order, and
  project-tracking migration.
- `internal/notes` owns the atomic, size-bounded, user-only General document,
  stable-ID detail documents, and versioned tab workspace.
- `internal/ollama` owns bounded HTTP calls to the fixed loopback Ollama API,
  local-artifact model filtering, exact availability validation, and bounded
  role-ordered selected-Notes conversation requests.
- `internal/agent` owns operational paths, per-project caches, protocol-v1
  transport, scheduling, subscriptions, lifecycle locking, and LaunchAgent
  installation.
- `internal/activity` owns structured hook decoding, session hashing, exact
  snapshot-based path mapping, transient state, physical expiry pruning, and
  project refresh coalescing without changing evidence policy.
- `internal/integrations` owns supported provider hook settings, exact
  version-marked handler installation/removal, restrictive backups,
  fingerprints, and callback-observation health.
- `internal/output` renders the same snapshot as compact terminal text or JSON.

Package dependencies should follow this flow. In particular, scanners collect
facts; they do not decide product policy, and output packages do not rescan or
reclassify lanes.

### Lane Identity and Correlation

- Correlation is scoped to a configured GitHub repository.
- A pull request first matches a local lane by head branch and confirms the
  head object ID when possible.
- A pull-request-backed lane ID is `gh:<owner>/<repo>#<number>`.
- A local-only lane ID is `git:<owner>/<repo>@<url-escaped-branch>`.
- A head-object mismatch is retained as a warning rather than silently hidden.
- Closing issues, `GH-<number>` branches, and exact Kit SPEC issue references
  correlate in that order after pull-request/local matching.
- Scoped pull requests remain visible without a local worktree, and unmatched
  scoped issues become issue-only lanes.
- Manual lanes use stable `manual:<id>` identities and require neither Git nor
  GitHub evidence.
- Beacon scans active linked worktrees and scoped open GitHub work only; it
  does not enumerate every unattached local branch.

### Evidence and Policy

The public signal vocabulary is versioned with the JSON schema:

- Worktree: `clean`, `dirty`, `conflicted`, `unavailable`, `not_local`.
- Publication: `base`, `no_upstream`, `unpushed`, `published`, `behind`,
  `diverged`, `unknown`.
- Pull request: `none`, `draft`, `open`.
- CI: `success`, `pending`, `failure`, `none`, `unknown`.
- Review: `none`, `review_required`, `feedback_pending`,
  `changes_requested`, `approved`, `unknown`.
- Merge: `clean`, `blocked`, `conflicting`, `unknown`.
- Freshness: `current`, `stale`.
- Issue: `none`, `open`.
- Action: `review_pr`, `resolve_conflict`, `fix_ci`, `address_review`,
  `inspect_local`, `push_branch`, `create_pr`, `mark_ready`, `wait_for_ci`,
	`manual_test_then_merge`, `merge_pr`, `refresh_state`, `resume_or_close`,
	`continue_work`, `start_issue`, `none`.

A lane is review-ready only when it has an open, non-draft pull request; any
matching local lane is clean and has no unpublished, missing-upstream,
diverged, or unknown publication state; the pull request is not conflicting;
CI is not failed or unknown; and GitHub is not reporting requested or unknown
review state. Remote-only pull requests may be review-ready. Pending or absent
CI is permitted but remains a warning.

Recommended actions use fixed precedence: resolve conflict, fix CI, address
actionable review feedback, inspect local work, push the branch, refresh
uncertain or diverged state, create a pull request, mark a draft ready, wait
for CI, merge an approved clean change, manually test then merge an unapproved
clean change, review manually, resume or close stale work, start an unlinked
issue, then no action. Ordinary PR comments remain visible but do not block.
Staleness remains an independent warning even when another action wins.

Review-ready lanes sort first and wait longest-first. Other lanes sort by
action precedence, then oldest update, repository, and branch. Ordering changes
are product-policy changes and require tests and documentation.

### Configuration

Configuration resolution order is:

1. `--config <path>`
2. `BEACON_CONFIG`
3. `$HOME/.config/beacon/config.yaml`

Configuration accepts YAML schema versions 1 and 2. Version 2 adds persistent
source roots, `github_scope: mine|all`, and optional `ollama_model`; version 1
remains readable. Unknown
fields, unsupported versions, duplicate names or sources, invalid durations or
scope, missing paths, and malformed `owner/repo` values are errors. A leading
`~` is expanded and paths are canonicalized. Defaults are `main`, `origin`, a
one-minute scan interval, a 45-minute remote refresh interval, a 24-hour
stale threshold, four workers, GitHub author `@me`, and scope `mine`.
An empty or unavailable Ollama default selects the first installed local model
in stable name order without rewriting the configured value. Settings may
atomically update this same field through the bundled helper.

`beacon init` and its `beacon config init` alias may merge new sources or
explicit repositories, preview the result, and atomically rewrite the file
only after confirmation. Existing entries are never removed. GitHub
credentials never belong in Beacon configuration; authentication is delegated
to `gh`.

User repository-following state is separate from declarative discovery. It is
stored in strict JSON at `$HOME/.local/state/beacon/tracking.json`; legacy
sibling `tracking.yaml` and version-1 inverse-selection state are migrated
atomically. Per-project last-good snapshots live under
`$HOME/.cache/beacon/projects/`, and the agent socket lives at
`$HOME/.cache/beacon/agent.sock`. Version-2 state records every known project,
its explicit followed membership, deterministic Git/GitHub evidence and probe
baselines, and the time and factual reason for the latest material activity
outside Following. New discoveries begin non-followed. Changed durable evidence
may move a non-followed project into Recently Updated but must never follow it;
incomplete collection evidence must never establish or compare a baseline.
Operational files and directories are user-only and never contain GitHub
credentials.

Current external task activity lives separately at
`${XDG_CACHE_HOME:-$HOME/.cache}/beacon/activity.json`. It contains only a
provider, normalized state, hashed opaque session key, exact project/lane
target, observation/expiry times, and short refresh-coalescing metadata. Go
physically removes expired records; no activity history is retained. The
sibling `integration-health.json` contains only current provider handler
fingerprints and observed flags. Neither file belongs to schema v3, tracking,
lane attention, policy, or durable evidence.

The optional global Markdown signal workspace keeps its pinned General document
at `$XDG_DATA_HOME/beacon/notes.md`, defaulting to
`$HOME/.local/share/beacon/notes.md`. Stable-ID detail documents and the
versioned open-tab manifest live in the sibling `notes/` directory. That
manifest is the single authority for ordered pinned detail IDs as well as open
tabs: General is permanently pinned first, pinned details remain open in their
persisted order, and unpinned open details retain their relative order. Every
document remains independently size-bounded and atomically replaced with
user-only permissions. Directory locking, same-directory replacement, path
validation, and symlink rejection apply to the complete workspace, which stays
separate from repository files, configuration, lane evidence, and lane-specific
notes. A close operation changes workspace metadata only, rejects pinned
details until they are unpinned, and never deletes a detail document. A distinct
permanent delete operation accepts detail notes only, removes the selected
Markdown file and its workspace metadata under the same directory lock, and
selects the active note's left neighbor or General.

Lane attention is stored separately in strict versioned JSON at
`$HOME/.local/state/beacon/lanes.json` (or the equivalent `XDG_STATE_HOME`
path). It retains only the previous/last-seen and current durable observations
needed for deltas, not an event history. Existing muted project lanes migrate
to parked lane entries without deleting configuration or legacy tracking
state.

### Process Execution, Timeouts, and Concurrency

- External commands use `exec.CommandContext` and explicit argument arrays.
- Never construct a shell command from configuration or repository data.
- Local Git commands use five-second timeouts, fetch uses 30 seconds, and
  GitHub commands use 20 seconds unless a later specification changes the
  contract deliberately.
- Repository-sync checks and updates run concurrently only up to
  `settings.max_parallel`; apply rechecks worktree cleanliness, checked-out
  branch, relevant refs, and worktree placement immediately before mutation.
- Refresh is deduplicated for worktrees that share a Git common directory.
  Frequent local observation never fetches; fetch is reserved for explicit
  refresh or a deliberately slow remote cadence.
- Repository scans may run concurrently up to `settings.max_parallel`, which
  defaults to four and must remain bounded.
- The background scheduler runs at most one job per project, coalesces duplicate
  refresh requests, and uses separate tracked-refresh and muted-probe cadences.
  It consults cached due times before discovery; when no cached project is due,
  a scheduler tick performs no source walk, fetch, or GitHub collection.
- All background `gh` collection shares one persistent user-only response cache
  and rate-budget guard. Beacon reserves 2,500 GraphQL points, 15 Search
  requests, and 1,500 REST Core requests for the user's other GitHub work;
  cached successful evidence may be served stale while a bucket is protected.
- Cache misses are serialized per GitHub rate bucket, GraphQL work is
  conservatively debited at 25 points per command, and authoritative allowance
  state is refreshed after no more than five misses. Repository discovery uses
  only local Git remote and branch metadata and spends no GitHub capacity.
- Dependency-limit inspection is never scheduled and never runs at application
  startup. Each explicit inspection invokes exactly one bounded
  `gh api rate_limit` request and reports GraphQL, REST Core, and Search without
  initiating any additional API operation.
- Provider hook handlers use a provider timeout of two seconds, a shell-level
  `>/dev/null 2>&1 || true` guard, a 32 KiB input limit, and an internal Beacon
  deadline below 500 milliseconds. Malformed input, missing or moved helpers,
  agent unavailability, lock contention, timeout, and cache errors must return
  success to the provider. Hook ingestion never starts the Beacon agent and
  never writes raw payloads to logs.
- A valid hook asks the existing agent for its current snapshot, maps `cwd` to
  the unique longest containing followed worktree or otherwise exactly one
  containing followed repository root, and refuses missing, ambiguous,
  unmatched, or non-followed paths. Only a mapped turn-stopped event requests
  the existing targeted project refresh, coalesced for at least ten seconds;
  presence and attention updates spend no GitHub budget.
- Under the default `mine` scope, one due-project batch performs one global
  authored-PR search and one global assigned-issue search, then enriches every
  open authored PR in followed projects plus matching recent outside activity.
  Explicit diagnostics may enrich all inactive work while retaining one
  batched collection. Quiet projects still share recent batch evidence. The `all` scope
  remains an explicitly more expensive repository-scoped mode.
- Following mutations use cached complete evidence and never require a
  synchronous GitHub probe. The next scheduled non-followed probe establishes
  its compact comparison baseline. Explicit selection may persist a pending
  baseline while evidence is incomplete; the first later complete collection
  initializes it without inventing recent activity.
- Cancellation and command errors must retain enough command and repository
  context to diagnose the failed evidence stage.

### Public CLI and JSON Contracts

The supported command surface is:

```text
beacon [--color auto|always|never]
beacon init [--source PATH ...] [--github-scope mine|all] [--yes]
beacon scan [--repo NAME] [--json] [--no-refresh]
beacon sync
beacon sync check [project...] [--no-fetch] [--json]
beacon sync apply <project>... [--yes] [--json]
beacon limits [--json]
beacon projects [--followed|--recent|--quiet]
beacon select
beacon projects follow <project>...
beacon projects unfollow <project>...
beacon projects track <project>...
beacon projects untrack <project>...
beacon follow <project>...
beacon unfollow <project>...
beacon track <project>...
beacon untrack <project>...
beacon lanes [--parked]
beacon pin <lane-id> [--off]
beacon reorder <lane-id>...
beacon park <lane-id>
beacon resume <lane-id>
beacon note <lane-id> [text]
beacon notes [--json]
beacon notes list|new|open|close|delete
beacon notes show|set|append|edit|path [--note <id-or-exact-title>]
beacon tag <lane-id> <tag>
beacon untag <lane-id> <tag>
beacon add --manual <title>
beacon seen <lane-id>
beacon refresh [project]
beacon integrations install <codex|claude-code>
beacon integrations status <codex|claude-code>
beacon integrations uninstall <codex|claude-code>
beacon agent install|start|serve|status|stop|uninstall
beacon doctor [--json]
beacon open <lane-id>
beacon open-next
beacon config init|path|validate|open
beacon version
```

Successful scans exit `0`, including when lanes need action. Fatal
configuration or startup failures and failed required doctor checks exit `1`.
Usage errors exit `2`. JSON mode writes JSON only to stdout and sends
diagnostics to stderr.

Bare `beacon` is an explicit manual action: it asks the user agent for a forced
all-project refresh, waits for completion, and renders that current working-set
snapshot. When the agent is unavailable it performs the same blocking
foreground scan rather than silently returning stale evidence. TTY execution
may show the lighthouse trivia loader; non-TTY output never emits cursor-control
sequences. Opening or reconnecting the macOS client remains cache-only.
`beacon refresh`, macOS `Scan Now`, `beacon scan`, and JSON scan modes are the
other intentional paths for current evidence.

Agent protocol version 1 is newline-delimited JSON over a user-only Unix-domain
socket. It carries scan IDs, per-project revisions, stages, single and batch
tracking, complete lane-order and lane-attention changes, selected Markdown documents, typed note
workspace/create/open/close updates, explicit repository-sync reports,
heartbeats, and snapshot-schema-v3 payloads. Protocol evolution is independent
from the evidence snapshot schema. Clients discard events from a different
active scan and older project revisions, then preserve last-good state on
malformed events or disconnects.

The schema-v3 snapshot is a public internal contract between the CLI and
clients. It contains generation/config/refresh/tracking and working-set
metadata, projects,
following/recent/quiet counts plus compatibility tracked/untracked counts,
ordered enriched lanes, grouped lane IDs,
lane attention and global order, optional notes and tags, previous/current observations, factual deltas,
project following and activity evidence, and repository-scoped
or global warnings and errors. Expected partial conditions—including inaccessible
source discoveries, prunable worktrees, result truncation, and untrusted
optional Kit progress documents—are warnings, not errors. Human output keeps
their full detail out of the primary dashboard while JSON retains every
diagnostic. The terminal `Errors` section is reserved for evidence-collection
failures.
Collections must encode as arrays rather than `null`. Additive changes must be
safe for existing decoders; incompatible semantic or structural changes
require a schema-version increment and coordinated client support.

External task activity is deliberately not an additive schema-v3 or
protocol-v1 field. Hidden helper commands exchange a separate versioned
normalized cache shape with Swift. An older helper therefore yields no overlay
without changing mixed-version agent transport or invalidating the last-good
evidence snapshot.

### macOS Application Boundary

The macOS application targets macOS 14 or later and combines SwiftUI
`MenuBarExtra` with one compact detachable dashboard window. It runs as a
regular application with a Dock icon and Command-Tab presence so users retain
an entry point when the menu-bar item is obscured. Closing the dashboard leaves
the menu extra and background connection running; ordinary launches and later
user activation reopen the singleton window. It executes the bundled
`beacon-cli` helper, requires schema v3, and renders the CLI-provided projects,
groups, evidence, and actions.

The application connects to the background agent through a Swift actor,
renders cached state immediately, applies monotonic incremental project events,
and reconnects after disconnects without initiating collection. `Scan Now`
forces a full evidence refresh. The separate repository-sync control loads
local refs without network work and fetches only after **Check for Updates** or
an update click. A third dependency-limit control invokes the bundled helper's
`limits --json` command only when selected; it never polls or joins application
startup. These remain top-right controls in both surfaces. Agent status is authoritative
for loading state, including scans that complete before their request
acknowledgement. Only `@MainActor` publishes UI state. A
failed refresh keeps the last successful snapshot visible with its timestamp
and an error or stale banner.
Application launch idempotently starts or reconnects to the single user-scoped
agent before subscription retry. Normal application termination synchronously
unloads the LaunchAgent and stops any remaining socket/PID authority before the
process exits; closing only the detachable dashboard is not termination.
Ordinary direct CLI work may best-effort start a stopped macOS agent without
polluting command output, while explicit lifecycle, initialization, version,
and diagnostic commands retain their own semantics. Start and stop are
idempotent and never remove user notes, following state, or caches.
Both macOS surfaces render one reusable SwiftUI dashboard over the same
`AppState`; they must not duplicate subscriptions, scans, Git/GitHub policy, or
snapshot interpretation. An embedded, signed login-item helper may launch the
main app quietly with `--login` when the user explicitly enables Open at Login.
Service Management owns registration and approval, and the helper performs no
evidence collection itself.
The shared `AppState` may also watch the separate normalized external-activity
cache and render one compact chip on the exact project header or lane card.
State priority is latest attention request, working, then turn finished;
concurrent sessions collapse to a count and mixed providers use `Agents`.
Swift must not parse provider payloads, map paths, infer completion/failure, or
hide expired rows itself. It schedules the next Go-reported expiry and invokes
the bundled helper to prune every overdue record before rescheduling. Settings
may show compact Codex and Claude Code integration-health rows, including that
Codex can require trust and Claude Code can be blocked by managed policy, but
must not add an inbox, aliases, timeline, or activity-management destination.
Secondary commands and preferences live in a top-right Settings menu. A
separate compact view control offers a persisted stacked list, horizontal tile
strips, an experimental state-column kanban board, and an adaptive experimental
Overview over the same ordered lanes. Overview uses the dense shared card,
collapses empty groups, minimizes Notes while active, and restores its prior
size on exit. Comfortable, Compact, and Dense are separately persisted shared
card-density contracts, not alternate policy or font-size settings. A compact
peer tab row presents Following by default, then Parking Lot,
Recently Updated, and Quiet. Following omits parked lanes; the other tabs render
their shared Go categories without reimplementing evidence policy. Every open
in-scope PR for a followed project remains in Following regardless of age until
it closes or is explicitly parked. Settings
must not duplicate primary Recently Updated or Quiet navigation. Tab and view
selection are presentation state only. Dashboard destinations use one mutually
exclusive presentation state: a destination control opens its page on first
selection, selecting that same control again returns to Following, and selecting
a different destination switches directly to it. Lane tags render as removable
chips and mutate through the Go background-agent authority. Ordinary interface
copy uses system UI typography with an 11-point minimum; monospaced typography
is reserved for code, branches, identifiers, timestamps, percentages, and
counters. Shared base-size choices may scale these roles without changing them.

The application also owns one retained native drop-down terminal session.
Command-J is handled by an application-local AppKit event monitor and toggles a
focused panel inside the current dashboard window frame only while Beacon is
active. Beacon must not reserve the shortcut system-wide, so other applications
retain their own Command-J behavior. The shortcut needs no Accessibility or
Input Monitoring permission. The terminal follows dashboard moves and resizes,
and its frame is clipped to the dashboard's visible screen. Persisted
presentation settings choose the top or bottom dashboard edge and a 30%, 45%,
or 60% height; they do not change workflow state.
The panel runs one local login shell in the user's home directory through a
pseudo-terminal, inherits a validated absolute `SHELL` or falls back to
`/bin/zsh`, and terminates the child when Beacon terminates. Its default text,
cursor, selection, and complete 16-color ANSI palette derive from the active
Beacon theme and refresh live. Foreground-capable terminal colors must meet the
same 4.5:1 contrast floor as normal interface text against the terminal canvas
for default input, the cursor, and ANSI-16 entries; ANSI black remains the
structural canvas/background entry. Beacon does not
persist terminal output, parse it as evidence, or transfer scanning, Git,
GitHub, agent, or notes authority into Swift. Warp remains an external
alternative because it exposes no supported embedding or window-control API;
Settings may detect and open Warp and its official hotkey guide but must not
modify Warp preferences or control it through Accessibility.
Both surfaces expose one Notes panel at 50% of the available Beacon surface by
default. A header double-click cycles 50%, 80%, minimized, then 50%, and the
explicit chevron minimizes or restores the most recent expanded size. General
remains the first permanently pinned Go-owned Markdown document; user-pinned
details precede stable-order unpinned tabs and can be reordered through the same
versioned Go manifest. Swift owns no files or independent persistence rule. The
native editor remains editable and selectable;
focus reconciliation may resign first responder only after an explicit focused
to unfocused transition. Both surfaces share one draft and autosave queue,
flush before switching or closing, and preserve the active tab when saving
fails. Closing remains non-destructive. Permanent detail-note delete actions on
tabs, New Tab history, and Command-K/Command-P results all route through one
native destructive confirmation alert; General and New Tab expose no delete
action. The rocket wordmark mark, Notes solar system, and empty-state orbit use
native animation, carry no evidence semantics, and remain stationary when
Reduce Motion is enabled. The switchers use an opaque semantic theme surface
over a theme-aware backdrop. Native
Command-K and Command-P switchers plus tab-cycle and numeric shortcuts operate
through the frontmost shared view hierarchy. When Following
contains no in-progress lanes and no projects are loading,
both surfaces replace the empty lane body with an adaptive celebratory state whose
copy describes lane state rather than repository-ref freshness.
Expanded Notes exposes one accessible, always-enabled AI action with a generous
native target and playful signal-and-spark mark derived from semantic theme
tokens. It opens one compact assistant below the header action and inside the
current Beacon bounds; the Notes and all-commands quick switchers expose the
same action and restore expanded Notes when necessary. Command-I opens a larger
conversation panel from the right edge, while Command-Shift-I opens the compact
panel; Reduce Motion removes the spatial transition. A non-empty native editor
selection is captured exactly, otherwise the entire current draft is captured,
including unsaved visible edits. That complete snapshot is displayed first in
the scrollable conversation as removable attached context, and the prompt may be
sent without Notes context.

The active in-memory conversation retains every user and assistant turn in
order. Follow-ups send the complete role-aware history without silent
truncation; invalid or oversized input fails inline before generation while the
displayed history remains intact. History scrolls independently and assistant
Markdown uses the shared read-only renderer, while the unsent prompt, model
selector, and send action remain pinned to the panel bottom. Switching panel
sizes preserves the active session. A labeled Cancel action exits and resets
context, conversation, draft, errors, progress, and stale-response eligibility.
The model selector lists only installed local Ollama artifacts, and its
per-request choice does not implicitly change the configured default. Swift
sends bounded JSON to the helper over stdin and keeps no persistent chat
history. The helper alone contacts `http://127.0.0.1:11434`, rejects cloud or
unavailable models, validates conversation roles and size, and disables
streaming. No assistant response may mutate Notes or become Beacon evidence, and
no background insight request is permitted.
The Beacon wordmark may animate a modest horizontally traveling gradient derived
from the selected theme. It must remain readable, use no evidence or status
policy, and render a static gradient when Reduce Motion is enabled.
The menu-bar label always shows a compact, non-template colored beacon dome.
The number of lanes across the CLI-provided active, waiting, and recently-active
groups appears inside that dome with adaptive width and type scale through
`99+`, preserving the app identity and a legible count in one item. The menu
window and detached dashboard must render from one semantic theme catalog, but
color must not introduce readiness or action policy in the Swift client. Every
lane explicitly labels and symbolizes Local, Pull Request, Issue, or Manual;
theme-specific Local, PR, and Issue accents only reinforce that invariant work-
item identity mapping.

Beacon ships exactly five stable built-in theme IDs: `lobster-nebula`,
`pampas-moon`, `solarized-dark`, `monokai`, and `selenized-dark`. Lobster Nebula
is the recommended default dark theme and Pampas Moon is the high-readability
light theme. One stable AppStorage preference applies live to the menu extra,
detached dashboard, AppKit Markdown editor, tabs, lanes, controls, switchers,
dialogs, Notes, and empty/error states; unknown stored IDs fall back to Lobster
Nebula. Each complete token set owns canvas, layered surfaces, borders,
primary/secondary/muted text, accent/focus, success/warning/danger/info,
Local/PR/Issue identities, editor roles, and a derived terminal palette.
Ordinary text, cards, controls, and borders use solid neutral surfaces and
minimal shadows; gradients are reserved for the wordmark, beacon/rocket, and
occasional illustration. Every built-in theme must pass automated token-
completeness, stable-ID, persistence, rendered smoke, 4.5:1 normal-text and
terminal-foreground contrast, and 3:1 non-text/large-indicator contrast checks.
Raw classic accents that miss these thresholds require accessible semantic
aliases.

Both surfaces respect Increase Contrast, Differentiate Without Color, Reduce
Transparency, and Reduce Motion. Higher contrast strengthens semantic borders;
differentiate-without-color retains explicit labels and SF Symbols; reduced
transparency substitutes opaque theme surfaces; reduced motion disables
decorative and layout animation. These settings never change evidence or saved
workflow state.
Individual evidence badges may be hidden as reversible local presentation
state. Dismissal is scoped to lane, evidence dimension, and exact value so a
changed signal reappears; it must never mutate or suppress canonical evidence
in the Go snapshot. Healthy values remain quiet; only actionable or uncertain
exceptions appear by default with an explicit label and symbol. `PR feedback ·
N` is the count of unresolved pull-request review threads, not issue comments.
The adjacent information control explains the canonical identity, attention,
next-action, evidence-exception, and optional-context hierarchy.

Card and review-feedback detail is cached evidence presentation, not a refresh
path. Issue and pull-request bodies are bounded to 64 KiB; unresolved review
threads and their comments retain deterministic order, direct links, and
explicit truncation. Hover, keyboard focus, panel pinning, dismissal, and Escape
must execute no Git or GitHub command. Native Markdown detail and every status
presentation remain usable without relying on color alone. All read-only
Markdown evidence must pass through one theme-aware block renderer that
preserves headings, paragraphs, lists and tasks, quotes, code, dividers, tables,
inline emphasis, and links. It must not reinterpret ordinary interface labels
or mutate the cached source body.

Human-facing lane detail remains lane-centered. The CLI groups Active, Waiting,
Recently Active, and Parked lanes. The macOS dashboard opens on Following,
which contains the focused working set for explicitly followed repositories.
Every scoped open pull request and issue in a followed repository remains in
that working set regardless of age until closure or explicit parking.
Recently Updated is an inbox for material activity outside Following within the
configured stale window; Quiet is every remaining discovered non-followed
project. Top-item actions skip parked and quiet work plus manual lanes without
an openable target. These are presentation rules only: schema-v3 JSON retains
the complete diagnostic inventory and working-set grouping.

Following membership changes only through explicit user action. New discoveries
begin Quiet, and outside activity moves a project to Recently Updated without
changing membership. The CLI provides an interactive multi-select plus
follow/unfollow commands, with track/untrack retained as compatibility aliases.
The macOS application keeps searchable Following management in Settings and
offers Follow directly from Recently Updated and Quiet. Both clients delegate
persistence, recent classification, and mutation ordering to the Go following
service.

The application may use `NSWorkspace` to open pull requests, worktree paths,
and `$HOME/.config/beacon/config.yaml`. It may invoke the bundled helper's
project follow/unfollow, repository-sync, dependency-limit, and local Ollama
model/default/chat commands but
must not execute Git or `gh` directly or contain correlation, repository-sync safety, readiness,
fingerprint, cache, scheduling, or recent-activity policy. The
bundled helper is named `beacon-cli` to avoid a case-insensitive filename
collision with the `Beacon` application executable. The helper build must
support the target Mac architectures; the standalone CLI remains named
`beacon`.

### Release And Distribution

A push to `main` is Beacon's release event. Release automation derives one
strict SemVer from Conventional Commit history since the latest
`vMAJOR.MINOR.PATCH` tag. Breaking changes bump major, `feat` changes bump
minor, and all other accepted changes bump patch. The CLI and macOS app always
share that version because the app bundles the CLI and both artifacts come from
the same commit.

Release builds inject version, commit, and UTC build date into the standalone
CLI and bundled helper. The macOS bundle uses that SemVer as
`CFBundleShortVersionString` and an Actions run number as `CFBundleVersion`.
Published artifacts are macOS and Linux CLI archives for `amd64` and `arm64`, a
universal macOS application zip, and a SHA-256 manifest. GitHub generates the
release notes and creates the version tag at the exact merged commit.

Release jobs are serialized, use only `contents: write`, and must validate
before publishing. Rerunning a tagged commit reuses its version rather than
creating another tag. The application is ad-hoc signed for bundle integrity;
Developer ID signing, notarization, automatic updates, installers, and package
manager distribution remain out of scope.

## IMPLEMENTATION CONVENTIONS

### Go

- Use the Go version declared in `go.mod`; Beacon v1 uses Go 1.26.
- Prefer explicit typed structs and string-backed enums over unstructured maps.
- Keep exported APIs small and package ownership clear.
- Introduce interfaces at command, scanner, clock, or client boundaries when
  they enable deterministic testing; do not abstract ordinary local helpers.
- Return errors with operation and repository context. Expected partial
  failures become snapshot errors rather than panics.
- Pass `context.Context` through all external work and honor cancellation.
- Prefer table-driven unit tests, fake boundary implementations, fixtures, and
  temporary real Git repositories for integration behavior.
- Keep source files around 300 lines or less when a split improves clarity and
  ownership; do not split mechanically.
- Run `gofmt`; use idiomatic names, package-private helpers, and direct control
  flow.

### Swift

- Mirror schema v3 with explicit `Codable` models and snake-case coding keys.
- Keep mutable application state on `@MainActor`.
- Put process execution behind an injectable client protocol.
- Run the helper away from the main actor and surface typed, user-readable
  missing-helper, exit-status, timeout, and decode failures.
- Treat unknown future enum strings as display data unless the schema version
  itself is unsupported; never infer new policy in the UI.
- Test decoding, grouping, ordering, process failure, overlapping-scan
  prevention, last-good-state retention, open targets, and application start.

### Dependencies

Dependencies must have a clear job and must not absorb domain policy:

| Dependency | Purpose | Boundary |
| --- | --- | --- |
| Go standard library | Processes, contexts, JSON, concurrency, filesystem, time | Preferred implementation base |
| Cobra `v1.10.2` | CLI command, flag, help, and usage structure | No configuration or domain policy |
| `go.yaml.in/yaml/v3` `v3.0.4` | Strict YAML decoding | No Viper; normalization remains Beacon code |
| Huh `v1` | Native keyboard-driven init forms | No configuration or discovery policy |
| Lip Gloss `v1.1` and `x/term` | ANSI styling, visible widths, and TTY detection | JSON and policy remain style-free |
| Git | Worktree, status, branch, commit, base, and remote evidence | Machine-readable porcelain only |
| GitHub CLI `gh` | Authenticated pull-request/check/review evidence | GitHub is the only v1 remote provider |
| SwiftUI, AppKit, Foundation | Native menu UI, URL/path opening, process and JSON support | Presentation and process-client concerns only |
| SwiftTerm `v1.11.2` | Native terminal rendering and local pseudo-terminal process lifecycle | One presentation-only login shell; no Beacon policy, evidence, or persistence |
| XCTest | macOS unit tests | No production policy |

Indirect dependencies introduced by Cobra are accepted only as transitive
implementation details. New direct dependencies require a demonstrated
reduction in complexity or risk and must be recorded in the applicable spec.

## CONSTRAINTS

- Go remains the only source of scanning, caching, tracking, and readiness truth.
- Beacon remains read-only with respect to observed repositories and GitHub,
  except for its documented bounded metadata fetch. Its only application-state
  writes are explicit configuration changes, managed project tracking,
  user-only caches, lifecycle files, and logs.
- Scanner code must never use shell-built command strings.
- Every external command and concurrent operation must be bounded and
  cancellable.
- Repository failures must not suppress unrelated repository results.
- Stable identities, deterministic ordering, JSON stdout purity, exit codes,
  and schema versioning are compatibility requirements.
- Beacon v1 supports GitHub through authenticated `gh`; another provider needs
  an explicit feature specification and an adapter that preserves the domain
  model.
- The macOS application remains developer-local and unsandboxed so it can read
  configured repositories and invoke the bundled helper and system tools.
- The macOS target is 14 or later. GitHub release packaging and ad-hoc signing
  follow `docs/specs/0003-beacon-github-releases/SPEC.md`; Developer ID signing,
  notarization, sandboxing, and other distribution channels require another
  explicit specification.
- Code changes must preserve or strengthen the read-only boundary and must be
  validated against mutation-sensitive tests or inspection.
- Existing user-owned work in the working tree must not be overwritten,
  staged, or included in delivery without explicit authorization.

### Kit-Managed Baseline Rules

<!-- BEGIN KIT-MANAGED BASELINE RULES -->
- Treat `docs/CONSTITUTION.md` as the canonical project contract.
- Keep `AGENTS.md`, `CLAUDE.md`, and `.github/copilot-instructions.md` aligned with the repo-local docs tree.
- Treat `docs/notes/<feature>` as optional source material, not canonical truth; promote durable decisions into `SPEC.md`, `docs/CONSTITUTION.md`, or durable references.
- Prefer implementation/source code files around 300 lines or less when splitting improves clarity and ownership.
- Do not apply the code-file size guideline to documentation files, all `docs/**`, all `.kit/**`, or `.kit.yaml`.
- Do not split or rewrite docs, generated state, or Kit config artifacts solely because they exceed 300 lines.
<!-- END KIT-MANAGED BASELINE RULES -->
### Project Progress Summary

- `docs/PROJECT_PROGRESS_SUMMARY.md` is the canonical project-level progress
  index.
- It must always reflect the highest completed artifact or formal phase for
  every active or delivered feature.
- Advance its entry in the same change that advances the canonical feature
  artifact. Never record an aspirational phase, unchecked work, or a delivery
  claim without evidence.
- For workflow v2, record the highest completed phase in the feature's single
  `SPEC.md`: `clarify`, `ready`, `implement`, `validate`, `reflect`, or
  `deliver`.
- For an explicitly selected legacy staged workflow, record the highest
  completed canonical artifact: `BRAINSTORM.md`, `SPEC.md`, `PLAN.md`, or
  `TASKS.md`/delivery evidence.
- Progress entries link to the canonical artifact and summarize status,
  delivery lane, and validation evidence. They must not become a second source
  of detailed requirements or task state.
- A discrepancy between the summary and the canonical artifact is a defect;
  the canonical artifact wins and the summary must be corrected immediately.

## CHANGE CLASSIFICATION

Classify work before editing so the amount of process matches the risk.

### Spec-Driven (Formal)

Use the formal track for new features, substantial architecture or behavioral
changes, public schema or policy changes, provider additions, packaging model
changes, or work explicitly started with Kit's spec workflow.

Workflow v2 uses one canonical `docs/specs/<feature>/SPEC.md` and advances it
through `clarify`, `ready`, `implement`, `validate`, `reflect`, and `deliver`.
The artifact owns requirements, assumptions, acceptance criteria, plan, tasks,
validation mapping, reflection, documentation, delivery decision, and evidence.
Update `docs/PROJECT_PROGRESS_SUMMARY.md` whenever its highest completed phase
changes.

Legacy staged documents (`BRAINSTORM.md`, legacy `SPEC.md`, `PLAN.md`, and
`TASKS.md`) are used only when explicitly selected. They are not automatically
canonical for a workflow-v2 feature.

### Ad Hoc (Lightweight)

Use the lightweight track for small bug fixes, reviews, refactors, dependency
updates, configuration changes, and narrow documentation refinements whose
requirements and validation are already clear.

The workflow is understand, implement, verify. Update only practical docs and
the progress summary when the change affects a feature's highest completed
state. Do not create formal or legacy feature artifacts merely to satisfy
ceremony.

### Ad Hoc with Existing Specs

When an ad hoc change alters behavior governed by an existing spec, update that
spec by default. A purely mechanical change such as formatting, a typo, or a
non-behavioral dependency refresh may omit a spec edit, but verification still
must cover the changed surface.

## VALIDATION AND COMPLETION

- Map each acceptance criterion to an executable test, inspection, or manual
  demonstration before claiming completion.
- Go changes run formatting checks, `go vet ./...`, unit tests, race tests, and
  a CLI build in proportion to their scope.
- macOS changes run XCTest and an Xcode Debug build for the supported target.
- Contract changes test deterministic ordering, JSON shape and stdout purity,
  exit behavior, partial failure, and client decoding.
- Scanner changes test cancellation, timeouts, argument safety, unusual paths,
  and preservation of the read-only boundary.
- Record exact results, including skipped, pending, absent, or failing checks;
  do not translate them into stronger claims.
- A formal feature reaches `deliver` only after acceptance criteria and task
  checklists are complete, required validation evidence is recorded,
  documentation is current, and the delivery decision is explicit.
- Before issue, branch, staging, commit, push, or pull-request mutation, follow
  `docs/agents/GUARDRAILS.md` and the applicable rules under
  `docs/references/rules/`. Repo-local Kit delivery rules outrank generic GitHub
  defaults.

The baseline validation commands are:

```text
make fmt-check
make vet
make test
make test-race
make build
make macos-test
```

Run CLI smoke checks and `xcodebuild` directly when the changed behavior or the
active specification requires them.

## LONG-TERM VISION

Beacon should become the dependable local working-set memory for approximately
three to eight simultaneous coding-agent lanes without becoming a
project-management suite. Future
surfaces may add notifications, history, risk evidence, additional providers,
or deeper review context, but they must consume a stable domain snapshot rather
than duplicate collection and policy.

Evolution should preserve the work-lane model, explainable independent
signals, deterministic actions, partial results, and read-only trust boundary.
New evidence sources should enrich the model without converting uncertainty
into false precision. New clients should be thin. New mutations, providers,
background services, storage, or distribution channels require an explicit
specification, threat and compatibility analysis, and user-visible controls.

## NON-GOALS

- A Kanban board, issue tracker, or general project-management system.
- Synthetic progress percentages or estimates of how much coding remains.
- Parsing agent chats or depending on Codex task internals as canonical state.
- Capturing macOS Notification Center, Accessibility state, unstructured
  notification text, prompts, transcripts, tool inputs, or assistant content.
- Automatically editing work, switching branches, committing, pushing,
  creating or updating pull requests, reviewing, or merging.
- Hiding multiple signals behind an unexplained traffic-light status.
- Duplicating scanner or readiness logic in Swift or other clients.
- Enumerating every unattached local branch in version 1.
- Supporting non-GitHub forges, multiple users, or hosted collaboration in
  version 1.
- A history database, web dashboard, Beacon-generated user notifications,
  external-activity history or management UI, Homebrew distribution,
  Developer ID signing, notarization, App Store distribution, automatic
  updates, or an in-app configuration editor.

These items may become future features only through explicit requirements; they
are not implied by Beacon's long-term vision.

## DEFINITIONS

- **Beacon**: the CLI, versioned snapshot contract, and native menu client that
  expose agent-work evidence and recommended review actions.
- **Repository**: a configured local Git repository paired with an
  `owner/repo`, base branch, and remote name.
- **Work lane**: one active local worktree/branch, optionally correlated with a
  pull request, or one remote-only authored pull request.
- **Local evidence**: durable Git facts collected from worktrees, status,
  commits, upstreams, bases, and remote-tracking refs.
- **Remote evidence**: pull-request, CI, review, merge, and update facts returned
  by authenticated GitHub CLI queries.
- **Signal**: one normalized, independent dimension of lane evidence.
- **Review-ready**: the policy conclusion that a non-draft pull request meets
  the documented local publication, conflict, CI, and review requirements for
  human inspection.
- **Next action**: the single deterministic recommendation selected from all
  lane evidence using fixed precedence.
- **Partial failure**: a collection failure represented as scoped error data
  while unaffected lanes remain available.
- **Snapshot**: the ordered, schema-versioned result shared by terminal, JSON,
  and macOS surfaces.
- **External task activity**: a bounded, normalized lifecycle cue from a
  documented local provider hook; transient context that is neither snapshot
  evidence nor durable progress.
- **Canonical artifact**: the repository document that owns detailed truth for
  a workflow, normally a workflow-v2 feature `SPEC.md`.
- **Highest completed artifact or phase**: the furthest evidence-backed workflow
  state actually completed for a feature, never the state merely planned next.
