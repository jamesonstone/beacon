# Commands

This is the concise command map for Beacon users and contributors. The
[user guide](USER_GUIDE.md) owns detailed product behavior and examples; the
[agent workflow](agents/README.md) and [references](references/README.md) own
implementation and delivery policy.

## Product CLIs

### Hyper-Light Scanner

`bctl` owns the version-2 focused work scan and never starts or talks to the
Beacon background agent.

| Command | Purpose |
| --- | --- |
| `bctl` | Scan the exact configured project selection. |
| `bctl scan` | Run the same configured scan explicitly. |
| `bctl scan PATH...` | Scan repository roots or parent directories without loading or writing configuration. |
| `bctl projects [--root PATH]` | Replace the configured project selection through the terminal selector. |
| `bctl version` | Print bctl version metadata. |

All scan forms accept `--include-idle`, `--json`, `--no-refresh`, and
`--color=auto|always|never`. Positional scans reject `--config` because they
are deliberately config-free.

### Legacy Dashboard And Agent

`beacon` remains the complete configured dashboard, background-agent lifecycle,
Following workspace, Notes, integrations, repository-sync, diagnostics, and
macOS helper command. Start with:

```bash
beacon init
beacon
beacon doctor
beacon agent status
```

See the [public CLI contract](CONSTITUTION.md#public-cli-and-json-contracts) for
the supported command surface and the [user guide](USER_GUIDE.md#everyday-use)
for operational examples.

## Contributor Commands

Run commands from the repository root.

| Command | Purpose |
| --- | --- |
| `make build` | Build `bin/beacon` and `bin/bctl`. |
| `make install` | Install both Go executables. |
| `make fmt-check` | Verify Go formatting without modifying files. |
| `make vet` | Run `go vet ./...`. |
| `make test` | Run Go tests. |
| `make test-race` | Run Go tests with the race detector. |
| `make release-test` | Validate release-version behavior. |
| `make macos-build` | Build the Debug macOS application without code signing. |
| `make macos-test` | Run macOS XCTest coverage. |
| `make scan` / `make scan-json` | Run the configured bctl scan in terminal or JSON form. |
| `make hyper` | Build and launch the standalone Hyperlite companion app. |
| `make doctor` | Run the legacy Beacon diagnostic checks. |

Choose validation in proportion to the changed surface. The Constitution's
[completion contract](CONSTITUTION.md#validation-and-completion) is
authoritative.

## Kit Documentation Workflow

Discover exact behavior before choosing a Kit command:

```bash
kit rules list
kit capabilities <command> --json
```

Use `kit reconcile` as the reviewed structural refresh surface:

```bash
kit reconcile --all --include-files --dry-run --diff
kit reconcile --all --include-files
```

`kit init --refresh` is the lower-level direct managed-file refresh path. It
writes scaffold files, registry rulesets, instruction entrypoints, and Kit
configuration but does not semantically rewrite `docs/CONSTITUTION.md`.

After a structural refresh, review project-specific documentation and then run:

```bash
kit check --project
kit check --all # when feature specs or repository instruction files changed
git diff --check
```

`kit project refresh` generates the semantic documentation-review prompt.
`kit project refresh --now` records the reviewed Constitution cadence only
after that review is complete; it does not rewrite the Constitution.

## Pull-Request Review Feedback

`kit pr fix [--pr TARGET]` copies a dispatch prompt from current unresolved
review feedback. Editing is opt-in:

```bash
kit pr fix --pr 14
kit pr fix --pr 14 --edit
```

`--vim` and `--editor <command>` also opt into editing. The command does not
edit project files, stage, commit, push, comment, or resolve review threads.

## Delivery Boundaries

Before Git or GitHub mutation, load
[`docs/agents/GUARDRAILS.md`](agents/GUARDRAILS.md) and the applicable
[delivery rules](references/rules/github-pr-delivery.md). Use the existing
checkout when it owns the requested lane; otherwise follow the canonical
[worktree contract](references/worktrees.md).

The `[skip ci]` suffix is allowed only when the complete pull-request diff is
documentation-only and repository required-check policy permits the skip. A
documentation-only follow-up on a mixed product pull request is not eligible.
