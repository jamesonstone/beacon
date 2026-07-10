---
kit_metadata_version: 1
artifact: spec
workflow_version: 2
phase: deliver
delivery_intent: issue_branch_pr_in_progress
clarification:
  status: ready
  confidence: 100
  unresolved_questions: 0
feature:
  id: "0002"
  slug: beacon-init-dashboard
  dir: 0002-beacon-init-dashboard
relationships:
  - type: builds_on
    target: 0001-beacon-v1
references:
  - id: issue-1
    name: Implement Beacon work-lane review radar
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/1
    relation: supports
    read_policy: conditional
    used_for: existing delivery lane selected by the user
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: architecture, read-only, schema, and delivery invariants
    status: active
skills: []
---

# Beacon Init and Progress Dashboard

## Thesis

Beacon should be useful immediately after installation. Guided initialization discovers local GitHub repositories, while the default command presents a project-grouped control plane derived from local Git, GitHub issues and pull requests, automation results, unresolved feedback, and optional Kit progress evidence.

## Context

Beacon v1 requires users to hand-write every repository entry and exposes lanes only through `beacon scan`. Large source trees make manual configuration tedious, and pull-request readiness alone does not explain the last durable feature phase or the next human action. This feature adds persistent source roots, safe configuration merging, issue and review evidence, deterministic progress summaries, a colorful dashboard, and matching macOS presentation.

The user explicitly selected issue `#1`, branch `GH-1`, and PR `#2` for delivery despite this feature being documented separately from completed v1.

## Clarifications

- Source roots are persisted and rediscovered on every scan.
- GitHub scope is configurable as `mine` or `all`, defaulting to `mine`.
- `mine` means authored PRs and assigned issues for `settings.github_author`.
- Bare `beacon init` is a native keyboard-driven selector; repeated `--source` adds entire roots without repository selection.
- Existing configs merge non-destructively and are rewritten only after confirmation.
- Missing tools are diagnosed with installation guidance; Beacon never installs packages automatically.
- Unauthenticated interactive users may choose to run `gh auth login`.
- Actionable feedback means unresolved review threads or changes-requested reviews, not ordinary comments.
- Kit progress documents are optional evidence and never a hard dependency.
- Passing automation with no approval recommends manual testing followed by merge; an approved clean PR recommends merge.
- The CLI and macOS app consume the same schema-v2 snapshot.
- The menu-bar label counts non-idle ready, action, and waiting lanes. With no active lanes, it displays a compact color neon-space glyph.

## Requirements

1. Support strict config versions 1 and 2, with v2 source roots and GitHub scope.
2. Discover Git repositories recursively from persisted sources without following symlinks, and deduplicate repositories and worktrees deterministically.
3. Implement `beacon init` with repeated sources, interactive selection, prerequisite checks, safe merging, preview, confirmation, and atomic writes.
4. Make bare `beacon` run the dashboard and offer initialization when config is missing in a TTY.
5. Query relevant open PRs and issues through `gh`, enrich PRs with checks, linked issues, reviews, comments, mergeability, and unresolved threads, and preserve partial results.
6. Parse optional Kit progress summaries and SPEC issue references without inventing progress percentages.
7. Emit schema version 2 with projects, issues, feedback, check summaries, progress evidence, and new deterministic actions.
8. Render a project-grouped, terminal-width-aware, color-controlled table while keeping JSON ANSI-free.
9. Update the macOS app to decode, group, display, and open schema-v2 project, issue, feedback, and progress evidence.
10. Preserve Beacon's read-only scanning boundary; only explicit confirmed initialization may write configuration.

## Assumptions

- GitHub remains the only remote provider.
- Source discovery may find more than 100 repositories, so `mine` uses global GitHub search before detailed enrichment.
- Search results are capped at 1,000 with explicit truncation warnings.
- Explicit repository entries override discovered metadata.
- Version-1 configs continue to scan without mutation and upgrade only through confirmed init.
- Huh and Lip Gloss are acceptable focused dependencies for terminal interaction and styling; Viper and a table framework remain out of scope.

## Acceptance Criteria

- [x] AC1: Version-1 configs still load, and version-2 configs strictly validate sources and `mine|all` scope.
- [x] AC2: Repeated source flags and interactive selection create or safely merge an atomic config without deleting existing entries.
- [x] AC3: Repository roots, parent roots, worktree `.git` files, overlapping roots, non-GitHub remotes, missing paths, and unusual names are handled deterministically.
- [x] AC4: Bare Beacon displays the same ordered snapshot as human `scan`, offers init only in a TTY, and keeps non-TTY behavior explicit.
- [x] AC5: Mine/all issue and PR discovery, detailed checks, linked issues, feedback, and unresolved threads are normalized through bounded `gh` commands.
- [x] AC6: PR, local branch, issue, and Kit feature correlation follows the documented precedence and retains unmatched issue-only work.
- [x] AC7: Every new action follows fixed precedence, including waiting for CI, manual test then merge, direct merge after approval, and starting an issue.
- [x] AC8: Schema-v2 JSON contains projects and enriched lanes, uses non-null arrays, is deterministic, and never contains ANSI escapes.
- [x] AC9: Terminal output respects `auto|always|never`, `NO_COLOR`, TTY width, and project grouping.
- [x] AC10: The macOS app decodes schema v2, displays the same evidence, opens issue-only lanes, retains last-good results, and switches its menu-bar label between the non-idle count and a neon-space zero-state glyph.
- [x] AC11: Malformed Kit or one-repository GitHub evidence produces scoped warnings/errors without suppressing healthy projects.
- [x] AC12: Formatting, vet, unit, race, CLI, config-isolation, and Xcode validations pass, and inspection confirms scanning remains read-only.

## Implementation Plan

1. Add config-v2 types, atomic serialization, repository discovery, prerequisite checks, and init interaction behind testable boundaries.
2. Extend the public model to schema v2 and add issue, feedback, checks, project, and Kit progress evidence.
3. Implement scalable GitHub mine/all discovery, detailed PR enrichment, unresolved-thread queries, and deterministic correlation.
4. Extend policy and ordering, share the root/scan runner, and render the adaptive colored dashboard.
5. Update the Swift models, state, menu, and open behavior for schema-v2 parity.
6. Complete documentation, validation, read-only verification, reflection, and delivery on the existing PR.

## Task Checklist

- [x] T1: Implement and test config v2, discovery, and guided init.
- [x] T2: Implement and test schema-v2 domain types and GitHub/issue/feedback collection.
- [x] T3: Implement and test Kit progress parsing and evidence correlation.
- [x] T4: Implement and test policy, ordering, bare command behavior, and colored output.
- [x] T5: Implement and test macOS schema-v2 parity.
- [x] T6: Update constitution, progress summary, README, and CLI/config documentation.
- [x] T7: Run the full validation map and perform read-only verification.
- [x] T8: Reflect, record evidence, commit, push, and update PR #2.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC3 | Config/discovery unit tests and temporary Git repository integration tests |
| AC4 | Cobra root/init tests with injected TTY and prompt boundaries |
| AC5-AC7 | GitHub fixtures, GraphQL fixtures, correlation tests, and policy tables |
| AC8-AC9 | JSON and terminal golden tests across color/width modes |
| AC10 | Swift Codable, state, grouping, and open-target tests |
| AC11 | Partial-failure and malformed-progress tests |
| AC12 | `make fmt-check vet test test-race build macos-test`, isolated-HOME smoke tests, diff review, and command inventory |

## Reflection Notes

- Keeping source roots durable but discoveries ephemeral preserves a simple configuration model without introducing the history database that v2 explicitly excludes.
- Global `mine` searches avoid multiplying basic GitHub queries across large source trees; `all` retains bounded repository-local collection for the intentionally broader scope.
- Current review decisions and each reviewer's latest state drive actions. Ordinary comments and superseded review events remain evidence without becoming false blockers.
- Kit documents remain optional and repository-scoped. Exact issue URLs correlate durable phase evidence, while malformed or stale documents only add diagnostics.
- Terminal tables are laid out before ANSI styling so color bytes cannot distort columns. Narrow output uses Lip Gloss visible-width wrapping for Unicode and styled text.
- The read-only verifier found three edge cases before delivery: mine-search truncation counting, Git-common-directory override identity, and narrow wrapping. Each was repaired with focused regression coverage; source paths were also hardened to canonicalize symlinked ancestors while rejecting a final symlink.

## Documentation Updates

- Update `docs/CONSTITUTION.md` for config/schema v2, source discovery, issue/feedback evidence, Kit progress, and terminal UI dependencies.
- Update `docs/PROJECT_PROGRESS_SUMMARY.md` whenever this feature phase advances.
- Update `README.md` with initialization, source, scope, bare dashboard, color, and migration examples.

## Delivery Decision

Continue on issue `#1`, branch `GH-1`, and ready PR `#2`, as explicitly directed by the user. Do not create another delivery lane.

## Evidence

- `make fmt-check`, `make vet`, `make test`, `make test-race`, and `make build` passed after the verifier repairs.
- `make macos-build` succeeded for the macOS 14 universal helper/app target.
- `make macos-test` passed all 14 Swift tests, including non-idle menu-bar counting and the zero-state path. Xcode emitted non-fatal `linkd.autoShortcut` host-service messages; the suite ended `TEST SUCCEEDED`.
- An isolated-`HOME` smoke test ran `beacon init --source <repository> --yes`, loaded the resulting version-2 config, and ran the bare dashboard without touching the real user config.
- Live `beacon scan --no-refresh --json` returned schema version 2 with one project, PR #2, issue #1, Kit feature `0002`, two passing checks, zero unresolved threads, non-null arrays, and no scan errors.
- CLI smoke checks confirmed `--color=always` emits ANSI, `NO_COLOR`/non-TTY output does not, invalid color and non-interactive bare init exit 2, and missing config exits 1 with an init hint.
- `beacon doctor --json` passed Git, `gh`, authentication, config-directory, repository, and GitHub-access checks.
- `kit check --all --project` reported that the project contract is coherent.
- Independent read-only verification confirmed all reported edge cases were repaired and found no remaining findings after a final 20-column output check.
- Production process execution uses `exec.CommandContext` with argument arrays. Inspection found no scanner writes beyond the documented bounded fetch; explicit confirmed init writes only Beacon's config through same-directory atomic replacement.
- Implementation commit `7572d9c19a99e38a253b9a9b3d866f1f43f0179e` was pushed to `origin/GH-1` under Jameson Stone's author and committer identity.
- Ready PR [#2](https://github.com/jamesonstone/beacon/pull/2) was updated in place with the required Conventional Commit title, template headings, schema-v2 description, verification steps, `Closes #1`, and `jamesonstone` assignment.
