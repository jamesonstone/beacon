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
    trigger: native selected-text Notes assistant UI from the user-provided screenshot
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

Beacon should let a user select a bounded fragment of a local Signal Note,
attach that exact selection to a one-turn prompt, and receive a response from a
locally installed Ollama model without turning Notes into an autonomous agent or
sending note content to a remote model. The interaction should remain visibly
user initiated, fit inside the current Beacon surface, and leave note content
unchanged unless the user edits it through the existing editor.

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
- The attached context is the exact non-empty editor selection captured when
  the assistant button is pressed. Changing the editor selection afterward does
  not silently change the attachment already shown in the panel.
- The user supplies an additional prompt. Send is disabled until the attachment,
  prompt, and selected model are all non-empty and no request is running.
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

## Acceptance Criteria

- [x] AC1: Selecting a non-empty subset in the native Notes editor enables an
  accessible top-right assistant button on both Beacon surfaces; no selection
  leaves the action disabled.
- [x] AC2: Pressing the button captures the exact selected text and opens one
  right-aligned assistant panel directly below the button, constrained within
  the existing Notes/Beacon bounds.
- [x] AC3: The panel presents the captured text as a distinct attached-context
  bubble, an editable prompt, a discovered local-model selector, and a
  right-aligned send button.
- [x] AC4: The Go helper queries loopback Ollama `/api/tags`, returns only local
  model artifacts in stable name order, and rejects a chat request for any model
  that is not in that set.
- [x] AC5: The model selected by `settings.ollama_model` is used when installed;
  otherwise the first discovered local model is selected without corrupting the
  configured value.
- [x] AC6: Settings lists the same discovered local models and persists an
  explicit default atomically to `settings.ollama_model` in the resolved
  `config.yaml` without removing existing configuration.
- [x] AC7: Sending writes a bounded JSON request containing the selection and
  prompt to the bundled helper over stdin, invokes Ollama `/api/chat` with
  streaming disabled, and renders the returned assistant content as a chat
  response inside the same panel.
- [x] AC8: While a request is running, duplicate sends are disabled and visible
  progress is shown; failures and empty-model states remain inline and retryable.
- [x] AC9: Selected note content is never placed in process arguments, logs,
  persistent chat state, Beacon evidence, or a remote Ollama endpoint, and a
  response never changes the note automatically.
- [x] AC10: Existing Notes editing, autosave, tabs, sizing, menu/dashboard shared
  state, theming, accessibility preferences, and background-agent behavior
  continue to pass their existing tests.
- [x] AC11: The implementation is documented in the README and constitution and
  is covered by Go configuration/client/CLI tests plus Swift selection,
  presentation, and state tests.

## Design

### Go authority

`internal/ollama` owns the fixed loopback endpoint, bounded HTTP decoding,
local-model filtering, exact availability validation, and the one-turn chat
request. `internal/cli` exposes JSON-only helper operations for model status,
chat, and the configured default. Chat input is JSON read from stdin so neither
the selected note text nor the prompt appears in the process list.

The helper sends a short system instruction followed by one user message that
labels the selected Notes context and the user's request. It disables streaming
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
current line. `MenuView` copies that selection into shared assistant state when
the header action opens. A Notes-owned overlay places the compact panel under
the trigger without creating another window. The attachment and assistant
answer are visually separate, the prompt remains editable, and native menus,
buttons, progress, focus, and semantic theme colors preserve macOS behavior.

`AppState` holds one shared assistant request state for the menu extra and
detached dashboard. Its injected Ollama helper boundary loads status, resolves
the effective model, persists an explicit default, serializes one request, and
normalizes recoverable errors. Dismissing the panel does not cancel Notes
autosave or mutate the draft.

## Tasks

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

## Validation Plan

- `gofmt` over changed Go files and `git diff --check`.
- `go test ./...`, `go test -race ./...`, `go vet ./...`, and the repository's
  standard lint/build targets.
- Focused `internal/ollama`, `internal/config`, and `internal/cli` tests with an
  `httptest` server or injected fake; no test contacts the user's real Ollama
  service or configuration file.
- Xcode test and build for the Beacon scheme, including selection extraction,
  model resolution, AppState success/failure, and bounded panel presentation.
- Live helper smoke against local Ollama: list installed local models, verify a
  cloud tag is absent, and send a short prompt using a lightweight installed
  local model without changing the user's configured default.
- Native macOS smoke: select Notes text, open the panel, verify the attachment,
  select a model, send, observe the inline response, dismiss, and confirm Notes
  remains unchanged and autosave continues.

## Delivery

- GitHub issue: [#45](https://github.com/jamesonstone/beacon/issues/45)
- Branch: `GH-45`
- Local validation: complete; Go, race, vet, build, release, macOS test/build,
  live Ollama, and native Notes assistant smoke checks pass
- Hosted validation: Go and macOS checks passed on implementation commit
  `db6abf2`; the ready pull request remains the human review boundary
- Pull request: [#46](https://github.com/jamesonstone/beacon/pull/46)
