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
remote-only scoped pull request, or an unlinked scoped issue waiting to start.

### One Domain Model, Multiple Surfaces

The Go CLI is the source of truth for collection, correlation, policy,
ordering, and actions. Terminal output, JSON output, and the macOS menu
application must present the same snapshot. A client must not reimplement Git,
GitHub, correlation, or readiness rules.

### Read-Only by Default

Observation must not change the work being observed. Beacon may perform a
bounded `git fetch --prune --no-tags` to refresh remote-tracking metadata.
Scanning must never edit files, switch branches, create commits, push, create
or update pull requests, submit reviews, or merge.

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
humans, the menu application, scripts, and future integrations to trust the
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
- Provide a useful standalone CLI and a native macOS menu application backed
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
                       /              \
              terminal / JSON     SwiftUI menu
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
- `internal/githubscan` queries scoped open pull requests and issues through
  authenticated `gh` and normalizes checks, comments, reviews, unresolved
  threads, linked issues, and merge state.
- `internal/progress` parses optional Kit project summaries and exact SPEC
  issue references as non-authoritative progress evidence.
- `internal/model` owns schema v2 types and typed signal/action enums.
- `internal/policy` correlates local and remote evidence and derives readiness,
  explanations, and the next action as pure domain logic.
- `internal/scan` coordinates bounded repository concurrency, preserves partial
  results, orders lanes, and creates groups and summary counts.
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
  `start_issue`, `none`.

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
source roots and `github_scope: mine|all`; version 1 remains readable. Unknown
fields, unsupported versions, duplicate names or sources, invalid durations or
scope, missing paths, and malformed `owner/repo` values are errors. A leading
`~` is expanded and paths are canonicalized. Defaults are `main`, `origin`, a
one-minute scan interval, a five-minute remote refresh interval, a 24-hour
stale threshold, four workers, GitHub author `@me`, and scope `mine`.

`beacon init` and its `beacon config init` alias may merge new sources or
explicit repositories, preview the result, and atomically rewrite the file
only after confirmation. Existing entries are never removed. GitHub
credentials never belong in Beacon configuration; authentication is delegated
to `gh`.

### Process Execution, Timeouts, and Concurrency

- External commands use `exec.CommandContext` and explicit argument arrays.
- Never construct a shell command from configuration or repository data.
- Local Git commands use five-second timeouts, fetch uses 30 seconds, and
  GitHub commands use 20 seconds unless a later specification changes the
  contract deliberately.
- Refresh is deduplicated for worktrees that share a Git common directory and
  is skipped until the configured refresh interval has elapsed.
- Repository scans may run concurrently up to `settings.max_parallel`, which
  defaults to four and must remain bounded.
- Cancellation and command errors must retain enough command and repository
  context to diagnose the failed evidence stage.

### Public CLI and JSON Contracts

The supported command surface is:

```text
beacon [--color auto|always|never]
beacon init [--source PATH ...] [--github-scope mine|all] [--yes]
beacon scan [--repo NAME] [--json] [--no-refresh]
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

Bare `beacon` shows a rotating lighthouse sweep while an interactive terminal
waits for the live scan. The loader starts only after configuration and any
initialization prompt succeed, clears its line before dashboard output, and
restores the cursor on success, error, cancellation, or panic unwinding.
Redirected output, JSON, and explicit `beacon scan` commands never emit loader
frames or cursor-control sequences. `--color=never` keeps the animation but
removes its color; `NO_COLOR` has the same effect in automatic mode.

The schema-v2 snapshot is a public internal contract between the CLI and
clients. It contains generation/config/refresh metadata, projects, summary
counts, ordered enriched lanes, grouped lane IDs, and repository-scoped or
global warnings and errors. Expected partial conditions—including inaccessible
source discoveries, prunable worktrees, result truncation, and untrusted
optional Kit progress documents—are warnings, not errors. Human output keeps
their full detail out of the primary dashboard while JSON retains every
diagnostic. The terminal `Errors` section is reserved for evidence-collection
failures.
Collections must encode as arrays rather than `null`. Additive changes must be
safe for existing decoders; incompatible semantic or structural changes
require a schema-version increment and coordinated client support.

### macOS Application Boundary

The macOS application targets macOS 14 or later, uses SwiftUI `MenuBarExtra`
with window style, and runs as an `LSUIElement` application without a Dock
icon. It executes the bundled `beacon-cli` helper, requires schema v2, and
renders the CLI-provided projects, groups, evidence, and actions.

The application scans at launch, at most once every 60 seconds, and on explicit
request. Overlapping scans are prohibited. A failed scan keeps the last
successful snapshot visible with its timestamp and an error or stale banner.
The menu-bar label shows the number of non-idle lanes across the CLI-provided
ready, action, and waiting groups. Active counts use a high-contrast dark badge
with a luminous neon-gradient border so the value remains visible over changing
menu-bar backgrounds. When that count is zero, it shows a compact color
neon-space glyph instead of a numeric badge. The menu window may use coordinated
pastel and neon accents to distinguish existing CLI-provided groups and signals,
but color must not introduce readiness or action policy in the Swift client.

The application may use `NSWorkspace` to open pull requests, worktree paths,
and `$HOME/.config/beacon/config.yaml`. It must not execute Git or `gh`
directly or contain correlation/readiness policy. The bundled helper is named
`beacon-cli` to avoid a case-insensitive filename collision with the `Beacon`
application executable. The helper build must support the target Mac
architectures; the standalone CLI remains named `beacon`.

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

- Mirror schema v2 with explicit `Codable` models and snake-case coding keys.
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
| XCTest | macOS unit tests | No production policy |

Indirect dependencies introduced by Cobra are accepted only as transitive
implementation details. New direct dependencies require a demonstrated
reduction in complexity or risk and must be recorded in the applicable spec.

## CONSTRAINTS

- The Go CLI remains the only source of scanning and readiness truth.
- Beacon remains read-only except for its documented, bounded metadata fetch
  and explicit configuration-file creation.
- Scanner code must never use shell-built command strings.
- Every external command and concurrent operation must be bounded and
  cancellable.
- Repository failures must not suppress unrelated repository results.
- Stable identities, deterministic ordering, JSON stdout purity, exit codes,
  and schema versioning are compatibility requirements.
- Beacon v1 supports GitHub through authenticated `gh`; another provider needs
  an explicit feature specification and an adapter that preserves the domain
  model.
- The menu application remains developer-local and unsandboxed so it can read
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

Beacon should become the dependable local signal layer for supervising many
coding-agent work lanes without becoming a project-management suite. Future
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
- Automatically editing work, switching branches, committing, pushing,
  creating or updating pull requests, reviewing, or merging.
- Hiding multiple signals behind an unexplained traffic-light status.
- Duplicating scanner or readiness logic in Swift or other clients.
- Enumerating every unattached local branch in version 1.
- Supporting non-GitHub forges, multiple users, or hosted collaboration in
  version 1.
- A history database, background daemon, web dashboard, notifications,
  launch-at-login, Homebrew distribution, Developer ID signing, notarization,
  App Store distribution, automatic updates, or an in-app configuration editor.

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
- **Canonical artifact**: the repository document that owns detailed truth for
  a workflow, normally a workflow-v2 feature `SPEC.md`.
- **Highest completed artifact or phase**: the furthest evidence-backed workflow
  state actually completed for a feature, never the state merely planned next.
