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
    target: 0017-beacon-focus-notes
  - type: builds_on
    target: 0018-following-workspace
  - type: builds_on
    target: 0020-ollama-notes-assistant
references:
  - id: issue-74
    name: Stop hidden macOS animations and delay hover details
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/74
    relation: implements
    read_policy: must
    used_for: idle-rendering root cause, static decoration, three-second rich-hover threshold, and delivery lane
    status: active
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
  - id: focus-notes
    name: Beacon focus and Notes
    type: spec
    target: docs/specs/0017-beacon-focus-notes/SPEC.md
    relation: informs
    read_policy: must
    used_for: rocket, Notes mark, and empty-state decorative presentation
    status: active
  - id: ollama-notes-assistant
    name: Ollama Notes assistant
    type: spec
    target: docs/specs/0020-ollama-notes-assistant/SPEC.md
    relation: informs
    read_policy: must
    used_for: compact assistant mark and interaction accessibility
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
# Animated Project Watermarks

## PURPOSE

Fit Following should make the repository behind each lane immediately legible
without surrendering its defining promise that every current Following lane
remains visible above interactive Notes. Each fitted card therefore uses its
otherwise quiet background as a playful identity surface: the complete project
name appears oversized, faint, static, and theme-aware behind the factual lane
content.

Decorative identity must not create perpetual rendering work. The menu-extra
and retained dashboard surfaces may remain mounted while no Beacon window is
visible, so all decorative marks must render from stable values rather than
continuous SwiftUI timeline schedules. Rich detail must likewise require a
deliberate three-second pointer hover while remaining immediate through focus,
click, and other explicit access paths.

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

Live profiling of the current debug build with zero visible Beacon windows
showed the SwiftUI process sustaining roughly 65 percent CPU while the Go agent
was idle. The sampled main-thread render path attributed 727 of 741 relevant
`TimelineView` update samples to `ProjectWatermark`; the retained header, Notes,
assistant, and empty-state timelines added further hidden work. Debug
optimization amplifies the cost but does not cause the hidden render loop.

## REQUIREMENTS

- Render the canonical full project name as one oversized, single-line,
  clipped background watermark in every Fit Following lane card.
- Preserve the fitted card size, grid inputs, status order, lane actions,
  half-height Notes split, and no-scroll all-items guarantee exactly.
- Keep the existing lane content above the watermark and fully interactive.
- Render one centered, narrow color highlight through the watermark without a
  timeline, timer, display link, or time-dependent view invalidation.
- Define a dedicated watermark palette for every built-in theme. At maximum
  presentation strength, every palette color must remain faint against the
  card surface while existing semantic card foregrounds continue to meet their
  normal-text contrast requirement.
- Increase the watermark's static visibility when Increase Contrast is active
  and desaturate its highlight for Differentiate Without Color.
- Remove every continuous decorative `TimelineView` from the shared dashboard
  surfaces while preserving the marks, semantic theme treatment, geometry,
  labels, and accessibility behavior as stable compositions.
- Require three continuous seconds of pointer hover before lane-detail and
  taxonomy popovers open. Preserve immediate keyboard focus, click pinning,
  pointer traversal into an open popover, Escape/outside dismissal, and zero
  network work on hover.
- Keep duplicate decorative text out of the accessibility tree and preserve
  the existing card and action accessibility behavior.
- Cover palette completeness and contrast, static highlight presentation,
  the exact rich-hover threshold, fitted geometry preservation, and rendered
  theme/accessibility variants.
- Update canonical user and repository memory to describe the fitted project
  identity treatment.

Non-goals are removing bounded event-driven transitions, changing view modes,
adding a user preference, grouping lanes by project, changing project naming,
or adding any new data authority.

## ACCEPTED PLAN

The initial issue #57 delivery added the theme palettes, fitted-only watermark,
canonical project-name routing, accessibility adaptations, and focused theme
coverage. Issue #74 supersedes only its motion decision and coordinates the
shared idle-rendering repair:

1. Replace the time-derived watermark sweep with the existing centered static
   highlight while preserving the complete theme palette and card geometry.
2. Replace the rocket, Notes orbit, assistant sparkle, header wordmark, and
   decorative empty-state timelines with stable compositions. Keep ordinary
   bounded interaction transitions unchanged.
3. Change the one shared `RichHoverPresentation.openDelay` contract from 350
   milliseconds to exactly three seconds so lane details and taxonomy guidance
   stay consistent.
4. Update focused presentation tests to reject continuous timeline behavior and
   assert the exact hover duration; run the full macOS and repository gates.
5. Build and launch a fresh app, close its dashboard while leaving the process
   active, and verify the idle Swift process no longer consumes sustained CPU.
6. Curate every affected feature contract and user-facing document, then
   deliver issue #74 from exact branch `GH-74` in a ready pull request.

## DECISIONS

- Accepted a fitted-only background watermark rather than project headers or
  taller cards because background identity consumes no grid geometry.
- Accepted theme-owned near-surface colors rather than applying low opacity to
  bright semantic accents. Final background colors can be contrast-tested
  directly and avoid transiently weakening lane copy.
- Superseded the initial calm synchronized sweep after live profiling proved
  that per-card `TimelineView` invalidation continued on retained hidden
  surfaces and dominated the main thread. The centered Reduce Motion rendering
  becomes the normal presentation for every user.
- Accepted static compositions for all decorative marks because the animation
  conveys no workflow state and cannot justify continuous idle CPU cost.
- Accepted one shared three-second hover threshold rather than separate lane and
  taxonomy values so pointer intent remains predictable across the dashboard.
- Rejected a watermark accessibility label because it would duplicate visual
  decoration and risks obscuring the existing child actions. Existing semantic
  card accessibility remains authoritative.

## DISCOVERIES

The fitted view passes only the lane and dense density into the shared renderer,
while `state.projectGroup(for:)` resolves the canonical snapshot project name
with a repository fallback. The performance repair remains presentation-only
with no data-model expansion. `ProjectWatermark`, the shared header and Notes
marks, the compact assistant mark, and both decorative empty states account for
every source `TimelineView`; removing those schedules provides a directly
auditable no-continuous-rendering invariant.

## VALIDATION

- Source audit finds no `TimelineView` in the macOS application and no retained
  timing helpers for the removed decorative motion.
- All 157 native XCTest cases pass, including the centered static watermark,
  unchanged `220 x 88` fitted geometry, and exact three-second shared hover
  contract. Go formatting, vet, and unit tests also pass.
- A fresh Debug app launched with its dashboard closed, matching the diagnosed
  zero-visible-window condition. After the ordinary startup refresh completed,
  Beacon reported 0.0 percent CPU for 19 consecutive one-second samples rather
  than the prior sustained 65 to 66 percent; later work was brief and
  event-driven rather than continuous.
- The initial issue #57 palette, contrast, five-theme rendering, geometry, and
  hosted-check evidence remains valid; issue #74 changes only motion and the
  shared hover threshold.

## OUTCOME

Fit Following now resolves each lane's canonical project and renders that full
name as an oversized clipped background watermark inside the existing fixed
card. Each built-in theme owns a faint four-color palette with one centered
static highlight that does not change the grid algorithm, card size, Notes
split, or lane interaction.

Increase Contrast presents the tested palette at full strength, Differentiate
Without Color removes hue dependence, and every user receives the stable state
that was previously limited to Reduce Motion. The header, Notes, assistant, and
empty-state artwork is likewise static, while bounded event-driven interaction
transitions remain available. Rich detail and taxonomy guidance require three
continuous seconds of pointer hover but remain immediate through keyboard focus
and click. The decorative duplicate is hidden from accessibility while the
existing semantic card children remain authoritative.

## REPOSITORY MEMORY

- Updated this specification because live profiling superseded the original
  animation decision while the theme palette and rejected geometry-changing
  alternatives remain material rationale that code and tests cannot preserve.
- Updated the related Notes, assistant, Following, Constitution, progress, and
  user-guide contracts so current static presentation and three-second hover
  behavior are canonical without erasing the initial issue #57 history.
