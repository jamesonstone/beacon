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

`bctl` restores the v2 product to one useful job: scan a chosen set of project
repositories and show work that is already in progress. The normal workflow
should remember that project set in Beacon's config, let the user change it
through a small terminal directory browser, and scan it without requiring
paths on every invocation. The `beacon` executable remains the legacy
working-set dashboard and macOS helper so the two products have explicit
command boundaries.

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

Further dogfooding exposed that overloading `beacon scan` and `beacon projects`
with v2 behavior left bare `beacon` on a different legacy inventory and made
the product boundary unclear. A dedicated `bctl` executable makes the
hyper-light contract explicit while preserving the established `beacon`
dashboard, agent, and native application.

## REQUIREMENTS

- Make bare `bctl` and zero-argument `bctl scan` load the configured project
  list and render the hyper-light work projection.
- Keep one or more positional paths as an ad hoc scan override that neither
  loads nor writes Beacon configuration.
- Make interactive `bctl projects` browse from
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
- Keep `bctl` independent of the background agent for every configured or
  positional scan and for the interactive project selector.
- Remove the v2 selector and positional work-scan projection from `beacon`;
  restore `beacon scan` and `beacon projects` to their legacy dashboard
  contracts without compatibility aliases for the moved v2 commands.
- Build, install, test, and package `bctl` beside `beacon`; keep the embedded
  macOS helper on the legacy `beacon` executable.
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
2. Add a thin `cmd/bctl` entry point and a dedicated Cobra root that runs the
   configured scan when invoked bare, also exposes `bctl scan [path...]`, and
   never owns agent lifecycle behavior.
3. Move the configured project browser to `bctl projects` and restore the
   legacy `beacon scan` and `beacon projects` command contracts.
4. Add a lazy filesystem browser that lists safe child directories, identifies
   Git repository roots, constrains parent navigation to the chosen root, and
   gives Space, Enter, Tab, and Escape one stable action each.
5. Store v2 selection in a dedicated `projects` list so legacy discovery and
   Following inventory begin unselected, then atomically replace only that
   list when the user confirms.
6. Build and package both executables, keeping only `beacon` embedded in the
   macOS application.
7. Cover command separation, bare and explicit bctl scans, navigation
   boundaries, symlink exclusion, keyboard semantics, focus retention, empty
   selection, cancellation, config persistence, configured scan routing, and
   absence of agent lifecycle behavior.
8. Update the CLI documentation, validate the complete repository, curate
   durable memory, and update issue #76 and PR #77 from `GH-76`.

## DECISIONS

- Name the hyper-light executable `bctl` and keep `beacon` as the legacy
  working-set product. This is an executable boundary, not a mode flag.
- Let bare `bctl` perform the configured scan while retaining `bctl scan` as
  the explicit, script-friendly form and positional-path surface.
- Do not retain `beacon` aliases for v2 selection or positional scans; aliases
  would preserve the ambiguity this separation removes.
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
- Use `bctl projects` as the interactive v2 selection surface. Retain the
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
  includes the dedicated bctl root, bare and explicit configured scans,
  positional scans, rejected legacy command forms, executable-specific version
  output, default-root resolution, Space enter/up/toggle actions, Enter
  confirmation, Tab movement, focus retention, dedicated-list migration,
  outside-root preservation, empty selection, cancellation, atomic persistence,
  warning counts, and absence of bctl agent lifecycle behavior.
- `make macos-test` passed all 157 tests, proving the native application and
  legacy `beacon` helper and full schema-v3 dashboard remain compatible.
- `kit check 0022-beacon-v2-hyper-light` and `kit check --all` passed all 22
  feature checks after repository-memory curation.
- Built `beacon` and `bctl` executables reported their distinct names with the
  same build metadata. The bctl binary was 8.8 MB versus 13 MB for beacon.
- Bare `bctl --no-refresh --json` consumed the live default config and `jq`
  verified work-scan schema version 1 with zero projects, work items, errors,
  and warnings. Bare `bctl` and `bctl scan` also rendered the same immediate
  empty human result.
- An isolated real terminal session ran `bctl projects`; Space entered an owner
  directory, Tab advanced to its repository, Space selected it, and Enter
  atomically saved that one exact project without legacy inventory.
- `beacon scan /tmp` and `beacon projects --root /tmp` both exited 2, proving
  the moved v2 forms are absent from the legacy executable.
- Both executables cross-compiled with `CGO_ENABLED=0` for darwin/arm64,
  darwin/amd64, linux/arm64, and linux/amd64. Shell syntax validation passed for
  the release and helper scripts.

## OUTCOME

`bctl projects` opens a lazy directory browser rooted at
`~/go/src/github.com` by default. Tab and the arrow keys move, Space enters an
ordinary directory or toggles a Git repository, `..` navigates outward without
crossing the chosen root, Enter atomically confirms, and Escape cancels. The
dedicated `projects` config list begins empty instead of inheriting the legacy
inventory, preserves selected projects outside the current browser root, and
permits a valid empty selection.

Bare `bctl` and `bctl scan` read that configured list and use the same small
deterministic work projection as positional `bctl scan PATH...`. All modes avoid
tracking, Following, cache, Notes, agent, and macOS state; hide clean base-only
repositories unless `--include-idle` is present; classify failed evidence as
unknown; and preserve scoped partial results. Positional paths remain a
config-free override.

`beacon` is again an unambiguous legacy surface: bare execution renders the
working-set dashboard, `beacon scan` runs the schema-v3 configured diagnostic,
and `beacon projects` manages Following. Release archives and source installs
contain both standalone executables, while the native macOS application embeds
only the legacy `beacon` helper.

Issue #76 remains represented by ready pull request #77 from exact branch
`GH-76`.

## REPOSITORY MEMORY

Created this specification because the scope reset, retained v1 fallback,
evidence exclusions, and rejected adjacent product surfaces are material
rationale that code and tests cannot preserve. Updated the specification,
Constitution, progress summary, README, and user guide with the demonstrated
`bctl` executable boundary, dedicated `projects` authority, explicit keyboard
and config mutation behavior, shared configured/positional projection, legacy
`beacon` compatibility, and two-binary release contract.
