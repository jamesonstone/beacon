---
kit_metadata_version: 1
artifact: spec
workflow_version: 3
phase: deliver
delivery_intent: ready_pull_request
feature:
  id: 0021
  slug: animated-project-watermarks
  dir: 0021-animated-project-watermarks
relationships:
  - type: builds_on
    target: 0018-following-workspace
references:
  - id: issue-57
    name: Show animated project watermarks in Fit Following
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/57
    relation: implements
    read_policy: must
    used_for: original request, scope, acceptance criteria, and delivery lane
    status: active
  - id: pr-58
    name: Animated project watermarks delivery
    type: github-pr
    target: https://github.com/jamesonstone/beacon/pull/58
    relation: implements
    read_policy: must
    used_for: review, hosted validation, and merge boundary
    status: active
  - id: following-workspace
    name: Following workspace
    type: spec
    target: docs/specs/0018-following-workspace/SPEC.md
    relation: informs
    read_policy: must
    used_for: fitted geometry, dense-card taxonomy, themes, and accessibility contracts
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: no-scroll fitted layout, semantic themes, and accessibility invariants
    status: active
skills:
  - name: figma:figma-swiftui
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/figma/2.0.16/skills/figma-swiftui/SKILL.md
    trigger: native SwiftUI card hierarchy and semantic theme treatment
    required: true
---
# Animated Project Watermarks

## PURPOSE

Fit Following should make the repository behind each lane immediately legible
without surrendering its defining promise that every current Following lane
remains visible above interactive Notes. Each fitted card will therefore use
its otherwise quiet background as a playful identity surface: the complete
project name appears oversized and faint across the card while a slow,
theme-aware color sweep moves through the letters behind the factual lane
content.

## CONTEXT

The fitted dashboard owns a deterministic `220 x 88` point card and scales the
complete status grid into the available upper workspace. It calls the shared
dense lane card, whose only project identity is currently a one-letter glyph.
The stacked layout already resolves the canonical project name through shared
`AppState`, so this change requires no Go model, schema, cache, network, or
workflow-authority work.

Beacon's five themes already provide semantic foreground and surface colors
with automated WCAG checks. A bright semantic accent placed directly behind
lane copy would erode that contrast, especially in Selenized Dark and the light
Pampas Moon theme. Watermark colors must therefore be theme-owned, near-surface
background colors whose composited role remains faint while every existing
card foreground remains readable over the strongest sweep color.

The existing animated Beacon wordmark establishes a six-second, Reduce
Motion-aware decorative animation. The larger repeated card watermark should
move more slowly and at a lower refresh cadence so the grid feels coherent
rather than busy.

## REQUIREMENTS

- Render the canonical full project name as one oversized, single-line,
  clipped background watermark in every Fit Following lane card.
- Preserve the fitted card size, grid inputs, status order, lane actions,
  half-height Notes split, and no-scroll all-items guarantee exactly.
- Keep the existing lane content above the watermark and fully interactive.
- Move a narrow color highlight through the watermark on a calm cycle that is
  slower than the Beacon header wordmark and inexpensive across a full grid.
- Define a dedicated watermark palette for every built-in theme. At maximum
  presentation strength, every palette color must remain faint against the
  card surface while existing semantic card foregrounds continue to meet their
  normal-text contrast requirement.
- Increase the watermark's static visibility when Increase Contrast is active,
  desaturate its sweep for Differentiate Without Color, and stop motion at a
  useful centered highlight for Reduce Motion.
- Keep duplicate decorative text out of the accessibility tree and preserve
  the existing card and action accessibility behavior.
- Cover palette completeness and contrast, deterministic animation timing,
  fitted geometry preservation, and rendered theme/accessibility variants.
- Update canonical user and repository memory to describe the fitted project
  identity treatment.

Non-goals are changing other view modes, adding a user preference, grouping
lanes by project, changing project naming, or adding any new data authority.

## ACCEPTED PLAN

1. Extend each `BeaconTheme` with four tuned near-surface watermark colors: a
   stable base plus three highlight hues. Validate their surface contrast and
   every existing surface-text role against the complete watermark palette.
2. Add a small fitted-only SwiftUI watermark view. Scale the canonical project
   name to consume the full card, clip it to the rounded card, and animate a
   narrow gradient sweep across the text on a ten-second cycle.
3. Pass the canonical project name only from `FittedFollowingDashboard` into
   the shared card renderer. Keep every other dense-card caller unchanged and
   preserve the fitted geometry constants and sizing algorithm.
4. Respect Reduce Motion, Differentiate Without Color, and Increase Contrast
   inside the decorative view without changing the underlying card content.
5. Add focused timing, geometry, palette, contrast, and rendering coverage;
   then run the complete repository validation and native multi-theme smoke.
6. Curate README, Constitution, progress summary, and this specification to the
   delivered behavior, then deliver issue #57 through exact branch `GH-57` and
   a ready pull request.

## DECISIONS

- Accepted a fitted-only background watermark rather than project headers or
  taller cards because background identity consumes no grid geometry.
- Accepted theme-owned near-surface colors rather than applying low opacity to
  bright semantic accents. Final background colors can be contrast-tested
  directly and avoid transiently weakening lane copy.
- Accepted one calm synchronized sweep. Independent per-card animation offsets
  would make a populated fitted grid visually noisy and harder to inspect.
- Accepted a static centered highlight under Reduce Motion rather than hiding
  the watermark, preserving the requested project identity without motion.
- Rejected a watermark accessibility label because it would duplicate visual
  decoration and risks obscuring the existing child actions. Existing semantic
  card accessibility remains authoritative.

## DISCOVERIES

The fitted view currently passes only the lane and dense density into the
shared renderer, while `state.projectGroup(for:)` already resolves the
canonical snapshot project name with a repository fallback. This permits a
presentation-only change with no data-model expansion.

## VALIDATION

- Focused `BeaconThemeTests` and `DashboardFittedModeTests` pass, covering all
  five palette contracts, theme rendering, ten-second sweep timing, Reduce
  Motion phase, Increase Contrast strength, and unchanged `220 x 88` geometry.
- The complete repository gate passes: Go formatting, vet, unit and race tests,
  release tests, CLI build, 154 native XCTest cases, and a universal macOS
  debug build.
- Native smoke of the built app verifies all 12 current Following lanes remain
  visible above half-height Notes in Selenized Dark, Lobster Nebula, Pampas
  Moon, Solarized Dark, and Monokai. The watermark remains behind interactive
  content, changes color over time, and adds no duplicate accessibility node.
- Project-file lint, Kit repository checks, cross-platform Go build, whitespace
  review, and delivery-state checks complete before staging.

## OUTCOME

Fit Following now resolves each lane's canonical project and renders that full
name as an oversized clipped background watermark inside the existing fixed
card. Each built-in theme owns a faint four-color palette, and a synchronized
ten-second highlight crosses the letters without changing the grid algorithm,
card size, Notes split, or lane interaction.

Increase Contrast presents the tested palette at full strength, Differentiate
Without Color removes hue dependence, and Reduce Motion keeps a centered
static highlight. The decorative duplicate is hidden from accessibility while
the existing semantic card children remain authoritative. Issue #57 is
represented by ready pull request #58 from exact branch `GH-57`.

## REPOSITORY MEMORY

- Created this specification because the theme palette, accessibility
  fallbacks, animation pace, and rejected geometry-changing alternatives are
  material product rationale that code and tests alone would not preserve.
- Updated `README.md` with the user-facing behavior, `docs/CONSTITUTION.md` with
  the demonstrated fitted-geometry, palette, and accessibility invariants, and
  `docs/PROJECT_PROGRESS_SUMMARY.md` with the feature delivery state.
