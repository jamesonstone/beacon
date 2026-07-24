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
- Let the selector enter child directories, move back toward its configured
  root, toggle Git repository directories selected or unselected, cancel
  without writing, and explicitly save the complete selection.
- Persist selected Git repository roots as the config-owned project list.
  Expand legacy parent-directory sources to exact repository roots on the
  first saved selection, preserve explicit repository metadata for paths that
  remain selected, and preserve selected projects outside the current browser
  root.
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
   exposes deterministic toggle, save, and cancel actions.
4. Seed selection from both configured sources and explicit repositories,
   expanding parent sources with filesystem-only Git discovery, then
   atomically replace the configured project set when the user saves.
5. Cover navigation boundaries, symlink exclusion, migration, empty selection,
   cancellation, config persistence, configured scan routing, and agent
   suppression.
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
- Preserve the config-free positional scan as a deliberate override even
  though configured zero-argument scanning becomes the primary workflow.

## DISCOVERIES

No new Git parser, GitHub schema, terminal dependency, or config schema is
required. Exact repository roots are valid version-2 sources, the existing
atomic writer can replace that list safely, and the existing `huh` form runner
supports a repeated one-level browser without a custom TUI runtime.

Broad legacy sources must be expanded with filesystem-only Git discovery before
the first save. Reusing full GitHub discovery there would make a config edit
depend on network or remote validity and could silently lose a selected local
repository. Explicit repository entries require separate preservation so a
selection edit does not discard their base or remote overrides.

An empty configured selection exposed a projection edge case during terminal
dogfooding: discovery warnings were present but the early empty-result return
left the summary warning count at zero. The scanner now counts those warnings
before returning. Initial v2 dogfooding also proved that prunable linked
worktree records must stay diagnostic-only rather than hiding the real lane.

## VALIDATION

- `make fmt-check vet test test-race release-test build` passed. Coverage now
  includes default-root resolution, enter/up navigation, selection toggles,
  source migration, explicit metadata and outside-root preservation, empty
  selection, cancellation, atomic persistence, configured scan routing,
  configured/discovered repository deduplication, warning counts, and agent
  suppression.
- `make macos-test` passed all 157 tests, proving the native application and
  full schema-v3 dashboard remain compatible.
- A real terminal session entered and exited an isolated owner directory,
  selected two repositories, and atomically wrote two exact project roots.
- A second real terminal session used the repository itself as the browser
  root, selected Beacon into a fresh isolated config, and wrote normalized
  version-2 settings plus one exact source.
- The built binary then consumed that config with
  `scan --no-refresh --json`; `jq` verified work-scan schema version 1, one
  selected project, one active project, and at least one work item.
- The initial positional Beacon/Kit dogfood remains valid: both repositories'
  prunable temporary Kit-health worktrees stayed warnings.

## OUTCOME

`beacon projects` now opens a lazy directory browser rooted at
`~/go/src/github.com` by default. Ordinary directories navigate inward,
`..` navigates outward without crossing the chosen root, Git repository
directories toggle selected state, Cancel performs no write, and Save
atomically replaces the config-owned project list. Legacy parent sources are
expanded to exact roots; selected explicit metadata and projects outside the
current browser root survive the rewrite; an empty selection is valid.

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
Constitution with the now-demonstrated config-owned selection boundary,
explicit config mutation authority, and shared configured/positional v2
projection. Updated the project progress summary, README, and user guide with
the validated terminal workflow and migration behavior.
