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

Beacon should let a user ask a one-turn question about the current local Signal
Note, defaulting to the exact editor selection when one exists and otherwise the
entire visible note, and receive a response from a locally installed Ollama
model without turning Notes into an autonomous agent or sending note content to
a remote model. Context should remain removable, the interaction visibly user
initiated, the panel easy to enter and exit, and note content unchanged unless
the user edits it through the existing editor.

## Context

Beacon already owns one native AppKit Markdown editor shared by the menu extra
and detached dashboard, and its Go helper already accepts sensitive note bodies
over standard input. Ollama exposes a loopback HTTP API at
`http://127.0.0.1:11434`: `/api/tags` reports installed models and `/api/chat`
accepts message-based prompts. Beacon can therefore preserve its existing Go
authority by putting Ollama HTTP and model-validation policy in the bundled Go
helper while keeping Swift responsible for selection capture and presentation.

The user-provided Notes screenshot is the visual authority for placement. The
assistant trigger belongs at the top right of the expanded Notes header, before
the size control. Its panel appears immediately below and right-aligned to that
trigger, overlays the Notes content without expanding past Beacon's current
bounds, and uses existing semantic theme tokens, native controls, and SF
Symbols.

## Clarifications

- This slice is a single-turn assistant. It does not add background analysis,
  embeddings, retrieval, persistent chat history, automatic note organization,
  or note mutation.
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

1. Keep one user-initiated, single-turn Ollama assistant inside the current
   Notes and Beacon bounds without background analysis or automatic note edits.
2. Make the AI entry action always available, materially larger than the prior
   icon, and reachable from the Notes and all-commands quick switchers.
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

- [x] AC1: Expanded Notes presents an always-enabled, accessible, labeled AI
  button with a materially larger target on both Beacon surfaces.
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

## Design

### Go authority

`internal/ollama` owns the fixed loopback endpoint, bounded HTTP decoding,
local-model filtering, exact availability validation, and the one-turn chat
request. `internal/cli` exposes JSON-only helper operations for model status,
chat, and the configured default. Chat input is JSON read from stdin so neither
the selected note text nor the prompt appears in the process list.

The helper sends a short system instruction followed by one user message that
labels optional Notes context and the user's request. It disables streaming
and returns only the normalized model and assistant content needed by the thin
client. Network calls use explicit contexts and response-size limits.

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

`AppState` holds one shared assistant request state for the menu extra and
detached dashboard. Its injected Ollama helper boundary loads status, resolves
the effective model, persists an explicit default, serializes one request, and
normalizes recoverable errors. Dismissing the panel does not cancel Notes
autosave or mutate the draft.

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

## Validation Map

| Criterion | Verification |
| --- | --- |
| AC1-AC3 | Swift context-resolution tests plus live larger-button, whole-note, removable-attachment, and Cancel smoke |
| AC4-AC7 | Focused Go client/CLI tests, bounded helper tests, and live local-model request without context |
| AC8-AC9 | AppState progress/error tests, request-generation cancellation test, transport review, and unchanged-note smoke |
| AC10 | Swift command discovery/action test plus live Command-K and Command-P invocation while Notes is expanded and minimized |
| AC11 | Complete Go race/vet/build/release gates, 135-test native suite, universal macOS build, and live Save/Revert state inspection |
| AC12 | README, Constitution, project-summary, and feature-spec review plus focused 10-test Ollama XCTest suite |

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

## Documentation Updates

- [x] README usage now covers always-available AI, selection/full-note context,
  removal, Cancel, both quick switchers, and the local-only boundary.
- [x] Constitution records shared entry points, optional context, reset, and
  late-response safety as durable product invariants.
- [x] Project progress summary tracks the reopened implementation phase and
  follow-up scope on the existing delivery lane.
- [x] This specification maps the follow-up requirements, implementation,
  validation, reflection, delivery decision, and evidence.

## Delivery Decision

Continue the existing ready-for-review lane on issue #45, exact branch `GH-45`,
and PR #46 because this is a narrow follow-up to the same Ollama Notes assistant.
Use explicit staging, verified Jameson Stone author and committer identity, the
repository PR template, and literal final-head hosted-check reporting.

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
