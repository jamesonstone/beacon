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
    target: 0012-repository-sync-ui-refresh
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
  - id: pr-40
    name: Improve Following workspace clarity and themes
    type: github-pr
    target: https://github.com/jamesonstone/beacon/pull/40
    relation: verifies
    read_policy: evidence
    used_for: ready review and hosted validation
    status: active
  - id: issue-41
    name: Add lighthouse and restore configurable application typography
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/41
    relation: implements
    read_policy: must
    used_for: README masthead and installed-font Appearance continuation
    status: active
  - id: pr-42
    name: Customize Beacon branding and typography
    type: github-pr
    target: https://github.com/jamesonstone/beacon/pull/42
    relation: verifies
    read_policy: evidence
    used_for: ready review and hosted validation of the Appearance continuation
    status: active
  - id: issue-51
    name: Improve Notes AI and dashboard interactions
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/51
    relation: implements
    read_policy: must
    used_for: refresh-safe view-mode selection and hover taxonomy follow-up
    status: active
  - id: pr-52
    name: Improve Notes AI and dashboard interactions
    type: github-pr
    target: https://github.com/jamesonstone/beacon/pull/52
    relation: verifies
    read_policy: evidence
    used_for: existing ready pull request delivery lane for the interaction follow-up
    status: active
  - id: issue-55
    name: Add fitted Following dashboard view mode
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/55
    relation: implements
    read_policy: must
    used_for: fitted Following and Notes split-layout continuation
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
  - name: figma:figma-swiftui
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/figma/2.0.16/skills/figma-swiftui/SKILL.md
    trigger: native toolbar interaction changes from the user-provided screenshot
    required: true
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
  - name: github:gh-address-comments
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/github/0.1.8-2841cf9749ae/skills/gh-address-comments/SKILL.md
    trigger: live unresolved review feedback on the existing pull request
    required: true
  - name: computer-use:computer-use
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-bundled/computer-use/1.0.1000451/skills/computer-use/SKILL.md
    trigger: fresh native macOS interaction smoke
    required: true
---

# Following Workspace

## Thesis

Beacon should make every followed work lane understandable and movable without
turning presentation into a second workflow authority. One Go-owned user order,
three presentation densities, an adaptive experimental overview, and concise
exception evidence let the shared macOS workspace show substantially more work
while preserving factual status, deterministic next actions, and API reserves.
One shared installed-font preference keeps that workspace personally readable
without letting SwiftUI and AppKit typography drift apart.

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
- `Fit Following` is a fifth persisted view mode. It reserves the lower half of
  the dashboard for interactive Notes and dynamically scales every current
  Following lane into the remaining upper-half workspace without lane-area
  scrolling. Existing status sections and lane actions remain intact.
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
- Every read-only Markdown-backed field uses one block-aware renderer. Issue and
  pull-request descriptions plus review comments preserve headings, paragraphs,
  ordered/unordered/task lists, quotes, code, dividers, tables, inline emphasis,
  and links instead of flattening the document into one run of text. Ordinary
  interface labels and evidence strings remain explicit UI text.
- Issue and PR bodies are bounded at 64 KiB. A refresh retains at most 100
  unresolved threads and 20 comments per thread and makes truncation explicit.
  Details live only in the existing user-only cached evidence path.

### Theme Continuation

- The same issue #39, exact `GH-39` branch, and ready PR #40 continue after the
  complete Following delivery; no second delivery lane is created.
- Beacon ships exactly five built-in themes with stable IDs: Lobster Nebula,
  Pampas Moon, Solarized Dark, Monokai, and Selenized Dark. Lobster Nebula is
  the recommended default and Pampas Moon is the high-readability light theme.
- A theme is one complete semantic token set, not a collection of legacy neon
  color substitutions. Tokens cover canvas, layered surfaces, borders,
  primary/secondary/muted text, accent/focus, success/warning/danger/info,
  Local/PR/Issue identities, and every Markdown editor role.
- Theme selection is a live shared appearance preference for both the menu
  extra and detached dashboard. It persists through one stable AppStorage key,
  falls back safely when an unknown value is present, and updates the AppKit
  editor in the same render cycle.
- Ordinary text uses system UI typography. Monospaced typography is reserved
  for code, branches, identifiers, timestamps, percentages, and counters, and
  essential interface text remains at least 11 points.
- Ordinary canvas, card, control, tab, dialog, and border presentation uses
  solid neutral surfaces, restrained borders, and minimal shadow. Beacon keeps
  its playful gradient treatment only in the wordmark, rocket, and occasional
  illustration.
- Status meaning is invariant across themes and is always paired with explicit
  text and an SF Symbol. Increase Contrast and Differentiate Without Color
  strengthen borders and redundant cues; Reduce Transparency removes
  translucent overlays; Reduce Motion disables decorative and layout motion.
- Contrast tests use each token's declared sRGB value: normal text is at least
  4.5:1 against its intended surface and non-text or large indicators are at
  least 3:1. Accessible semantic aliases replace classic palette accents when
  raw colors cannot meet the role's threshold.

### Font Appearance Continuation

- Issue #41, exact branch `GH-41`, and ready PR #42 continue after the README
  masthead change because the user explicitly chose the existing delivery lane.
- `JetBrainsMono Nerd Font` is the exact default application font family.
- Settings → Appearance → Font lists every font family currently installed on
  the system and persists the user's explicit selection for both the menu extra
  and detached dashboard.
- The selected family applies live to ordinary interface copy, identifiers,
  code, counters, read-only Markdown, and the AppKit Signal Notes editor while
  preserving each role's existing size and weight.
- If the selected family is no longer installed, Beacon first falls back to the
  default family when available and otherwise uses the appropriate system UI or
  system monospaced fallback. Missing fonts never block application startup.
- The existing text-size and card-density preferences remain independent from
  the font-family choice.

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
13. Replace `BeaconPalette` and static neon role names with a typed semantic
    theme catalog containing exactly the five clarified complete token sets.
14. Give every theme a stable persisted ID, deterministic fallback, display
    metadata, dark/light appearance, and compact preview swatches; default to
    Lobster Nebula without rewriting an existing valid choice.
15. Apply the selected semantic theme live to the menu extra and detached
    dashboard, including lanes, tabs, controls, quick switcher, dialogs,
    empty/error states, Signal Notes, and the AppKit Markdown editor.
16. Add Settings → Appearance → Theme with all five names, a compact semantic
    color preview, a selected indicator, and an explicit recommended marker on
    Lobster Nebula and readability marker on Pampas Moon.
17. Preserve one label-and-symbol status grammar across themes, semantic
    Local/PR/Issue identities, quiet neutral surfaces, restrained borders,
    minimal shadows, and no ordinary gradients.
18. Use system UI type for ordinary copy and monospaced type only for code,
    branch, identifier, timestamp, percentage, and counter roles; keep essential
    rendered copy at least 11 points.
19. Respect Increase Contrast, Differentiate Without Color, Reduce
    Transparency, and Reduce Motion through native environment values without
    changing data or persisted workflow state.
20. Add automated catalog completeness, stable-ID/fallback, preference
    persistence, WCAG contrast, semantic identity, and rendered five-theme
    visual smoke coverage.
21. Update README, constitution, progress summary, this specification, and PR
    description/evidence for the completed theme continuation.
22. Commit and push the continuation only to `GH-39` / PR #40, then require the
    final local and hosted validation gates to pass on its exact head.
23. Replace the single-`Text` Markdown path with one theme-aware block renderer,
    route every read-only GitHub Markdown body/comment through it, and verify
    block separation, inline semantics, tables, task lists, text selection, and
    all-five-theme rendering without changing cached source text.
24. Restore one persisted application font-family preference with the exact
    default `JetBrainsMono Nerd Font` and deterministic missing-font fallback.
25. Enumerate every installed system font family in Settings → Appearance →
    Font and apply the selected family live to both shared macOS surfaces.
26. Route SwiftUI typography roles and AppKit Markdown editor fonts through the
    same selected family while retaining existing sizes, weights, and semantic
    styling.
27. Document and test default selection, installed-family enumeration,
    persistence, fallback, and shared SwiftUI/AppKit refresh behavior.
28. Keep the View Mode control and every layout choice interactive while a
    manual or background refresh publishes snapshot and progress updates.
29. Present the taxonomy guide after a deliberate information-control hover,
    preserve pointer traversal into the guide, and retain click pinning,
    outside/Escape dismissal, keyboard access, and zero refresh/network work.
30. Add persisted `Fit Following` as a no-scroll, scale-to-fit layout for every
    current Following lane above a fixed half-height Notes workspace. Preserve
    status order, refresh behavior, card actions, and all existing view modes.

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
- Theme IDs are product protocol and remain independent from display names.
- `Color`/`NSColor` are render forms derived from declared sRGB token values so
  automated contrast evidence and AppKit presentation share one source.
- Native macOS accessibility environment values are available on the minimum
  supported macOS 14 deployment target.
- AppKit's installed-family catalog is the local source of truth for font
  availability; no font download, bundling, or network lookup is required.

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
- [x] AC12: Canonical docs, focused tests, full Go/race/macOS/build/release/Kit
  checks, diff hygiene, and fresh-build visual interaction smoke pass before the
  ready PR is delivered.
- [x] AC13: The catalog contains exactly five complete stable-ID themes with
  Lobster Nebula as default/recommended and Pampas Moon as the only light theme.
- [x] AC14: Selection persists and updates menu, detached dashboard, editor,
  dialogs, switcher, states, lanes, tabs, controls, and Notes without restart.
- [x] AC15: Every ordinary surface uses semantic solid tokens with quiet borders
  and minimal shadow; no `BeaconPalette` or ordinary gradient implementation
  remains.
- [x] AC16: Appearance settings expose compact previews, selected state, names,
  recommendation/readability context, deterministic fallback, and stable IDs.
- [x] AC17: Status and Local/PR/Issue meaning remain label-and-symbol redundant
  and invariant under all themes and Differentiate Without Color.
- [x] AC18: System/monospaced typography roles, 11-point essential text, and the
  four clarified macOS accessibility preferences are applied consistently.
- [x] AC19: Automated completeness, fallback, persistence, contrast, identity,
  and rendered visual smoke tests pass across all five themes.
- [x] AC20: Canonical/user docs, complete local validation, fresh-build visual
  smoke, commit/push, PR #40 update, and final hosted checks are complete.
- [x] AC21: Every read-only Markdown-backed detail surface preserves block and
  inline formatting without concatenating text, remains selectable and linked,
  follows all five themes, and passes focused plus fresh-app visual validation.
- [x] AC22: With no saved family, Beacon selects `JetBrainsMono Nerd Font`; a
  valid installed family persists and updates both shared surfaces without
  restart.
- [x] AC23: Appearance exposes every installed family, and a removed, invalid,
  or unavailable selection deterministically falls back without startup or
  rendering failure.
- [x] AC24: SwiftUI copy and role fonts plus the AppKit Markdown editor resolve
  through the same selected family while preserving text-size, weight, theme,
  and density behavior.
- [x] AC25: During an active Following refresh, the View Mode menu remains open
  long enough to select any mode, applies that selection immediately, and does
  not cancel, restart, or otherwise alter the refresh.
- [x] AC26: Hovering the information control opens the complete taxonomy guide;
  moving into the guide keeps it open, click pins/unpins it, outside click or
  Escape dismisses it, and keyboard/accessibility activation still works.
- [x] AC27: Fit Following keeps Notes interactive in the lower half, displays
  every current Following lane simultaneously in the upper workspace without
  lane-area scrolling, and recomputes its scale for menu or window resizing.

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
6. Add one reusable block-aware Markdown document renderer, migrate issue, PR,
   and feedback content to it, and validate representative GitHub Markdown
   structures across themes before updating the existing delivery lane.
7. Explicitly stage, commit, push, create the assigned ready PR, observe hosted
   checks literally, and record final evidence.
8. Extend the delivered feature specification for the explicitly queued theme
   continuation and reconfirm the same issue, branch, and PR lane.
9. Introduce the semantic theme catalog, preference resolver, SwiftUI/AppKit
   render forms, and Settings appearance picker before migrating consumers.
10. Migrate every shared surface and typography/status role, then implement the
   native accessibility adaptations and remove the legacy palette API.
11. Add focused catalog, persistence, contrast, and visual rendering tests;
    update user/canonical docs and review the complete semantic-role inventory.
12. Run the full local and fresh-build visual gates, commit/push to PR #40,
    update delivery evidence, and wait for final hosted checks on the exact head.
13. Extend the active Appearance contract for the issue #41 font continuation
    and align its existing issue, branch, and PR lane.
14. Add the installed-family catalog, persistent Appearance picker, shared
    SwiftUI/AppKit font resolution, focused tests, and user documentation.
15. Run focused and complete macOS/Kit/diff validation, review the combined
    lane, commit/push to PR #42, and wait for hosted checks on the exact head.
16. Isolate the View Mode menu from unrelated observed refresh publications and
    use direct native menu actions so a live scan cannot invalidate selection.
17. Reuse Beacon's delayed hover/pointer-traversal interaction contract for the
    taxonomy guide, then validate both changes in a freshly built app.
18. Add the issue #55 persisted mode and a deterministic fitted-grid sizing
    model that preserves status sections and scales all lanes into its bounds.
19. Lock Notes presentation to the lower half while the fitted mode is active,
    restore the prior Notes size on exit, and cover transitions and resizing.
20. Run focused and complete validation, visually inspect both dashboard
    surfaces, and deliver the assigned ready issue #55 pull request.

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
- [x] T8: Commit, push, create the assigned ready PR, observe hosted checks, and
  record final evidence.
- [x] T9: Verify clean same-lane alignment after the Following delivery and add
  the queued theme contract to this active specification.
- [x] T10: Implement the complete semantic theme catalog, stable preference,
  appearance settings, and live shared application.
- [x] T11: Migrate all SwiftUI/AppKit surfaces, typography, status identity, and
  accessibility behavior; remove the legacy palette implementation.
- [x] T12: Add theme completeness, ID/fallback, persistence, WCAG contrast, and
  five-theme rendered visual smoke tests plus user documentation.
- [x] T13: Run all local and interaction gates, repair issues, commit/push to PR
  #40, update evidence, and require the final hosted checks to pass.
- [x] T14: Implement, document, test, and visually verify shared block-aware
  formatting for every read-only Markdown-backed detail surface.
- [x] T15: Record the issue #41 installed-font continuation in this active
  specification and align the existing delivery lane.
- [x] T16: Implement and document the JetBrainsMono default, installed-family
  Appearance picker, persistence, fallback, and shared font application.
- [x] T17: Add focused tests, run all relevant validation, review the combined
  diff, update issue/PR metadata, push, and verify final hosted checks.
- [x] T18: Reproduce and trace the refresh-time View Mode failure and document
  the interaction follow-up on the existing GH-51 / PR #52 lane.
- [x] T19: Implement refresh-independent direct View Mode actions and hover,
  traversal, pinning, dismissal, and accessibility behavior for taxonomy help.
- [x] T20: Add focused coverage, run local and native interaction validation,
  update PR #52, push the follow-up, and verify its exact final hosted head.
- [x] T21: Create assigned issue #55 and exact branch `GH-55` from refreshed
  `origin/main`, then record the accepted fitted-layout contract.
- [x] T22: Implement and test fitted Following geometry, half-height Notes,
  stable persistence, existing-mode compatibility, and refresh-safe selection.
- [ ] T23: Run full validation and native interaction smoke, review and stage
  explicitly, then deliver the assigned ready PR and verify its hosted head.

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
| AC13-AC16 | theme catalog, stable-ID/fallback/persistence, semantic-role inventory, settings preview, and shared-surface tests plus visual smoke |
| AC17-AC18 | status identity/label/symbol assertions, typography-role tests, accessibility environment variants, and visual/accessibility smoke |
| AC19 | token completeness and WCAG math tests plus AppKit-rendered preview smoke for every built-in theme |
| AC20 | full make gate, Linux builds, Kit/diff/secret review, stable-app interaction smoke, exact branch/PR recon, and hosted checks |
| AC21 | parser block/inline/table/task fixtures, five-theme render smoke, macOS test/build, and fresh detail-popover visual smoke |
| AC25 | stable View Mode control identity/action tests plus active-refresh native selection smoke across layouts |
| AC26 | hover timing contract tests plus native hover, traversal, click pin, outside/Escape, and keyboard accessibility smoke |
| AC22-AC24 | font-catalog/default/fallback tests, UserDefaults persistence assertions, SwiftUI/AppKit resolution tests, macOS test/build, Appearance inspection, and relaunch smoke |
| AC27 | fitted-grid geometry and mode-transition tests, 300-line audit, fresh menu/window resize smoke, full macOS validation, and hosted checks |

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
- Declared sRGB tokens made contrast defects visible before visual smoke. Testing
  every interface text and indicator role against canvas, base, raised, and
  overlay surfaces required quieter elevations for Pampas Moon, Solarized Dark,
  Monokai, and Selenized Dark while preserving their requested core palettes.
- One semantic catalog now drives SwiftUI, AppKit Markdown styling, the menu-bar
  beacon, both dashboard surfaces, and accessibility adaptations; themes cannot
  drift into separate status or identity grammars.
- Foundation's Markdown conversion retains document structure in
  `PresentationIntent` while its character stream omits the separators between
  headings, paragraphs, and list items. Rendering one whole attributed string
  therefore flattened the detail body; a shared block parser now preserves that
  structure without rewriting or mutating the cached GitHub source.
- The refresh scan publishes unrelated `AppState` changes while the native View
  Mode submenu is open. Isolating direct mode actions in an equatable child
  keeps the menu identity stable without pausing, cancelling, or restarting the
  scan.
- The taxonomy guide now reuses the detail popover's delayed hover and pointer
  traversal timings while retaining click, Escape, outside-dismissal, keyboard,
  and accessibility activation.
- Review of PR #52 found that the helper's bounded conversation input could
  eventually reject every follow-up. Requests now retain the newest complete
  turns within both helper limits while the entire transcript remains visible,
  and either panel initially scrolls to the newest visible turn.
- Enumerating candidate column counts makes the fitted grid deterministic and
  keeps its sizing model independent from SwiftUI rendering. Explicit rows,
  rather than a lazy container, ensure every scaled lane remains present and
  accessible even when its visual bounds are compact.
- Fresh-app accessibility inspection caught that a container-level label hid
  the individual lane labels. Removing that override preserved the complete
  child accessibility tree while keeping the visual fit unchanged.

## Documentation Updates

- Document Following ordering, density, Overview, canonical taxonomy, and rich
  evidence details in README.
- Update the constitution's Go authority, GitHub evidence model, public JSON,
  shared macOS surface, ordering, and API-budget contracts.
- Add feature 0018 to `docs/PROJECT_PROGRESS_SUMMARY.md` and advance it with
  evidence-backed workflow phases.
- Document the exact default family, installed-font Appearance selector, live
  shared application, persistence, and safe missing-font fallback in README.

## Delivery Decision

- Deliver issue #39 on exact branch `GH-39` in one ready pull request targeting
  `main`, assigned to `jamesonstone`, and closing issue #39.
- Use explicit staging, verified Jameson Stone author/committer identity, the
  repository PR template, and literal hosted-check reporting.
- Continue the user-approved font work on issue #41, exact branch `GH-41`, and
  ready PR #42; update their scope to cover both the README masthead and shared
  Appearance typography before delivery.
- Deliver the fitted Following continuation on issue #55, exact branch `GH-55`,
  in a new ready pull request targeting `main` and assigned to `jamesonstone`.

## Evidence

- Clarification completed at 100% confidence with zero unresolved decisions.
- Issue: https://github.com/jamesonstone/beacon/issues/39, assigned to
  `jamesonstone`.
- Branch: `GH-39`, created from refreshed `origin/main` at
  `e2237f7ff1c02e6aea5ddb1bf2a94b2838849f27` after clean `0/0` staleness recon.
- Implementation commit:
  `3da3b77496bbee1cd704aff70bd017e705862387` by Jameson Stone
  `<jameson@stone.tc>`.
- Ready pull request: https://github.com/jamesonstone/beacon/pull/40,
  targeting `main`, assigned to `jamesonstone`, and configured to close issue
  #39.
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
- Hosted checks passed on the implementation head: `go` in 57 seconds, `macos`
  in 2 minutes 20 seconds, and configured-maintainer assignment in 4 seconds.
- The Following delivery evidence head
  `6e19a6d9682096469bdf85432d783f7fa1382583` passed `go` in 55 seconds and
  `macos` in 2 minutes 4 seconds before the queued theme continuation began.
- Immediately before theme work, local `GH-39`, `origin/GH-39`, and PR #40 all
  resolved to `6e19a6d9682096469bdf85432d783f7fa1382583`; the tree was clean, the issue
  and ready PR were open and assigned to `jamesonstone`, and the base was `main`.
- Theme validation passed `make fmt-check vet test test-race build release-test
  macos-test macos-build`; the macOS suite executed 103 tests with zero failures,
  including exact catalog/ID/core-palette checks, all-surface WCAG matrices, and
  rendered semantic smoke fixtures for all five themes.
- Linux amd64/arm64 builds, all 18 Kit feature checks, `git diff --check`,
  changed-line secret review, legacy-palette/gradient audit, and source-size
  review passed.
- Fresh rebuilt-app smoke switched Lobster Nebula, Pampas Moon, Solarized Dark,
  Monokai, and Selenized Dark live across the detached dashboard and AppKit
  editor; the menu extra shared Lobster Nebula, Pampas Moon was readable in its
  light appearance, and Lobster Nebula remained selected after quit/relaunch.
- Theme implementation commit:
  `19d118ad63d1331903f1e43072558465fdd3e7c3` by Jameson Stone
  `<jameson@stone.tc>`, pushed to the existing `GH-39` branch and ready PR #40.
- PR #40 was updated in place to describe and validate both the completed
  Following workspace and semantic-theme continuation; it remains ready,
  targets `main`, is assigned to `jamesonstone`, and closes issue #39.
- Hosted checks passed on the theme implementation head: `go` in 50 seconds and
  `macos` in 2 minutes 1 second.
- The Markdown follow-up routes issue descriptions, pull-request descriptions,
  and review comments through one theme-aware renderer that preserves headings,
  paragraphs, unordered and ordered lists, task items, quotes, dividers, tables,
  fenced code, inline emphasis/code/links, text selection, and direct links.
- Focused parser and five-theme rendering tests pass, and the complete local
  gate passes `make fmt-check vet test test-race build release-test macos-test
  macos-build`; the macOS result contains 109 passing, zero failing, and zero
  skipped tests.
- Linux amd64/arm64 builds, all 18 Kit feature checks, `git diff --check`,
  changed-line secret review, Markdown-consumer audit, and source-size review
  pass for the follow-up.
- Fresh rebuilt-app smoke against `lsmc-bio/terrarium` issue #105 confirms the
  supplied body now displays distinct Original ask, Scope boundaries,
  Acceptance criteria, and Expected verification sections with readable list
  markers and spacing in the detailed hover panel.
- Markdown-formatting implementation commit:
  `02673786fc53d36a6ce4b3e6f22152cb5ad0a78b` by Jameson Stone
  `<jameson@stone.tc>`, pushed to the existing `GH-39` branch and ready PR #40.
- PR #40 was updated in place with the shared Markdown renderer, root cause,
  coverage, and exact visual verification step while preserving its template,
  ready state, assignment, base, and issue closure.
- Hosted checks passed on the Markdown implementation head: `go` in 56 seconds
  and `macos` in 2 minutes 53 seconds.
- Issue #41 continues on branch `GH-41` and ready PR #42 after the user chose
  the existing lane for the installed-font Appearance follow-up.
- The typography implementation uses `JetBrainsMono Nerd Font` by default,
  lists all installed AppKit font families under Settings → Appearance → Font,
  persists the selection, and routes SwiftUI plus AppKit font roles through one
  fallback-aware resolver.
- Five focused typography tests pass for the exact default, installed-family
  enumeration, persistence, missing-font fallback, and shared AppKit role
  resolution.
- The complete local gate passes `make fmt-check vet test test-race build
  release-test macos-test macos-build`; the fresh macOS suite executed 113
  tests with zero failures, and the fresh macOS application build succeeded.
- Linux amd64/arm64 builds, all 18 Kit feature checks, `git diff --check`,
  changed-file secret-pattern review, system-font override audit, and source-size
  review pass for the typography continuation.
- Typography implementation commit
  `52e90fa68851670c21c9d70e05ff2992811bfc05` by Jameson Stone
  `<jameson@stone.tc>` is pushed to `GH-41` and ready PR #42.
- Issue #41 and PR #42 now describe both the six-line README lighthouse and the
  configurable application-font continuation while preserving the assigned,
  ready, `main`-targeting delivery lane.
- Hosted checks passed on the typography implementation head: `go` in 54
  seconds and `macos` in 1 minute 59 seconds.
- A fresh built-app smoke opened with `JetBrainsMono Nerd Font`, exposed all 218
  installed families in Appearance, switched live to Menlo, restored the
  default, and retained that restored choice after quit/relaunch; the existing
  background agent remained untouched.
- The GH-51 interaction follow-up keeps View Mode selectable during an active
  Following refresh and adds delayed hover presentation to the taxonomy guide
  with traversal, click, outside-click, Escape, keyboard, and accessibility
  behavior preserved.
- Focused tests pass for stable View Mode identity, taxonomy hover traversal,
  bounded AI request history, and full visible transcript preservation.
- The complete local gate passes `make BEACON_DERIVED_DATA=/tmp/beacon-gh51-followup-final
  fmt-check vet test test-race release-test build macos-test macos-build`; the
  macOS suite executed 145 tests with zero failures and zero skips.
- All 20 feature checks pass `kit check --all`; `git diff --check`, plist lint,
  changed-file secret-pattern review, and the six touched Swift source-size
  checks pass. `kit check --project` retains three pre-existing instruction
  scaffold findings in files outside this change.
- Fresh built-app smoke changed Stacked to Horizontal Tiles while the refresh
  control remained busy, with the same scan continuing. Pointer-only hover
  opened the taxonomy guide, traversal into the guide preserved it, moving away
  closed it, and click plus Escape preserved explicit activation and dismissal.
- Interaction and review-repair implementation commit
  `97d9ca4625df5e59ba98b71ec689f729d247718c` by Jameson Stone
  `<jameson@stone.tc>` is pushed to the existing `GH-51` branch and ready PR
  #52.
- Issue #51 and PR #52 are open, assigned to `jamesonstone`, aligned to
  `GH-51` targeting `main`, and describe the Notes AI plus dashboard interaction
  scope while preserving the repository template and issue closure.
- Hosted checks passed on the implementation head: `go` in 1 minute 7 seconds
  and `macos` in 2 minutes 37 seconds. CodeRabbit completed successfully, and
  its sole current review thread is resolved as addressed by the implementation
  commit.
- Issue #55 is assigned to `jamesonstone`; branch `GH-55` was created from
  refreshed `origin/main` at `1a283a6a28513620b2bf3f1786b9fcb83e7f0715`.
- Fit Following persists as the fifth view mode, fits all 12 live Following
  lanes into the upper workspace at both normal and compact window sizes, and
  keeps interactive Notes fixed to the lower half without a lane-area scroll.
- Five focused fitted-layout and transition tests plus the two existing mode
  contract tests pass. The complete local gate passes `make
  BEACON_DERIVED_DATA=/tmp/beacon-gh55-final fmt-check vet test test-race
  release-test build macos-test macos-build`; the macOS suite executed 150
  tests with zero failures and zero skips.
- Linux amd64 build, all 20 Kit feature checks, project-contract validation,
  plist lint, `git diff --check`, and the application/code source-size audit
  pass; zero Go, Swift, shell, JavaScript, TypeScript, or Python files exceed
  300 lines.
- Fresh built-app smoke confirmed all 12 lane actions and accessibility labels,
  half-height Notes, persisted relaunch state, compact resize recomputation, and
  live View Mode interaction while the scan control remained busy.
