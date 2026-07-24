---
kit_metadata_version: 1
artifact: spec
workflow_version: 3
phase: deliver
delivery_intent: ready_pull_request
feature:
  id: 0022
  slug: beacon-v2-hyper-light
  dir: 0022-beacon-v2-hyper-light
relationships:
  - type: builds_on
    target: 0001-beacon-v1
  - type: builds_on
    target: 0009-beacon-working-set-radar
references:
  - id: issue-76
    name: Build Beacon v2 hyper-light project scanner
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/76
    relation: implements
    read_policy: must
    used_for: product goal, scope, acceptance, and delivery lane
    status: active
  - id: beacon-v1
    name: Beacon v1
    type: spec
    target: docs/specs/0001-beacon-v1/SPEC.md
    relation: informs
    read_policy: must
    used_for: lane evidence, read-only boundary, and scanner contracts
    status: active
  - id: pr-77
    name: Add hyper-light project scan
    type: github-pr
    target: https://github.com/jamesonstone/beacon/pull/77
    relation: implements
    read_policy: must
    used_for: issue-76 implementation, review, and hosted validation
    status: active
  - id: working-set-radar
    name: Beacon working-set radar
    type: spec
    target: docs/specs/0009-beacon-working-set-radar/SPEC.md
    relation: informs
    read_policy: must
    used_for: original narrow usefulness thesis and factual next-action policy
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: read-only authority, evidence ordering, and terminal behavior
    status: active
skills: []
---
# Beacon v2 Hyper-light

## PURPOSE

Beacon v2 restores the product to one useful job: scan a chosen set of project
repositories and show work that is already in progress. The normal workflow
should remember that project set in Beacon's config, let the user change it
through a small terminal directory browser, and scan it without requiring
paths on every invocation.

## CONTEXT

Beacon's existing Git worktree, GitHub pull-request, correlation, and
next-action policy are proven and well tested. The accumulated product,
however, makes the simplest scan depend on persistent configuration, project
following, cached attention state, a background agent, and a large adjacent
surface including Notes, terminal, AI, themes, and macOS presentation.

The v2 slice should reuse the trustworthy evidence collectors without routing
through those product layers. `internal/discovery` already accepts either a
repository root or a parent directory, deduplicates linked worktrees, and
derives GitHub, base-branch, and remote identities. `internal/gitscan` already
reports every linked worktree. `internal/githubscan.Client.ListOpen` can query
only authored open pull requests per selected repository instead of invoking
the broader issue, tracking, and cache collectors.

The initial v2 slice made positional paths config-free and left zero-argument
configured scans on the v1 projection. Dogfooding established that repeatedly
supplying paths is unnecessary friction: project selection is durable user
intent, while the work evidence itself remains read-only and ephemeral.
Positional paths remain useful as an ad hoc override.

## REQUIREMENTS

- Make zero-argument `beacon scan` load the configured project list and render
  the hyper-light work projection.
- Keep one or more positional paths as an ad hoc scan override that neither
  loads nor writes Beacon configuration.
- Make interactive `beacon projects` browse from
  `~/go/src/github.com` by default, with an optional root override.
- Let Tab and the arrow keys move the highlight, Space enter child directories
  or toggle Git repository directories selected or unselected, Enter confirm
  and save the complete selection, and Escape cancel without writing.
- Persist selected Git repository roots in a dedicated config-owned `projects`
  list. Preserve selected projects outside the current browser root, but begin
  with no v2 selection when only legacy sources, explicit repositories, or
  managed Following state exist.
- Permit a valid empty version-2 project list so the user can unselect every
  project and receive an empty scan instead of retaining an accidental project.
- Skip background-agent auto-start for every configured or positional v2 scan
  and for the interactive project selector.
- Collect only local Git worktrees and authored open pull requests for the
  selected repositories. Do not load tracking, Following, Notes, activity,
  repository-sync, progress, or agent state.
- Treat dirty or conflicted worktrees, non-base branches, commits ahead of the
  base or upstream, detached worktrees, and open pull requests as work in
  progress. Report prunable worktrees as warnings without promoting them to
  active work.
- Exclude issue-only backlog and clean base-only repositories by default.
- Preserve `--include-idle`, `--no-refresh`, `--json`, and color behavior for
  positional scans.
- Emit a deliberately small versioned JSON document and a concise deterministic
  terminal table.
- Continue reporting healthy repositories when discovery, Git, fetch, or
  GitHub evidence fails elsewhere. Do not classify a repository as idle when
  required evidence failed.
- Preserve Beacon's read-only authority: no working-file, branch, commit, push,
  review, or merge mutation.

Non-goals are deleting v1 features, changing the macOS application,
introducing a daemon or database, following filesystem symlinks, scanning
non-GitHub repositories, or turning Beacon into a task manager.

## ACCEPTED PLAN

1. Preserve the in-memory positional source constructor and existing
   hyper-light scanner as the single evidence path.
2. Route zero-argument configured scans through that scanner and suppress
   background-agent startup for all `scan` invocations.
3. Add a lazy filesystem browser that lists safe child directories, identifies
   Git repository roots, constrains parent navigation to the chosen root, and
   gives Space, Enter, Tab, and Escape one stable action each.
4. Store v2 selection in a dedicated `projects` list so legacy discovery and
   Following inventory begin unselected, then atomically replace only that
   list when the user confirms.
5. Cover navigation boundaries, symlink exclusion, keyboard semantics, focus
   retention, empty selection, cancellation, config persistence, configured
   scan routing, and agent suppression.
6. Update the CLI documentation, validate the complete repository, curate
   durable memory, and update issue #76 and PR #77 from `GH-76`.

## DECISIONS

- Use positional paths on the existing `scan` verb rather than a second
  top-level product vocabulary.
- Keep v1 available while dogfooding rather than beginning with a destructive
  repository-wide rewrite.
- Query open pull requests per selected repository and omit open issues because
  the question is current work, not available backlog.
- Reuse the existing lane policy after deliberately narrowing its evidence
  inputs, so status and next-action facts remain compatible with v1.
- Keep metadata-only fetch behavior under the existing `--no-refresh` switch;
  this refreshes Git evidence without editing a checkout.
- Treat exact configured project roots as durable selection, not live work
  state; use the existing atomic YAML writer instead of a second state store.
- Use `beacon projects` as the interactive v2 selection surface. Retain the
  older explicit follow/unfollow commands for compatibility, but do not route
  v2 project selection through tracking, agent cache, or Following state.
- Browse lazily one directory at a time rather than recursively building a
  large terminal tree. Repository directories are toggle targets and ordinary
  directories are navigation targets.
- Make Space the highlighted-row action and Enter the single confirmation
  action. Keeping Save and Cancel out of the item list makes Tab a pure forward
  navigation key and prevents accidental confirmation while browsing.
- Keep hyper-light selection in `projects`, separate from legacy discovery
  sources, explicit repositories, and managed Following state. The initial
  source-expansion migration was rejected after it made 88 repositories appear
  selected before the user chose any.
- Preserve the config-free positional scan as a deliberate override even
  though configured zero-argument scanning becomes the primary workflow.

## DISCOVERIES

No new Git parser or GitHub schema is required. The existing Bubble Tea
dependencies provide the needed keyboard loop, filtering, pagination, and
terminal resize handling. The atomic writer can update the dedicated
`projects` list while leaving legacy discovery configuration untouched.

Treating existing sources and explicit repositories as the v2 selection was
semantically wrong: those fields describe the broader v1 inventory, so
expanding them made every discovered repository appear deliberately selected.
A separate `projects` field preserves both meanings and gives an absent field a
safe zero-selection migration.

An empty configured selection exposed a projection edge case during terminal
dogfooding: discovery warnings were present but the early empty-result return
left the summary warning count at zero. The scanner now counts those warnings
before returning. Initial v2 dogfooding also proved that prunable linked
worktree records must stay diagnostic-only rather than hiding the real lane.

## VALIDATION

- `make fmt-check vet test test-race release-test build` passed. Coverage now
  includes default-root resolution, Space enter/up/toggle actions, Enter
  confirmation, Tab movement, focus retention, dedicated-list migration,
  outside-root preservation, empty selection, cancellation, atomic persistence,
  configured scan routing, warning counts, and agent suppression.
- `make macos-test` passed all 157 tests, proving the native application and
  full schema-v3 dashboard remain compatible.
- A real terminal session against the default config opened at
  `~/go/src/github.com` with zero selected projects, moved forward with Tab,
  and confirmed with Enter. The atomic rewrite preserved five legacy sources
  and one explicit repository while persisting `projects: []`.
- An isolated terminal session used Space to enter an owner directory, Tab to
  focus its repository, Space to select and unselect it across redraws, and
  Enter to persist the final empty selection.
- The built binary then consumed the default config with
  `scan --no-refresh --json`; `jq` verified work-scan schema version 1 and zero
  projects, work items, errors, and warnings.
- The initial positional Beacon/Kit dogfood remains valid: both repositories'
  prunable temporary Kit-health worktrees stayed warnings.

## OUTCOME

`beacon projects` now opens a lazy directory browser rooted at
`~/go/src/github.com` by default. Tab and the arrow keys move, Space enters an
ordinary directory or toggles a Git repository, `..` navigates outward without
crossing the chosen root, Enter atomically confirms, and Escape cancels. The
dedicated `projects` config list begins empty instead of inheriting the legacy
inventory, preserves selected projects outside the current browser root, and
permits a valid empty selection.

Zero-argument `beacon scan` now reads that configured list and uses the same
small deterministic work projection as positional scans. Both configured and
positional modes avoid tracking, Following, cache, Notes, agent, and macOS
state; hide clean base-only repositories unless `--include-idle` is present;
classify failed evidence as unknown; and preserve scoped partial results.
Positional paths remain a config-free override. The full v1 dashboard, explicit
Following controls, and native macOS application remain available.

Issue #76 remains represented by ready pull request #77 from exact branch
`GH-76`.

## REPOSITORY MEMORY

Created this specification because the scope reset, retained v1 fallback,
evidence exclusions, and rejected adjacent product surfaces are material
rationale that code and tests cannot preserve. Updated the specification and
Constitution with the now-demonstrated dedicated `projects` boundary, explicit
keyboard and config mutation authority, and shared configured/positional v2
projection. Updated the project progress summary, README, and user guide with
the validated terminal workflow and zero-selection migration behavior.
