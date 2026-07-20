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
  id: "0020"
  slug: ollama-notes-assistant
  dir: 0020-ollama-notes-assistant
relationships:
  - type: builds_on
    target: 0013-signal-note-tabs
  - type: builds_on
    target: 0017-beacon-focus-notes
references:
  - id: issue-45
    name: Add Ollama assistant to Notes
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/45
    relation: implements
    read_policy: must
    used_for: original request, scope, acceptance criteria, and delivery lane
    status: optional
  - id: issue-51
    name: Expand Notes AI into a conversation panel
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/51
    relation: implements
    read_policy: must
    used_for: conversation history, presentation modes, keyboard shortcuts, and delivery lane
    status: optional
  - id: issue-62
    name: Refine Notes assistant button
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/62
    relation: implements
    read_policy: must
    used_for: compact icon-only control, motion, accessibility, and delivery lane
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: Go authority, local-only Notes, configuration, helper, and native macOS boundaries
    status: active
skills:
  - name: figma:figma-swiftui
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/figma/2.0.16/skills/figma-swiftui/SKILL.md
    trigger: native Notes assistant UI from the user-provided screenshots
    required: true
  - name: github:github
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/github/0.1.8-2841cf9749ae/skills/github/SKILL.md
    trigger: GitHub issue and delivery orientation
    required: true
  - name: github:yeet
    source: codex
    path: /Users/jamesonstone/.codex/plugins/cache/openai-curated-remote/github/0.1.8-2841cf9749ae/skills/yeet/SKILL.md
    trigger: user-requested branch, commit, push, and pull request delivery
    required: true
---

# Ollama Notes Assistant

## Thesis

Beacon should let a user have an in-memory conversation about the current local
Signal Note, defaulting to the exact editor selection when one exists and
otherwise the entire visible note, and receive responses from a locally
installed Ollama model without turning Notes into an autonomous agent or sending
note content to a remote model. Context should remain removable, every user and
assistant turn should remain visible in order, the interaction visibly user
initiated, the composer easy to reach, and note content unchanged unless the
user edits it through the existing editor.

## Context

Beacon already owns one native AppKit Markdown editor shared by the menu extra
and detached dashboard, and its Go helper already accepts sensitive note bodies
over standard input. Ollama exposes a loopback HTTP API at
`http://127.0.0.1:11434`: `/api/tags` reports installed models and `/api/chat`
accepts message-based prompts. Beacon can therefore preserve its existing Go
authority by putting Ollama HTTP and model-validation policy in the bundled Go
helper while keeping Swift responsible for selection capture and presentation.

The user-provided Notes screenshots are the visual authority for placement and
scale. The assistant trigger belongs at the top right of the expanded Notes
header, before the size control, and matches that adjacent control's 20-by-20
point footprint. Its panel appears immediately below and right-aligned to that
trigger, overlays the Notes content without expanding past Beacon's current
bounds, and uses existing semantic theme tokens, native controls, and SF
Symbols.

## Clarifications

- This slice is an active-session, multi-turn assistant. It does not add
  background analysis, embeddings, retrieval, persistent chat history,
  automatic note organization, or note mutation.
- The assistant action is always available while Notes is expanded. It captures
  the exact non-empty editor selection when one exists and otherwise the entire
  current draft, including unsaved text visible in the editor.
- The attached context is a snapshot captured when the panel opens. Changing
  the editor afterward does not silently change it. The user may remove the
  attachment and continue with a prompt that has no Notes context.
- The user supplies an additional prompt. Send is disabled until the prompt and
  selected model are non-empty and no request is running; context is optional.
- A dedicated labeled Cancel action exits and resets the assistant immediately.
  The Notes and all-commands quick switchers expose the same entry action.
- Command-I opens the larger conversation presentation with a right-edge slide
  transition. Command-Shift-I opens the existing compact Notes presentation.
  Switching presentations preserves the active conversation; Cancel resets it.
- The conversation renders every completed user and assistant turn in order in
  one independently scrolling history. The unsent prompt composer and send/model
  controls remain pinned to the bottom in both presentations.
- Each follow-up request carries the complete active conversation to Ollama with
  native user and assistant roles. Beacon never truncates displayed history
  silently; oversized input remains visible and fails as a recoverable inline
  error before generation.
- Ollama remains fixed to its loopback endpoint. Beacon does not expose a remote
  host setting, authentication field, or arbitrary URL in this feature.
- Only models reported as real local artifacts are offered. Cloud entries such
  as names ending in `:cloud`, zero-detail entries, and zero-sized entries are
  excluded from both selection and chat validation.
- `settings.ollama_model` is an optional version-2 YAML field. When it is empty
  or unavailable, Beacon selects the first installed local model in stable
  name order. Choosing a model in Settings atomically rewrites the same field.
- Selecting a different model in the Notes panel affects that request but does
  not change the configured default; Settings is the explicit persistence
  boundary.
- Ollama unavailability, no installed local models, a missing configured model,
  malformed API output, and chat failures are recoverable inline errors. They
  do not affect Notes saving, the background agent, or canonical work evidence.

## Requirements

1. Keep one user-initiated, active-session Ollama conversation inside the current
   Notes and Beacon bounds without background analysis or automatic note edits.
2. Make the assistant entry action always available as a 20-by-20 point,
   icon-only control that matches the adjacent Notes size control and remains
   reachable from the Notes and all-commands quick switchers.
3. Snapshot exact selected text when present and otherwise the entire current
   draft, including unsaved visible edits.
4. Present captured context as an optional removable attachment and allow a
   prompt to be sent after the attachment is removed or when the note is empty.
5. Provide a dedicated labeled Cancel action that resets the interaction and
   prevents a late response from repopulating the dismissed panel.
6. Preserve local-only model discovery, explicit model choice, bounded stdin
   transport, and loopback non-streaming chat through the Go helper.
7. Preserve existing Notes editing, autosave, tabs, sizing, shared state,
   theming, accessibility, and background-agent behavior.
8. Keep every user and assistant turn in one ordered, in-memory conversation and
   render the complete active history rather than replacing the prior answer.
9. Send the complete active history to Ollama with explicit user and assistant
   roles while preserving bounded stdin transport and strict role/content
   validation.
10. Pin the unsent prompt composer and model/send controls to the bottom while
    conversation history scrolls independently and follows newly completed turns.
11. Open a materially larger, in-bounds conversation panel from the right with
    Command-I while Command-Shift-I opens the existing compact panel.
12. Use Beacon's established `brain.head.profile` Ollama symbol with a subtle,
    whimsical animated sparkle, current semantic theme tokens, increased-
    contrast support, and a static Reduce Motion state.

## Assumptions

- The current `notesDraft` is the authoritative text visible to the user and
  intentionally includes edits that have not reached autosave yet.
- A non-empty selection takes precedence over whole-note context because it is
  the user's more specific signal; no selection falls back to the full draft.
- Removing context affects only the pending assistant request and never changes
  the underlying note or configured default model.
- The existing menu-extra and detached-dashboard `MenuView` surfaces share the
  same `AppState`, so one command and panel lifecycle serves both surfaces.

## Acceptance Criteria

- [x] AC1: Expanded Notes presents an always-enabled, accessible, icon-only
  assistant button in the same 20-by-20 point footprint as the adjacent Notes
  size control on both Beacon surfaces.
- [x] AC2: Pressing the button captures the exact selected text when non-empty,
  otherwise the entire current draft, and opens one right-aligned assistant
  panel directly below the button within the existing Notes/Beacon bounds.
- [x] AC3: The panel presents optional captured context as a removable
  attachment, an editable prompt, a discovered local-model selector, a
  right-aligned send button, and a dedicated labeled Cancel action.
- [x] AC4: The Go helper queries loopback Ollama `/api/tags`, returns only local
  model artifacts in stable name order, and rejects a chat request for any model
  that is not in that set.
- [x] AC5: The model selected by `settings.ollama_model` is used when installed;
  otherwise the first discovered local model is selected without corrupting the
  configured value.
- [x] AC6: Settings lists the same discovered local models and persists an
  explicit default atomically to `settings.ollama_model` in the resolved
  `config.yaml` without removing existing configuration.
- [x] AC7: Sending writes a bounded JSON request containing optional context and
  the prompt to the bundled helper over stdin, invokes Ollama `/api/chat` with
  streaming disabled, and renders the returned assistant content as a chat
  response inside the same panel.
- [x] AC8: While a request is running, duplicate sends are disabled and visible
  progress is shown; failures and empty-model states remain inline and retryable.
- [x] AC9: Attached note content is never placed in process arguments, logs,
  persistent chat state, Beacon evidence, or a remote Ollama endpoint, and a
  response never changes the note automatically.
- [x] AC10: The Notes and all-commands quick switchers include an Ask AI command
  that restores expanded Notes when necessary and opens the same assistant with
  the same context-resolution behavior.
- [x] AC11: Existing Notes editing, autosave, tabs, sizing, menu/dashboard shared
  state, theming, accessibility preferences, and background-agent behavior
  continue to pass their existing tests.
- [x] AC12: The follow-up is documented and covered by Go optional-context tests
  plus Swift context resolution, removal, cancellation, presentation, and
  quick-switcher command tests.
- [x] AC13: The assistant header control uses the established
  `brain.head.profile` symbol, semantic theme colors, no visible text, a subtle
  animated sparkle, and the existing accessible action name; Reduce Motion
  presents the same mark without continuous movement.
- [x] AC14: Command-I opens a conversation panel larger than the compact panel,
  aligned to the Beacon surface's right edge, and inserts/removes it with a
  right-edge transition unless Reduce Motion is enabled.
- [x] AC15: Command-Shift-I opens the existing compact Notes-owned panel, and
  changing between compact and conversation presentations does not reset the
  active context, messages, model, or unsent prompt.
- [x] AC16: Sending appends the trimmed user turn, clears the pinned composer,
  passes every prior turn to the helper in order, and appends the returned model
  answer without removing earlier messages.
- [x] AC17: Both presentations render the entire ordered history in an
  independently scrolling region while the attachment, composer, model selector,
  and send action remain reachable; a failed send restores its prompt as unsent.
- [x] AC18: Cancel clears context, ordered history, prompt, errors, progress, and
  stale-response eligibility without mutating the current Note.

## Design

### Go authority

`internal/ollama` owns the fixed loopback endpoint, bounded HTTP decoding,
local-model filtering, exact availability validation, and role-ordered chat
requests. `internal/cli` exposes JSON-only helper operations for model status,
chat, and the configured default. Chat input is JSON read from stdin so neither
the selected note text nor conversation messages appear in the process list.

The helper sends a short system instruction followed by the complete ordered
active conversation using native user and assistant roles. Optional Notes
context is clearly delimited as untrusted data on the first user turn. Legacy
single-prompt stdin remains accepted for compatibility, while new conversation
input is role-validated, UTF-8 validated, and total-size bounded before Ollama
is called. It disables streaming and returns only the normalized model and
assistant content needed by the thin client. Network calls use explicit
contexts and response-size limits.

### Configuration

The strict version-2 settings schema gains optional `ollama_model`. The value is
trimmed, must not contain control characters, and is preserved by load, merge,
and atomic rewrite. Version 1 rejects the version-2-only field. The helper's
set-default command loads the complete resolved config, changes only this
setting in memory, and writes the complete canonical version-2 document through
the existing atomic writer.

### Native presentation

`LiveMarkdownEditor` binds its exact current selection in addition to its
current line. `MenuView` resolves that selection or the entire current draft
into shared assistant state when the larger header action or quick-switcher
command opens. A Notes-owned overlay places the compact panel under the trigger
without creating another window. The removable attachment and assistant answer
are visually separate, the prompt remains editable without context, and native
menus, buttons, progress, focus, and semantic theme colors preserve macOS
behavior. A labeled Cancel action dismisses and resets the interaction.

`AppState` holds one shared assistant request state and ordered in-memory message
history for the menu extra and detached dashboard. Its injected Ollama helper
boundary loads status, resolves the effective model, persists an explicit
default, serializes complete conversation requests, and normalizes recoverable
errors. The compact Notes overlay and larger right-edge overlay reuse one panel
component whose history expands while its composer stays at the bottom.
Dismissing the panel does not cancel Notes autosave or mutate the draft.

## Implementation Plan

1. Extend the existing helper chat contract so bounded Notes context is
   optional while model and prompt validation remain strict.
2. Resolve selection-first or whole-draft context in shared Swift state and add
   attachment removal, reset, and stale-response protection.
3. Replace the selection-gated header icon with a larger semantic AI control,
   labeled Cancel, and an optional attachment presentation.
4. Add one shared Ask AI command to the Notes and all-command switcher scopes,
   restoring expanded Notes before presentation when necessary.
5. Add focused Go and Swift coverage, update canonical documentation, run the
   complete repository gates, and exercise the built app against local Ollama.
6. Deliver the follow-up on the existing issue #45, branch `GH-45`, and ready
   PR #46 lane, then verify the final hosted head.
7. Replace the single response slot with an ordered in-memory user/assistant
   history and send that complete history through the bounded helper protocol.
8. Refactor the panel into flexible scrollable history plus a bottom-pinned
   composer shared by compact and conversation presentations.
9. Add Command-I for the large right-edge transition, Command-Shift-I for the
   compact presentation, and a Beacon-themed AI mark without changing the
   existing button label or accessible intent.
10. Extend focused Go/Swift coverage, refresh product documentation, run the
    complete local gates, and smoke both presentation modes.
11. Deliver the extension on issue #51, branch `GH-51`, and a ready pull request,
    then verify the exact final hosted head.
12. Replace the labeled prominent header control with an icon-only 20-by-20
    point assistant mark that matches the adjacent Notes control, remains
    theme- and contrast-aware, and becomes static under Reduce Motion.
13. Add focused symbol, sizing, and motion coverage; update affected product
    documentation; run the complete native and repository gates; and smoke the
    built control at its actual header size.
14. Deliver the refinement on issue #62, exact branch `GH-62`, and a ready pull
    request, then verify the exact final hosted head.

Rollback restores required context in the helper, removes the follow-up state
and switcher command, and returns the header to selection-gated presentation.
No persisted data or configuration migration is involved.

## Task Checklist

- [x] T1: Add the strict `settings.ollama_model` configuration field, atomic
  update path, documentation, and tests.
- [x] T2: Add the bounded local-only Ollama HTTP client and model/chat tests.
- [x] T3: Add JSON helper commands for model status, default persistence, and
  stdin chat plus CLI tests and no-agent-autostart routing.
- [x] T4: Add Swift helper models/protocol methods and shared AppState lifecycle
  for discovery, default resolution, persistence, send, progress, and errors.
- [x] T5: Bind exact AppKit editor selection and add the header action plus
  in-bounds native assistant panel to both shared Notes surfaces.
- [x] T6: Add focused Swift tests and update the Xcode project.
- [x] T7: Run formatting, Go unit/race/lint/build checks, macOS tests/build,
  targeted live Ollama smoke, and review the complete diff.
- [x] T8: Complete issue #45 delivery on `GH-45` with an explicit commit, push,
  ready pull request, and hosted-check verification.
- [x] T9: Add whole-note fallback, optional context, explicit removal/reset, and
  stale-response protection to the shared Go and Swift assistant state.
- [x] T10: Replace the small selection-gated icon with a larger always-enabled
  AI button, labeled Cancel, and removable attachment presentation.
- [x] T11: Add the shared Ask AI command to both quick-switcher scopes and cover
  context resolution, removal, cancellation, command discovery, and context-free chat.
- [x] T12: Refresh docs, rerun all local/live gates, update ready PR #46, and
  verify the final hosted head.
- [x] T13: Extend the helper chat protocol with bounded, role-validated ordered
  messages while preserving legacy prompt compatibility and local-only policy.
- [x] T14: Replace the Swift single-response state with ordered active-session
  messages, complete-history sends, failed-prompt restoration, and reset safety.
- [x] T15: Render all turns in a flexible scrolling history with the prompt,
  model picker, and send action pinned to the bottom in both panel sizes.
- [x] T16: Add the large right-edge conversation presentation, Command-I and
  Command-Shift-I routing, mode-preserving transitions, and Beacon-themed mark.
- [x] T17: Add focused helper, state, sizing, mode, failure, history, and reset
  tests; update README and Constitution behavior contracts.
- [x] T18: Run full local and native smoke validation, complete GH-51 delivery,
  and verify the exact ready-PR hosted-check state.
- [x] T19: Replace the visible `AI` label and prominent button chrome with a
  20-by-20 point icon-only assistant control matching the Notes size control.
- [x] T20: Add a theme-aware animated assistant mark with increased-contrast
  treatment, a Reduce Motion static state, and focused presentation coverage.
- [x] T21: Update affected documentation and complete local/native validation.
- [x] T22: Deliver GH-62 and verify the exact final hosted head.

## Validation Map

| Criterion | Verification |
| --- | --- |
| AC1-AC3 | Swift context-resolution and 20-by-20 icon-only presentation tests plus live whole-note, removable-attachment, and Cancel smoke |
| AC4-AC7 | Focused Go client/CLI tests, bounded helper tests, and live local-model request without context |
| AC8-AC9 | AppState progress/error tests, request-generation cancellation test, transport review, and unchanged-note smoke |
| AC10 | Swift command discovery/action test plus live Command-K and Command-P invocation while Notes is expanded and minimized |
| AC11 | Complete Go race/vet/build/release gates, 135-test native suite, universal macOS build, and live Save/Revert state inspection |
| AC12 | README, Constitution, project-summary, and feature-spec review plus focused 10-test Ollama XCTest suite |
| AC13-AC15 | Swift symbol, sizing, motion-phase, shortcut, and Reduce Motion wiring review plus native icon/compact/large transition smoke |
| AC16-AC18 | Go ordered-message/limit tests, Swift AppState history/failure/reset tests, full native suite, and unchanged-note smoke |

## Reflection Notes

Selection-first resolution preserves the precise original workflow while an
empty selection now makes the whole current draft the useful default. Keeping
that resolution in shared state means the header and both switcher scopes behave
identically. Context remains a snapshot instead of a live binding, so removing
it or continuing without it cannot mutate the draft. A request generation token
lets Cancel reset immediately and discards any response that returns afterward
without coupling Notes lifecycle to network cancellation.

The native smoke showed that the larger semantic control is quick to target,
the panel remains right-aligned and in bounds, both quick switchers find the
same command, minimized Notes restores before opening, and exact selection still
wins over whole-note fallback. A real local model answered after context removal
while Save and Revert stayed disabled, confirming no note mutation.

The conversation extension keeps presentation and session ownership in shared
`AppState`, so the menu extra and detached dashboard cannot reset one another.
The complete role-ordered transcript is sent on every turn, remains visible in
both panel sizes, and survives Command-I/Command-Shift-I mode changes together
with any unsent draft. The composer remains outside the history scroll at the
bottom, while Cancel continues to be the explicit reset boundary.

The extension smoke exercised both shortcuts while native text editors were
focused, removed real Notes context before generation, completed a two-turn
exchange with `gemma3:270m`, and verified that the second answer depended on the
first turn. All four messages and an additional unsent draft remained visible
with the composer pinned at the bottom; Cancel cleared the session while Notes
Save and Revert remained disabled.

The earlier larger labeled button optimized first-use discoverability, but the
follow-up screenshot shows that its text and prominent chrome dominate the
otherwise compact Notes header. The accepted refinement keeps discoverability
in the tooltip, accessibility label, and quick switchers while using the same
small footprint as the adjacent Notes control. A familiar assistant symbol and
slow orbiting sparkle carry the visual meaning without permanent label text;
Reduce Motion freezes that composition instead of removing its identity.

## Documentation Updates

- [x] README usage now covers always-available AI, selection/full-note context,
  removal, Cancel, both quick switchers, and the local-only boundary.
- [x] Constitution records shared entry points, optional context, reset, and
  late-response safety as durable product invariants.
- [x] README covers both keyboard presentations, complete active-session
  history, the pinned composer, and ephemeral local conversation behavior.
- [x] Constitution records ordered role transport, shared presentation state,
  and the pinned-composer conversation layout.
- [x] Project progress summary tracks the reopened implementation phase and
  follow-up scope on the existing delivery lane.
- [x] This specification maps the follow-up requirements, implementation,
  validation, reflection, delivery decision, and evidence.
- [x] Product documentation describes the compact icon-only assistant control
  without referring to a larger visible `AI` label.

## Delivery Decision

Deliver the conversation extension on issue #51 and exact branch `GH-51` because
it changes the assistant from one-turn response replacement to an active
multi-turn conversation and adds a distinct presentation mode. Use explicit
staging, verified Jameson Stone author and committer identity, the repository PR
template, a ready pull request, and literal final-head hosted-check reporting.

Deliver the compact control refinement separately on issue #62 and exact branch
`GH-62` because the previous assistant issues and pull requests are closed and
the active GH-59 lane has unrelated settings/documentation scope. Keep the
existing assistant behavior unchanged and use the same explicit staging,
human-identity, ready-PR, and final-head verification gates.

## Evidence

- GitHub issue: [#45](https://github.com/jamesonstone/beacon/issues/45)
- Branch: `GH-45`
- Local validation: complete; Go, race, vet, build, release, macOS test/build,
  live Ollama, and native Notes assistant smoke checks pass
- Hosted validation: Go and macOS checks passed on implementation commit
  `db6abf2`; the ready pull request remains the human review boundary
- Pull request: [#46](https://github.com/jamesonstone/beacon/pull/46)
- Follow-up local validation: focused Go packages, full Go format/test/race/vet/
  build/release targets, 10 focused Ollama XCTest cases, the complete 135-test
  native suite, universal macOS build, `kit check --all`, project-file lint, and
  whitespace checks pass.
- Follow-up live validation: the built app attached the full current draft with
  no selection, removed it and sent a context-free prompt to `gemma3:270m`,
  reset through Cancel, opened through Command-K and Command-P, restored
  minimized Notes, and captured the exact `collection_date` selection without
  enabling Save or Revert.
- Follow-up hosted validation: required Go and macOS checks passed on
  implementation commit `80bd1f3`; PR #46 remains ready for human review.
- Conversation extension issue: [#51](https://github.com/jamesonstone/beacon/issues/51)
- Conversation extension branch: `GH-51`
- Conversation extension local validation: focused Go client/CLI tests, full Go
  format/test/race/vet/build/release targets, 14 focused Ollama XCTest cases,
  the complete 140-test native suite, universal macOS build, `kit check --all`,
  project-file lint, whitespace checks, and independent diff review pass.
- Conversation extension native validation: the built app opened the large
  right-edge panel with Command-I and the compact panel with Command-Shift-I
  while text inputs were focused; an unsent prompt survived mode switching; a
  context-free two-turn `gemma3:270m` exchange retained all four ordered turns;
  the bottom composer kept an additional unsent draft visible; and Cancel reset
  the session without enabling Notes Save or Revert.
- Conversation extension hosted validation: required Go and macOS checks passed
  on implementation commit `24eba88`; CodeRabbit remained optional and in
  progress when delivery evidence was recorded.
- Conversation extension pull request: [#52](https://github.com/jamesonstone/beacon/pull/52),
  ready for human review
- Compact control refinement issue: [#62](https://github.com/jamesonstone/beacon/issues/62)
- Compact control refinement branch: `GH-62`
- Compact control refinement local validation: the complete 154-test native
  suite, universal macOS build, full Go format/test/race/vet/build/release
  targets, all 21 Kit feature checks, the Kit project contract, whitespace
  checks, and focused valid-symbol/size/orbit tests pass.
- Compact control refinement native validation: the rebuilt app renders the
  icon-only assistant and adjacent Notes control in matching compact footprints;
  the accessibility tree exposes `Ask AI About Current Note`; clicking the mark
  opens the existing assistant with the current note and clicking it again
  dismisses the panel without changing Notes.
- Compact control refinement pull request: [#63](https://github.com/jamesonstone/beacon/pull/63),
  ready for human review and assigned to Jameson Stone.
- Compact control refinement hosted validation: the initial Xcode 15.4 macOS run
  exposed an ambiguous trigonometric overload that the newer local toolchain
  accepted. Commit `6276124` selects the explicit CoreGraphics `CGFloat`
  overloads; required Go and macOS checks then passed on that exact head, and
  CodeRabbit completed with no actionable review threads.
