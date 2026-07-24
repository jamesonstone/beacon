```text
██████╗ ███████╗ █████╗  ██████╗ ██████╗ ███╗   ██╗    \  |  /
██╔══██╗██╔════╝██╔══██╗██╔════╝██╔═══██╗████╗  ██║     .---.
██████╔╝█████╗  ███████║██║     ██║   ██║██╔██╗ ██║     |[_]|
██╔══██╗██╔══╝  ██╔══██║██║     ██║   ██║██║╚██╗██║     |   |
██████╔╝███████╗██║  ██║╚██████╗╚██████╔╝██║ ╚████║     |   |
╚═════╝ ╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝    /_____\

                        working-set memory for coding agents
```

Beacon keeps a small, durable memory of the work lanes that need your
attention. It combines local Git evidence, conservatively cached GitHub
evidence, factual changes, and optional notes to answer: what am I working on,
what changed, and what should I do next?

<!-- BEGIN KIT-MANAGED README BADGES -->
[![Last commit](https://img.shields.io/github/last-commit/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/commits) [![Open issues](https://img.shields.io/github/issues/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/issues) [![Pull requests](https://img.shields.io/github/issues-pr/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/pulls) [![CI](https://github.com/jamesonstone/beacon/actions/workflows/ci.yml/badge.svg)](https://github.com/jamesonstone/beacon/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/releases)
<!-- END KIT-MANAGED README BADGES -->

The Go CLI and its user-scoped background agent are the source of truth. The
native macOS app provides a menu-bar extra and detachable dashboard backed by
the same cached evidence.

## macOS App

### Following dashboard

![Beacon macOS dashboard showing active and waiting work lanes](docs/images/beacon-macos-dashboard.jpg)

### Signal Notes

![Beacon macOS Signal Notes panel with a Markdown release checklist](docs/images/beacon-macos-signal-notes.jpg)

## Requirements

- `git`
- [GitHub CLI](https://cli.github.com/) authenticated with `gh auth login`
- macOS 14 or later for the macOS app
- Go 1.26 and Xcode only when building from source

## Install

Download the CLI archive for your platform or the universal macOS app from the
[latest release](https://github.com/jamesonstone/beacon/releases), and verify
the asset with the published `checksums.txt`. The
[user guide](docs/USER_GUIDE.md#install-from-a-release) includes exact asset,
checksum, Gatekeeper, and upgrade instructions.

## Quick Start

```bash
gh auth login
beacon projects
beacon scan
```

`beacon projects` opens the hyper-light v2 project selector at
`~/go/src/github.com`. Enter owner directories, toggle repository directories,
move back with `..`, and save the selection to Beacon's config. `beacon scan`
then prints only dirty worktrees, non-base branches, unpublished commits, and
authored open pull requests for those projects, without starting the background
agent. Use `beacon projects --root PATH` to browse elsewhere.

Pass paths directly for an ad hoc scan that neither loads nor writes config:

```bash
beacon scan ~/go/src/github.com/jamesonstone/beacon \
  ~/go/src/github.com/jamesonstone/kit
```

Use `--include-idle` to show clean base-only projects, `--no-refresh` to skip
metadata fetches, or `--json` for the small versioned work-scan schema.

The full v1 dashboard and macOS application remain available while v2 is
dogfooded:

```bash
beacon init --source ~/go/src/github.com --yes
beacon agent install
beacon
```

Run `beacon init` without flags for the interactive repository selector. The
macOS app uses the same configuration and scan snapshot as the CLI.

Beacon observes repositories but does not edit working files, change branches,
push, review, or merge. See the [read-only boundary](docs/USER_GUIDE.md#read-only-boundary)
for the exact local state Beacon manages.

## Build From Source

```bash
make build
make test
make macos-build # macOS only
```

The CLI is written to `bin/beacon`. See the
[development and release commands](docs/USER_GUIDE.md#build-from-source) for
the complete build, test, install, and packaging workflow.

## Documentation

- [User guide](docs/USER_GUIDE.md) — installation, configuration, commands,
  integrations, macOS behavior, operations, and troubleshooting
- [Documentation index](docs/README.md) — contributor references, feature
  specifications, and repository guidance
- [Project releases](https://github.com/jamesonstone/beacon/releases) — signed
  checksums and downloadable CLI and macOS artifacts

## Maintainers

Maintained with 🪖 and ❤️ by [Jameson](https://github.com/jamesonstone) (`jamesonstone`).
