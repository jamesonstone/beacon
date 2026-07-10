```text
██████╗ ███████╗ █████╗  ██████╗ ██████╗ ███╗   ██╗
██╔══██╗██╔════╝██╔══██╗██╔════╝██╔═══██╗████╗  ██║
██████╔╝█████╗  ███████║██║     ██║   ██║██╔██╗ ██║
██╔══██╗██╔══╝  ██╔══██║██║     ██║   ██║██║╚██╗██║
██████╔╝███████╗██║  ██║╚██████╗╚██████╔╝██║ ╚████║
╚═════╝ ╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝

                            signal layer for coding agents
```

Beacon discovers local GitHub projects and correlates linked worktrees, branches, pull requests, issues, checks, review feedback, and optional Kit feature evidence to answer one question: what needs attention next?

<!-- BEGIN KIT-MANAGED README BADGES -->

[![Last commit](https://img.shields.io/github/last-commit/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/commits) [![Open issues](https://img.shields.io/github/issues/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/issues) [![Pull requests](https://img.shields.io/github/issues-pr/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/pulls) [![CI](https://github.com/jamesonstone/beacon/actions/workflows/ci.yml/badge.svg)](https://github.com/jamesonstone/beacon/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/releases)

<!-- END KIT-MANAGED README BADGES -->

The Go CLI is the source of truth. The native macOS menu-bar app polls the same versioned JSON contract through a bundled copy of the CLI.

## Requirements

- macOS 14 or later for the menu-bar app
- Go 1.26 or later
- `git`
- [GitHub CLI](https://cli.github.com/) authenticated with `gh auth login`
- Xcode to build the menu-bar app

## Build

```bash
make build
make test
```

The standalone executable is written to `bin/beacon`. Install it on your `PATH` with:

```bash
make install
```

Build, test, or launch the menu-bar app with:

```bash
make macos-build
make macos-test
make macos-run
```

## Configuration

Beacon resolves configuration in this order:

1. `--config <path>`
2. `BEACON_CONFIG`
3. `$HOME/.config/beacon/config.yaml`

Let Beacon create or extend the default file. A repository path becomes an
explicit repository; a parent directory is persisted as a source and
rediscovered on every scan.

```bash
beacon init --source ~/go/src/github.com/jamesonstone/beacon --yes
beacon init --source ~/go/src/github.com --source ~/Projects --yes
# Or use the interactive directory/repository selector:
beacon init

beacon config validate
```

`beacon config init` remains an alias for `beacon init`. Initialization checks
for `git`, `gh`, and GitHub authentication, previews non-destructive changes,
and atomically writes the configuration only after confirmation. It never
installs software or removes existing sources or repositories.

Example:

```yaml
version: 2

settings:
  scan_interval: 1m
  remote_refresh_interval: 5m
  stale_after: 24h
  max_parallel: 4
  github_author: '@me'
  github_scope: mine

sources:
  - path: ~/go/src/github.com

repositories:
  - name: beacon
    path: ~/go/src/github.com/jamesonstone/beacon
    github: jamesonstone/beacon
    base: main
    remote: origin
```

`github_scope: mine` includes PRs authored by `github_author` and issues
assigned to that identity. Use `all` to include every open PR and issue in each
discovered project. Explicit repository metadata overrides a discovery for the
same local or GitHub repository.

Configuration is strict: unknown fields, duplicate names or sources, invalid
durations or scope, missing paths, and malformed GitHub names are rejected.
Existing version-1 files remain readable and are migrated only by a confirmed
init operation.

## CLI

```bash
beacon
beacon --color=always
beacon doctor
beacon scan
beacon scan --json
beacon scan --color=never
beacon scan --repo beacon
beacon open 'gh:jamesonstone/beacon#2'
beacon open-next
beacon config path
beacon config open
beacon version
```

Bare `beacon` and human-readable `beacon scan` render the same project-grouped
dashboard with project, work item, status, last durable progress, and next
action. `--color=auto|always|never` controls ANSI styling; auto requires a TTY
and honors `NO_COLOR`. Narrow terminals use wrapped evidence rows.

`scan --json` emits schema version 2 with projects, ordered lanes, issues,
checks, feedback, optional Kit progress, and scoped errors. It never emits ANSI
or additional stdout logging, making it safe for the menu app and automation.

## Read-only boundary

Scanning may run a timeout-bounded `git fetch --prune --no-tags` to refresh
remote-tracking metadata. Beacon never edits working files, changes branches,
pushes commits, creates pull requests, changes reviews, or merges work. The only
other write is an explicit, confirmed `beacon init` update to Beacon's own
configuration file.

## Architecture

- `cmd/beacon` and `internal/` implement config, source discovery, Git/GitHub/Kit evidence collection, lane correlation, policy, and output.
- `macos/Beacon` contains the SwiftUI `MenuBarExtra` app and its tests.
- The Xcode build embeds the Go executable as `Contents/MacOS/beacon-cli`; the standalone executable remains `beacon`.
- Work lanes are active Git worktrees, scoped open pull requests, and scoped open issues. Unattached local branches are not scanned.

## Troubleshooting

Run `beacon doctor` first. It checks `git`, `gh`, authentication, configuration, local repositories, and GitHub access. A repository-specific failure is reported in the snapshot without suppressing results from healthy repositories.

## Maintainers

Maintained with 🪖 and ❤️ by [Jameson](https://github.com/jamesonstone) (`jamesonstone`).
