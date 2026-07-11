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

- `git`
- [GitHub CLI](https://cli.github.com/) authenticated with `gh auth login`
- macOS 14 or later for the menu-bar app
- Go 1.26 and Xcode only when building from source

## Install From A Release

Each [GitHub release](https://github.com/jamesonstone/beacon/releases) contains:

- `beacon_<version>_darwin_arm64.tar.gz` for Apple Silicon Macs
- `beacon_<version>_darwin_amd64.tar.gz` for Intel Macs
- `beacon_<version>_linux_arm64.tar.gz` and `beacon_<version>_linux_amd64.tar.gz`
- `Beacon_<version>_macos_universal.zip` for the macOS menu application
- `checksums.txt` for SHA-256 verification

Download the archive for your platform from the latest release. For example,
to install the Apple Silicon CLI into `~/.local/bin`:

```bash
VERSION=0.1.0
ASSET="beacon_${VERSION}_darwin_arm64.tar.gz"
curl -LO "https://github.com/jamesonstone/beacon/releases/download/v${VERSION}/${ASSET}"
curl -LO "https://github.com/jamesonstone/beacon/releases/download/v${VERSION}/checksums.txt"
grep " ${ASSET}$" checksums.txt | shasum -a 256 -c -
tar -xzf "${ASSET}"
mkdir -p ~/.local/bin
install -m 0755 "beacon_${VERSION}_darwin_arm64/beacon" ~/.local/bin/beacon
beacon version
```

Use the `amd64` archive on an Intel Mac or x86-64 Linux machine. Ensure
`~/.local/bin` is on your `PATH`.

For the menu application, download the universal zip, verify it against
`checksums.txt`, expand it, and move `Beacon.app` into `/Applications`.

```bash
VERSION=0.1.0
ASSET="Beacon_${VERSION}_macos_universal.zip"
curl -LO "https://github.com/jamesonstone/beacon/releases/download/v${VERSION}/${ASSET}"
curl -LO "https://github.com/jamesonstone/beacon/releases/download/v${VERSION}/checksums.txt"
grep " ${ASSET}$" checksums.txt | shasum -a 256 -c -
ditto -x -k "${ASSET}" .
mv Beacon.app /Applications/
open /Applications/Beacon.app
```

Release applications are ad-hoc signed for bundle integrity but are not
Developer ID signed or notarized. If Gatekeeper blocks a checksum-verified
download from this repository, remove its quarantine attribute once, then open
it again:

```bash
xattr -dr com.apple.quarantine /Applications/Beacon.app
open /Applications/Beacon.app
```

Do not remove quarantine from an app you did not obtain from a verified Beacon
release.

## Quick Start

Authenticate GitHub, initialize Beacon with one or more repository roots, then
run the dashboard:

```bash
gh auth login
beacon init --source ~/go/src/github.com --yes
beacon
```

Run `beacon init` without flags for an interactive directory and repository
selector. The menu application uses the same configuration and displays the
same scan snapshot as the CLI.

## Build From Source

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

Release packaging is validated with:

```bash
make release-test
```

Local `make build` binaries report embedded VCS information as a revision-based
development version, including a `dirty` marker when applicable. Published
artifacts report the exact release SemVer and release commit instead.

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
  - path: ~/go/src/github.com/jamesonstone
  - path: ~/go/src/github.com/lsmc-bio
  - path: ~/go/src/github.com/limina-dev
  - path: ~/go/src/github.com/spectral7-ltd
  - path: ~/go/src/github.com/appliedsymbolics

repositories:
  - name: beacon
    path: ~/go/src/github.com/jamesonstone/beacon
    github: jamesonstone/beacon
    base: main
    remote: origin
```

Source entries are directory roots, not shell globs. A source such as
`~/go/src/github.com/jamesonstone` discovers every GitHub repository beneath
that directory on every scan; do not store a trailing `/*` in the YAML.

Use `repositories` only when you want to watch one repository explicitly or
override its discovered name, GitHub slug, base branch, or remote. Beacon
deduplicates overlapping sources and explicit repositories.

`github_scope: mine` includes PRs authored by `github_author` and issues
assigned to that identity. Use `all` to include every open PR and issue in each
discovered project. Explicit repository metadata overrides a discovery for the
same local or GitHub repository.

Configuration is strict: unknown fields, duplicate names or sources, invalid
durations or scope, missing paths, and malformed GitHub names are rejected.
Existing version-1 files remain readable and are migrated only by a confirmed
init operation.

Project attention choices are stored separately in the sibling
`$HOME/.config/beacon/tracking.yaml` file. Beacon creates this managed file only
after you explicitly untrack a project; do not add exclusions to source roots
or remove repositories from `config.yaml` just to quiet the dashboard. With a
custom `--config` or `BEACON_CONFIG`, `tracking.yaml` is stored beside that
resolved configuration file.

## Everyday Use

```bash
beacon
beacon --include-idle
beacon --color=always
beacon doctor
beacon scan
beacon scan --include-idle
beacon scan --json
beacon scan --color=never
beacon scan --repo beacon
beacon projects
beacon projects --untracked
beacon projects untrack owner/old-project
beacon projects track owner/old-project
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

Idle work is treated as inventory instead of queue content. Human output hides
all-idle projects by default and replaces them with a compact count; pass
`--include-idle` to list those quiet projects. `beacon scan --repo NAME` always
shows the selected project even when it is idle. An idle base lane is omitted
when its project already has active work. JSON remains complete regardless of
these presentation filters.

Use `beacon projects` in a terminal for a searchable multi-select of every
discovered project. Deselecting a project moves all of its lanes out of the
active, quiet, and top-item views; `beacon projects --untracked` shows that
secondary inventory. Repository-scoped warnings and errors from untracked
projects remain in JSON but no longer consume the human dashboard; global
diagnostics remain visible. The explicit `track` and `untrack` subcommands
accept a stable `owner/repository` identity or a unique discovered project name.

Untracking records the project's current Git and GitHub evidence as a baseline.
Existing work therefore stays quiet, but any later commit, worktree-state,
issue, pull-request, check, review, feedback, or merge-state change permanently
restores the project to tracked views. Beacon will not establish or compare a
baseline while the project's evidence scan has errors, preventing missing
GitHub or Git data from causing a false reactivation.

The menu application exposes the same controls under **Projects**, with
separate **Tracked** and **Untracked** tabs, search, and Track/Untrack buttons.
It delegates changes to the bundled CLI, refreshes the shared snapshot after a
mutation, and shows an automatic-reactivation banner when new activity restores
a project.

When bare `beacon` performs a live scan in an interactive terminal, a rotating
lighthouse sweep shows a shuffled deck of 150 original odd trivia facts. A new
fact arrives after a random one-to-five-second interval, and no fact repeats
during one command run. Facts are truncated to the current terminal width. The
animation is omitted from explicit `beacon scan` commands, redirected output,
and JSON. Cursor state is restored even when a scan fails or is cancelled.

Non-blocking discovery, prunable-worktree, search-truncation, and optional Kit
progress diagnostics contribute to the warning count in the dashboard header;
they do not appear as fatal errors. Full warning detail remains available in
`beacon scan --json`. The red `Errors` section is reserved for failures that
prevent Beacon from collecting expected evidence.

`scan --json` emits schema version 2 with projects, ordered lanes, issues,
checks, feedback, optional Kit progress, scoped warnings, and scoped errors. It
never emits ANSI or additional stdout logging, making it safe for the menu app
and automation.

Common workflows:

- Run `beacon` for the colorful project dashboard.
- Run `beacon --include-idle` when auditing quiet projects.
- Run `beacon projects` to curate the tracked project set.
- Run `beacon projects --untracked` to inspect deliberately quieted projects.
- Run `beacon open-next` to open the highest-priority review or action item.
- Run `beacon scan --repo NAME` to focus on one configured project.
- Run `beacon scan --json` for scripts or diagnostics.
- Click a menu-app lane to open its pull request, issue, or local worktree.
- Run `beacon init --source <new-root> --yes` to merge another persistent source
  into the existing configuration without removing current entries.

## Updating

Merges to `main` create one semantic version for both the CLI and menu app.
Breaking Conventional Commits bump the major version, `feat` commits bump the
minor version, and all other accepted commits bump the patch version. GitHub
generates release notes from the merged changes.

To update, download and checksum the matching assets from the latest release,
replace the CLI binary and/or `/Applications/Beacon.app`, and confirm the CLI
version:

```bash
beacon version
```

Upgrading does not rewrite `$HOME/.config/beacon/config.yaml`.

## Read-only boundary

Scanning may run a timeout-bounded `git fetch --prune --no-tags` to refresh
remote-tracking metadata. Beacon never edits working files, changes branches,
pushes commits, creates pull requests, changes reviews, or merges work. Beacon
writes only its own configuration during confirmed `beacon init` operations and
its own managed `tracking.yaml` when the user changes project tracking or new
evidence automatically reactivates a previously untracked project.

## Architecture

- `cmd/beacon` and `internal/` implement config, source discovery, Git/GitHub/Kit evidence collection, lane correlation, managed tracking state, policy, and output.
- `macos/Beacon` contains the SwiftUI `MenuBarExtra` app and its tests.
- The Xcode build embeds the Go executable as `Contents/MacOS/beacon-cli`; the standalone executable remains `beacon`.
- `.github/workflows/release.yml` validates, versions, packages, and publishes both products after a merge reaches `main`.
- Work lanes are active Git worktrees, scoped open pull requests, and scoped open issues. Unattached local branches are not scanned.

## Troubleshooting

Run `beacon doctor` first. It checks `git`, `gh`, authentication, configuration, local repositories, and GitHub access. A repository-specific failure is reported in the snapshot without suppressing results from healthy repositories.

## Maintainers

Maintained with 🪖 and ❤️ by [Jameson](https://github.com/jamesonstone) (`jamesonstone`).
