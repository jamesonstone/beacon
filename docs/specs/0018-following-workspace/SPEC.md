---
kit_metadata_version: 1
artifact: spec
workflow_version: 2
phase: validate
delivery_intent: ready_pull_request
clarification:
  status: ready
  confidence: 100
  unresolved_questions: 0
feature:
  id: "0018"
  slug: following-workspace
  dir: 0018-following-workspace
relationships:
  - type: builds_on
    target: 0008-github-api-budget
  - type: builds_on
    target: 0009-beacon-working-set-radar
  - type: builds_on
    target: 0010-project-following
  - type: builds_on
    target: 0017-beacon-focus-notes
references:
  - id: issue-39
    name: Improve Following workspace density and clarity
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/39
    relation: implements
    read_policy: must
    used_for: clarified requirements and ready pull request lane
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: Go authority, evidence policy, API budget, schema, and thin macOS boundary
    status: active
  - id: github-api-budget
    name: GitHub API budget
    type: spec
    target: docs/specs/0008-github-api-budget/SPEC.md
    relation: constrains
    read_policy: must
    used_for: cached budgeted rich evidence collection
    status: active
  - id: working-set-radar
    name: Beacon working-set radar
    type: spec
    target: docs/specs/0009-beacon-working-set-radar/SPEC.md
    relation: informs
    read_policy: must
    used_for: lane state, ordering, tags, evidence badges, and shared layouts
    status: active
  - id: project-following
    name: Project Following
    type: spec
    target: docs/specs/0010-project-following/SPEC.md
    relation: informs
    read_policy: must
    used_for: membership, lane visibility, and Parking Lot behavior
    status: active
skills:
  - name: github:github
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/github/0.1.8-2841cf9749ae/skills/github/SKILL.md
    trigger: GitHub issue and repository delivery orientation
    required: true
  - name: github:yeet
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/github/0.1.8-2841cf9749ae/skills/yeet/SKILL.md
    trigger: user-requested branch, commit, push, and pull request delivery
    required: true
---

# Following Workspace

## Thesis

Beacon should make every followed work lane understandable and movable without
turning presentation into a second workflow authority. One Go-owned user order,
three presentation densities, an adaptive experimental overview, and concise
exception evidence let the shared macOS workspace show substantially more work
while preserving factual status, deterministic next actions, and API reserves.

## Context

The current shared dashboard supports stacked, horizontal-tile, and
experimental kanban layouts over schema-v3 lanes. Kanban places every Active
lane in one narrow scrolling column, while each card repeats healthy worktree,
CI, review, and freshness values. The half-height Notes panel further limits
the lane viewport. Lane order is deterministic but cannot be personalized, and
Ignore is the only direct movement action.

Evidence badges and optional user tags are separate systems, but both appear as
small pills. The emphasized `2 Unresolved` badge identifies a PR review-thread
count without saying so. Swift tooltips cannot present the issue body or the
individual review threads because the Go model currently retains only issue
metadata and aggregate feedback counts. Rich details therefore require bounded
collection through the existing GitHub cache and a coordinated additive model.

## Clarifications

- A Following item is a work-lane card, not a row in Manage Following.
- Users reorder lanes through a visible drag handle. Card click continues to
  open the work item, and card hover or keyboard focus presents detail.
- Reordering is durable Go-owned state shared by CLI, agent, menu, detached
  dashboard, and every layout. One global lane order is projected into the
  evidence-derived Active, Waiting, Recent, and Parked sections.
- Dragging within Following changes order only. It cannot override Active,
  Waiting, or Recent. Dragging to Parking Lot invokes the existing Ignore
  mutation; dragging back invokes Resume and lets Go select the derived group.
- A lane retains relative priority when evidence moves it to another section.
  A newly discovered lane enters at the top of its derived section. Pinning
  remains independent and neither locks nor overrides order.
- Density is independent from font size and persists across both macOS
  surfaces. Comfortable shows full approved content, Compact shows identity,
  next action, age/delta, and exceptions, and Dense shows identity, next action,
  and one exception summary.
- `Overview (Experimental)` is a fourth persisted view mode. It uses adaptive
  dense grids, collapses empty sections, and temporarily minimizes Notes while
  restoring the prior Notes size on exit.
- At the supplied detached-window size and default font, Overview must show the
  eleven current Following lanes without lane-area scrolling. Smaller windows
  and the 430-point menu may scroll. Parking Lot, Recently Updated, and Quiet
  remain separate tabs.
- Canonical taxonomy is presentation over existing policy: project membership,
  lane attention, one next action, evidence exceptions, then optional user
  context. It does not create a linear project-management workflow.
- Healthy worktree, CI, review, and freshness values are hidden by default.
  Actionable exceptions use concise labels including `Local changes`,
  `CI pending`, `CI failed`, `Stale`, and `PR feedback · <count>`.
- User tags remain optional free-form local context and are never created
  automatically.
- One info control beside View and Settings opens the taxonomy/status guide,
  including hierarchy, statuses, colors, order, Parking Lot, and freshness.
- Issue cards expose full bounded issue detail. PR cards expose bounded PR and
  linked-issue detail. A PR-feedback badge exposes each collected unresolved
  thread with synthesized file/line and author heading, comment content,
  dates, and individual links. Local and manual cards expose available path,
  action, reason, warning, blocker, note, and tag context.
- Rich detail opens after hover or keyboard focus, remains open while traversed,
  can be pinned by click, closes with Escape, and never initiates network work.
- Issue and PR bodies are bounded at 64 KiB. A refresh retains at most 100
  unresolved threads and 20 comments per thread and makes truncation explicit.
  Details live only in the existing user-only cached evidence path.

## Requirements

1. Extend versioned lane state with a normalized global lane order that
   migrates older files additively, removes stale IDs, preserves valid relative
   order, and inserts newly observed lanes at the front of their derived group.
2. Add atomic complete-order mutation through the working-set manager, agent
   protocol, CLI, and Swift clients. Invalid duplicate,
   unknown, missing, or incomplete orders fail without changing state.
3. Apply the user order consistently to all working-set groups without changing
   evidence-derived attention, next actions, pins, candidate rules, or top-item
   eligibility.
4. Add accessible lane drag handles to shared card presentation. Reordering
   operates within a displayed section; Following/Parking drop targets use the
   existing shared Ignore/Resume mutations.
5. Add persisted Comfortable, Compact, and Dense density values with one shared
   card renderer that exposes the clarified information contract in stacked,
   tile, kanban, and overview layouts.
6. Add persisted `Overview (Experimental)` as an adaptive dense-grid layout that
   collapses empty groups, uses the available width, preserves ordered lanes,
   minimizes Notes on entry, and restores its prior size on exit.
7. Replace the always-on evidence-badge row with exception-only canonical
   evidence derived from existing signals. Preserve reversible exact-value
   dismissal for displayed exceptions and retain user tags as separate context.
8. Add an accessible taxonomy information control with the complete clarified
   membership, attention, action, evidence, context, color, ordering, Parking
   Lot, and freshness explanations.
9. Add bounded body fields to issue and pull-request evidence and bounded typed
   unresolved thread/comment detail to PR feedback. Preserve partial results,
   array-not-null JSON, deterministic ordering, and old-cache decoding.
10. Collect rich evidence only in the existing budgeted refresh paths, cache it
    with the existing user-only command-response and project caches, and perform
    zero GitHub or Git commands on hover, focus, popover pin, or dismissal.
11. Add shared rich detail presentation for issue, PR, feedback, local, and
    manual lanes with native Markdown, direct links, truncation disclosure,
    delayed hover, click pinning, pointer traversal, Escape, and keyboard focus.
12. Reconcile README, constitution, progress summary, and this specification,
    then deliver the complete issue #39 scope in one ready pull request.

## Assumptions

- The current schema-v3 contract accepts additive optional fields; older cached
  snapshots decode with empty rich details and do not require a schema bump.
- The current protocol-v1 request envelope accepts an additive order request
  without changing existing event shapes.
- The existing rate guard continues to conservatively debit each GraphQL cache
  miss; bounded detail is fetched with PR enrichment rather than a hover path.
- GitHub review threads do not have user-authored titles, so presentation derives
  a stable heading from path, line, and author without inventing feedback text.
- Native SwiftUI drag/drop and popover APIs available on macOS 14 are sufficient;
  no dependency or image asset is required.
- Detail truncation is a visible partial result, not a collection failure.

## Acceptance Criteria

- [x] AC1: Lane order persists atomically across reload, agent restart, all
  working-set sections, both macOS surfaces, and every layout.
- [x] AC2: Invalid or concurrent reorder requests preserve a complete unique
  order; stale IDs disappear; new lanes enter at the top of their derived group.
- [x] AC3: Drag handles reorder without hijacking card open/detail gestures;
  Following/Parking drops map only to Resume/Ignore; pinning and policy remain
  unchanged.
- [x] AC4: Comfortable, Compact, and Dense persist and render exactly the
  clarified content across the menu and detached dashboard.
- [x] AC5: Overview collapses empty groups, preserves ordered status sections,
  minimizes/restores Notes, fits eleven lanes at the supplied window size, and
  degrades safely with scrolling at smaller sizes.
- [x] AC6: Canonical exception badges hide healthy values, label review threads
  `PR feedback · <count>`, remain dismissible by exact value, and never mingle
  with optional user tags.
- [x] AC7: The information control accurately explains the complete canonical
  hierarchy, statuses, visual identities, ordering, Parking Lot, and freshness.
- [x] AC8: Additive JSON and Swift models preserve bounded issue/PR bodies and
  deterministic unresolved thread/comment details with explicit truncation and
  old-cache compatibility.
- [x] AC9: Rich detail panels expose all clarified type-specific content,
  individual links, hover traversal, click pinning, Escape, and keyboard focus
  without network work.
- [x] AC10: Detail collection remains read-only, budgeted, cached,
  partial-failure tolerant, and zero-work on subscription or hover.
- [x] AC11: Existing lane policy, Following membership, top-item actions,
  external activity, notes, CLI output purity, API reserves, and release behavior
  do not regress.
- [ ] AC12: Canonical docs, focused tests, full Go/race/macOS/build/release/Kit
  checks, diff hygiene, and fresh-build visual interaction smoke pass before the
  ready PR is delivered.

## Implementation Plan

1. Record issue #39, clarified decisions, acceptance criteria, validation map,
   and ready-PR lane in this v2 specification.
2. Implement Go-owned lane order persistence, mutation, protocol/CLI plumbing,
   deterministic projection, migration, and tests.
3. Implement bounded issue, PR, review-thread, and comment detail collection,
   caching compatibility, model/fixture updates, and tests.
4. Implement shared Swift models, clients, AppState mutations, drag/drop, density,
   overview, canonical badges/help, rich details, and accessibility tests.
5. Reconcile documentation, run focused and full validation, perform fresh-build
   interaction smoke, and review the complete diff.
6. Explicitly stage, commit, push, create the assigned ready PR, observe hosted
   checks literally, and record final evidence.

## Agent Team Plan

- One supervisor owns the specification, Go/Swift contract, implementation,
  integration, validation, documentation, and delivery.
- No subagents are used because ordering, evidence shape, cache compatibility,
  client decoding, and shared UI behavior form one tightly coupled contract.

## Task Checklist

- [x] T1: Create assigned issue #39 and exact branch `GH-39` from refreshed
  `origin/main` after a clean delivery preflight.
- [x] T2: Create this ready v2 specification and progress entry.
- [x] T3: Implement and test Go lane-order persistence, projection, mutations,
  protocol, and CLI behavior.
- [x] T4: Implement and test bounded rich GitHub evidence collection and
  additive cache/model compatibility.
- [x] T5: Implement and test shared macOS density, Overview, drag/drop,
  canonical taxonomy/help, and rich detail interaction.
- [x] T6: Reconcile README, constitution, and project progress documentation.
- [x] T7: Run focused and full validation, visual smoke, secret scan, and diff
  review; repair every relevant issue.
- [ ] T8: Commit, push, create the assigned ready PR, observe hosted checks, and
  record final evidence.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1-AC3 | working-set state/mutation/projection tests, protocol mutation tests, CLI tests, Swift request/AppState/drag tests |
| AC4-AC5 | density/overview presentation tests, shared-surface persistence assertions, Xcode build, fresh detached/menu visual smoke |
| AC6-AC7 | Swift badge vocabulary/dismissal and taxonomy content/accessibility tests plus visual smoke |
| AC8 | GitHub parser/query tests, JSON fixtures, cache migration tests, Swift model decoding tests |
| AC9 | Swift detail-content, link, hover/pin/Escape/focus tests and manual interaction smoke |
| AC10 | runner command-count, cache-hit, protected-budget, partial/truncation, and zero-hover-command tests |
| AC11 | existing Go, race, CLI, agent, Swift, notes, external-activity, and release suites |
| AC12 | `make fmt-check vet test test-race build release-test macos-test macos-build`, Linux build, `kit check --all`, `git diff --check`, secret review, and PR checks |

## Reflection Notes

- Keeping the durable order in the existing Go working-set state avoided a
  second Swift preference authority and made CLI, menu, and detached behavior
  converge on the same complete-order validation.
- Rich bodies and review comments fit the existing GitHub budget because the
  previous review-thread request was enriched in place; hover and focus remain
  cache-only presentation paths.
- Current-schema caches created before this feature decode new arrays as nil,
  so snapshot finalization normalizes order, threads, and comments to empty
  arrays before any JSON reaches clients.
- A per-lane evidence-hover coordinator prevents card and feedback popovers
  from racing when the pointer is over `PR feedback · N`.
- The live working set contained twelve active lanes, one more than the supplied
  acceptance fixture; Overview displayed all twelve in one detached frame with
  Notes minimized and no lane-area scrolling.

## Documentation Updates

- Document Following ordering, density, Overview, canonical taxonomy, and rich
  evidence details in README.
- Update the constitution's Go authority, GitHub evidence model, public JSON,
  shared macOS surface, ordering, and API-budget contracts.
- Add feature 0018 to `docs/PROJECT_PROGRESS_SUMMARY.md` and advance it with
  evidence-backed workflow phases.

## Delivery Decision

- Deliver issue #39 on exact branch `GH-39` in one ready pull request targeting
  `main`, assigned to `jamesonstone`, and closing issue #39.
- Use explicit staging, verified Jameson Stone author/committer identity, the
  repository PR template, and literal hosted-check reporting.

## Evidence

- Clarification completed at 100% confidence with zero unresolved decisions.
- Issue: https://github.com/jamesonstone/beacon/issues/39, assigned to
  `jamesonstone`.
- Branch: `GH-39`, created from refreshed `origin/main` at
  `e2237f7ff1c02e6aea5ddb1bf2a94b2838849f27` after clean `0/0` staleness recon.
- Focused Go validation passed for `internal/workset`, `internal/githubscan`,
  `internal/agent`, `internal/scan`, and `internal/cli`.
- Full local validation passed: `make fmt-check vet test test-race build
  release-test macos-test macos-build`; the macOS result contained 92 passing,
  zero failing, and zero skipped tests.
- Linux amd64 and arm64 builds, `kit check --all` for all 18 features,
  `git diff --check`, changed-line secret-pattern review, and source-size review
  passed.
- Fresh stable-build interaction smoke confirmed 12 ordered active cards in one
  Overview frame, Notes minimize/restore, all three persisted density choices,
  Following/Parking drop targets, keyboard Move Up/Down actions, explicit
  exception labels including `PR feedback · N`, the complete taxonomy popover,
  and cached issue/PR Markdown detail with direct links and Escape dismissal.
