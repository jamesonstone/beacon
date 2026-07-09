# beacon
🔍 Signal Layer for Coding Agents

<!-- BEGIN KIT-MANAGED README BADGES -->
[![Last commit](https://img.shields.io/github/last-commit/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/commits) [![Open issues](https://img.shields.io/github/issues/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/issues) [![Pull requests](https://img.shields.io/github/issues-pr/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/pulls) [![Release](https://img.shields.io/github/v/release/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/releases)
<!-- END KIT-MANAGED README BADGES -->

Beacon scans configured Git repositories, linked worktrees, and GitHub pull requests to answer one question: which agent-driven work lanes are ready for review, and which need attention?

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

Create the default file:

```bash
beacon config init
beacon config validate
```

Example:

```yaml
version: 1

settings:
  scan_interval: 1m
  remote_refresh_interval: 5m
  stale_after: 24h
  max_parallel: 4
  github_author: "@me"

repositories:
  - name: beacon
    path: ~/go/src/github.com/jamesonstone/beacon
    github: jamesonstone/beacon
    base: main
    remote: origin
```

Configuration is strict: unknown fields, duplicate repository names, invalid durations, missing paths, and malformed GitHub names are rejected.

## CLI

```bash
beacon doctor
beacon scan
beacon scan --json
beacon scan --repo beacon
beacon open 'gh:jamesonstone/beacon#1'
beacon open-next
beacon config path
beacon config open
beacon version
```

`scan` groups lanes into Ready for Review, Needs Action, Waiting, Idle, and Errors. `scan --json` emits schema version 1 with no additional stdout logging, making it safe for the menu app and other automation.

## Read-only boundary

Scanning may run a timeout-bounded `git fetch --prune --no-tags` to refresh remote-tracking metadata. Beacon never edits working files, changes branches, pushes commits, creates pull requests, changes reviews, or merges work.

## Architecture

- `cmd/beacon` and `internal/` implement config, Git/GitHub evidence collection, lane correlation, policy, and output.
- `macos/Beacon` contains the SwiftUI `MenuBarExtra` app and its tests.
- The Xcode build embeds the Go executable as `Contents/MacOS/beacon-cli`; the standalone executable remains `beacon`.
- Work lanes are active Git worktrees plus open pull requests authored by the authenticated GitHub user. Unattached local branches are not scanned in v1.

## Troubleshooting

Run `beacon doctor` first. It checks `git`, `gh`, authentication, configuration, local repositories, and GitHub access. A repository-specific failure is reported in the snapshot without suppressing results from healthy repositories.

## Maintainers

Maintained with 🪖 and ❤️ by [Jameson](https://github.com/jamesonstone) (`jamesonstone`).
