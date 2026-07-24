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
repositories and show work that is already in progress. A user should be able
to point one command at repository paths or parent directories, receive a
small factual list of active lanes, and leave without configuring, installing,
or starting anything else.

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

This is an additive dogfood path. Existing configured scans and the macOS
application remain available until the hyper-light workflow proves useful
enough to replace them.

## REQUIREMENTS

- Extend `beacon scan` so one or more positional paths select repository roots
  or recursively discovered source directories without loading or writing a
  Beacon configuration.
- Skip background-agent auto-start for every positional-path scan.
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

Non-goals are deleting v1 features, changing the configured scan contract,
changing the macOS application, introducing a daemon or database, persisting a
new v2 configuration, scanning non-GitHub repositories, or turning Beacon into
a task manager.

## ACCEPTED PLAN

1. Add an in-memory source configuration constructor that applies the existing
   path canonicalization and strict source rules without a config file.
2. Add a small `workscan` scanner that composes repository discovery, local Git
   scanning, authored open-PR collection, and existing lane policy in bounded
   parallel workers.
3. Project the collected evidence into a flat schema containing repository,
   worktree, branch, state, change counts, PR identity, next action, update
   time, and scoped diagnostics.
4. Route positional `beacon scan` invocations through this scanner and renderer
   while leaving the zero-argument configured path unchanged.
5. Cover source construction, in-progress classification, partial failures,
   terminal/JSON output, CLI routing, and background-agent suppression.
6. Document the dogfood command, validate the complete repository, curate
   durable memory, and deliver issue #76 from `GH-76`.

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

## DISCOVERIES

No new Git parser or GitHub schema is required. The essential reset is a
composition boundary: discover selected repositories, scan only work evidence,
filter factual in-progress lanes, and render the result directly. Initial
dogfooding also proved that prunable linked-worktree records must stay
diagnostic-only: presenting a deleted temporary checkout as active work hid the
real lane rather than helping the scan.

## VALIDATION

- `make fmt-check vet test test-race release-test build` passed. This includes
  config construction, scanner classification and partial-failure coverage,
  deterministic terminal output, JSON routing, argument conflicts, and
  background-agent suppression.
- `make macos-test` passed all 157 tests, proving the unchanged configured
  schema-v3 helper and native application remain compatible.
- With `BEACON_CONFIG` pointed at a nonexistent file, the built binary's
  positional JSON scan succeeded and `jq` verified schema version 1, one
  selected project, one active project, and only Beacon work items.
- A built-binary scan of Beacon and Kit returned two active projects and five
  work items without configuration or agent state. Both repositories'
  prunable temporary Kit-health worktrees remained warnings.
- `git diff --check` passed after the implementation and documentation updates.

## OUTCOME

`beacon scan PATH...` now builds a validated source selection entirely in
memory, discovers repository roots or parent-directory contents, scans linked
Git worktrees and authored open pull requests with bounded parallelism, and
projects only evidence-backed in-progress lanes into a small schema.

Terminal and JSON output share the same deterministic selection. Clean
base-only repositories are hidden unless `--include-idle` is present,
repositories with failed evidence are unknown rather than idle, prunable
worktrees remain diagnostic-only, and one repository's failure does not erase
other usable results. Positional scans reject persistent config and repository
filters and never start the background agent. The configured v1 CLI and native
macOS application remain unchanged while this workflow is dogfooded.
Issue #76 is represented by ready pull request #77 from exact branch `GH-76`.

## REPOSITORY MEMORY

Created this specification because the scope reset, retained v1 fallback,
evidence exclusions, and rejected adjacent product surfaces are material
rationale that code and tests cannot preserve. Updated the Constitution,
project progress summary, README, and user guide with the validated config-free
authority and its explicit boundary from the configured schema-v3 product.
