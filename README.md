```text
██████╗ ███████╗ █████╗  ██████╗ ██████╗ ███╗   ██╗
██╔══██╗██╔════╝██╔══██╗██╔════╝██╔═══██╗████╗  ██║
██████╔╝█████╗  ███████║██║     ██║   ██║██╔██╗ ██║
██╔══██╗██╔══╝  ██╔══██║██║     ██║   ██║██║╚██╗██║
██████╔╝███████╗██║  ██║╚██████╗╚██████╔╝██║ ╚████║
╚═════╝ ╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝

                        working-set memory for coding agents
```

Beacon keeps a small, durable memory of the three to eight work lanes that need
your attention. It combines near-real-time local Git evidence, conservatively
cached GitHub evidence, factual changes since you last looked, and optional
short notes to answer: what am I working on, what changed, and what should I do
next?

<!-- BEGIN KIT-MANAGED README BADGES -->
[![Last commit](https://img.shields.io/github/last-commit/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/commits) [![Open issues](https://img.shields.io/github/issues/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/issues) [![Pull requests](https://img.shields.io/github/issues-pr/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/pulls) [![CI](https://github.com/jamesonstone/beacon/actions/workflows/ci.yml/badge.svg)](https://github.com/jamesonstone/beacon/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/jamesonstone/beacon)](https://github.com/jamesonstone/beacon/releases)
<!-- END KIT-MANAGED README BADGES -->

The Go CLI and its user-scoped background agent are the source of truth. The
native macOS application provides both a menu-bar extra and a detachable Dock
window; both consume the same cached schema-v3 snapshots and incremental agent
events through a bundled copy of the executable.

## Requirements

- `git`
- [GitHub CLI](https://cli.github.com/) authenticated with `gh auth login`
- macOS 14 or later for the macOS app
- Go 1.26 and Xcode only when building from source

## Install From A Release

Each [GitHub release](https://github.com/jamesonstone/beacon/releases) contains:

- `beacon_<version>_darwin_arm64.tar.gz` for Apple Silicon Macs
- `beacon_<version>_darwin_amd64.tar.gz` for Intel Macs
- `beacon_<version>_linux_arm64.tar.gz` and `beacon_<version>_linux_amd64.tar.gz`
- `Beacon_<version>_macos_universal.zip` for the universal macOS application
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

For the macOS application, download the universal zip, verify it against
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
beacon agent install
beacon
```

Run `beacon init` without flags for an interactive directory and repository
selector. The macOS application uses the same configuration and displays the
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

Build, test, or launch the macOS app with:

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
installs packages or removes existing sources or repositories; interactive
setup may separately offer to enable Beacon's current-user background agent.

Example:

```yaml
version: 2

settings:
  scan_interval: 1m
  remote_refresh_interval: 45m
  stale_after: 24h
  max_parallel: 4
  github_author: '@me'
  github_scope: mine
  tracked_refresh_interval: 1m
  untracked_probe_interval: 10m

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

Legacy project attention choices are stored separately in
`$HOME/.local/state/beacon/tracking.json`. Beacon creates this managed file only
after you explicitly untrack a project; do not add exclusions to source roots
or remove repositories from `config.yaml` just to quiet the dashboard. Existing
sibling `tracking.yaml` state is migrated automatically and archived with a
`.migrated` suffix.

`tracked_refresh_interval` defaults to one minute and controls complete
local observations. These frequent observations do not fetch or contact
GitHub. `untracked_probe_interval` defaults to ten minutes and
controls inexpensive local/GitHub summary probes for muted projects. If you
manage a large deliberately quiet inventory, increase it to `1h` to reduce
background GitHub traffic while retaining automatic reactivation. The agent
checks cached due times before discovery, so a scheduler tick with no due
project performs no source walk, `git fetch`, or `gh` command.

Beacon shares a persistent user-only cache across pull requests, issues, review
feedback, and muted probes. Most GitHub evidence is
reused for `remote_refresh_interval`; stable repository metadata is reused for
seven days. Cached activity can provide bounded stale fallback for 24 hours,
and repository metadata for 30 days, when GitHub capacity is protected. Set
`remote_refresh_interval: 45m` for a conservative daily-driver profile.

Before a cache miss Beacon reads `gh api rate_limit` and preserves 2,500
GraphQL points, 15 Search requests, and 1,500 REST Core requests for your own
interactive `gh` use. GitHub cache misses are serialized per rate bucket,
GraphQL work is conservatively budgeted at 25 points per command, and the
authoritative allowance is refreshed after at most five calls. When a bucket
reaches its reserve, Beacon serves its last successful cached result when
available and pauses new calls until the reported reset. Source discovery uses
only local Git remote and branch metadata and never calls GitHub. Beacon
never changes your `gh` credentials or stores a token. Cached evidence is
stored with user-only permissions under `$HOME/.cache/beacon/github/`.

With the default `github_scope: mine`, every due-project batch uses one global
authored-PR search and one global assigned-issue search, independent of the
number of configured repositories. Beacon enriches only matching PRs with
activity in the last six hours during background collection; explicit scans
and lane refreshes can inspect older work. Muted projects share that same
batched evidence instead of polling each repository. `github_scope: all`
is intentionally more expensive because it must enumerate repository-scoped
work.

## Everyday Use

```bash
beacon
beacon --no-watch
beacon --include-idle
beacon --color=always
beacon doctor
beacon lanes
beacon lanes --parked
beacon pin 'gh:jamesonstone/beacon#5'
beacon park 'git:jamesonstone/beacon@GH-5'
beacon resume 'git:jamesonstone/beacon@GH-5'
beacon note 'git:jamesonstone/beacon@GH-5' 'finish the macOS smoke test'
beacon seen 'git:jamesonstone/beacon@GH-5'
beacon add --manual 'Research a smaller cache format'
beacon scan
beacon scan --include-idle
beacon scan --json
beacon scan --color=never
beacon scan --repo beacon
beacon projects
beacon select
beacon projects --tracked
beacon projects --untracked
beacon untrack owner/old-project
beacon untrack owner/one owner/two owner/three
beacon track owner/old-project
beacon projects untrack owner/old-project
beacon projects track owner/old-project
beacon refresh
beacon refresh beacon
beacon agent status
beacon open 'gh:jamesonstone/beacon#2'
beacon open-next
beacon config path
beacon config open
beacon version
```

Bare `beacon` connects to the user background agent, renders the cached working
set,
and observes scheduled updates without requesting a refresh. Opening or
reconnecting either client is cache-only. Use `beacon refresh`, the macOS
`Scan Now` action, or `beacon scan` when you explicitly want current evidence.
`--no-watch` renders the cache and exits. If no agent is available, bare execution
returns an installation hint instead of silently paying direct-scan latency;
use `beacon agent install` to restore cache-first operation or `beacon scan` for
an explicit blocking scan.

`beacon scan` remains the explicit, blocking diagnostic path and returns the
complete repository inventory. `scan --json` remains deterministic, ANSI-free, and
does not require the agent. `--color=auto|always|never` controls human styling;
auto requires a TTY and honors `NO_COLOR`.

The default working-set view groups lanes as **Active**, **Waiting**,
**Recently Active**, and **Parked**. Dirty or unpublished work, recent local
commits, recent authored PRs, pinned lanes, and manual lanes can enter the
working set. Old authored PRs stay out unless pinned. `beacon lanes --parked`
reveals parked lanes without allowing a large historical inventory to consume
the primary view.

Lane notes are optional memory cues, never status truth. Beacon stores them
with attention and last-seen observations in the user-only strict JSON file
`$HOME/.local/state/beacon/lanes.json`. When Git or GitHub evidence changes
after a note, both clients label that note as stale and show a factual delta
such as `new commit observed`, `PR #5 opened`, or `CI changed from pending to
success`. Manual lanes support planning or research without requiring Git,
GitHub, Kit, or a Codex task API.

Idle work is treated as inventory instead of queue content. Human output hides
all-idle projects by default and replaces them with a compact count; pass
`--include-idle` to list those quiet projects. `beacon scan --repo NAME` always
shows the selected project even when it is idle. An idle base lane is omitted
when its project already has active work. JSON remains complete regardless of
these presentation filters.

Project Track/Untrack remains a compatibility inventory control rather than
the primary attention model. Use `beacon select` (or the compatible bare
`beacon projects`) in a terminal
for a colorful searchable multi-select of every discovered project. Existing
tracked projects start highlighted; use the arrow keys to scroll, Space to
toggle a project, `/` to filter, and Enter to confirm. Deselecting a project
moves all of its lanes out of the
active, quiet, and top-item views; `beacon projects --untracked` shows that
secondary inventory. Repository-scoped warnings and errors from untracked
projects remain in JSON but no longer consume the human dashboard; global
diagnostics remain visible. The explicit `track` and `untrack` subcommands
accept a stable `owner/repository` identity or a unique discovered project name.
Multiple arguments are applied as one atomic agent mutation without a GitHub
scan.

Untracking records the project's current Git and GitHub evidence as a baseline.
Existing work therefore stays quiet, but any later commit, worktree-state,
issue, pull-request, check, review, feedback, or merge-state change permanently
restores the project to tracked views. If the latest evidence scan has errors,
an explicit deselection is still authoritative: Beacon records a pending
baseline, keeps the project untracked, and initializes the baseline from the
first later complete scan without reactivation. Incomplete evidence is never
compared as if it were a material change.

The macOS application exposes the same controls under **Projects**, with
separate **Tracked** and **Untracked** tabs, search, and Track/Untrack buttons.
It sends changes through the shared agent protocol, consumes the same cached
snapshot as the CLI, and shows an automatic-reactivation banner when new
activity restores a project.

Track and Untrack actions are optimistic and nonblocking. Each selection moves
the project immediately and joins a visible background queue. The Go agent
acknowledges from cached evidence without a network probe; incomplete evidence
creates a pending baseline that a later complete scan initializes. The next
scheduled muted probe establishes the compact comparison baseline. Beacon
processes the queue in selection order, keeps navigation and scanning
available, and rolls back only the affected project if a request fails.

## macOS Dashboard

Beacon remains in the menu bar and also runs as a regular macOS application,
so its neon-space icon is available in the Dock and Command-Tab when a camera
notch or a crowded menu bar hides the menu item. Ordinary launches open one
compact dashboard window. Close the window to keep Beacon running quietly;
choose **Window** in the menu extra or activate Beacon from the Dock or
Command-Tab to reopen the same window.

The menu and detached window are two views over one shared background-agent
connection. They show the same active, quiet, and untracked projects and never
start duplicate repository scans.

Use **Open Beacon at Login** in either view to enable quiet startup. Beacon
registers its embedded login helper through macOS Service Management. A login
launch starts without opening the dashboard; selecting Beacon later opens it.
If macOS requires approval, choose **Approve in Settings** and enable Beacon in
System Settings > General > Login Items. This preference is off by default.

The rotating lighthouse trivia loader remains available during interactive
onboarding before the first background cache is ready.

Non-blocking discovery, prunable-worktree, search-truncation, and optional Kit
progress diagnostics contribute to the warning count in the dashboard header;
they do not appear as fatal errors. Full warning detail remains available in
`beacon scan --json`. The red `Errors` section is reserved for failures that
prevent Beacon from collecting expected evidence.

`scan --json` emits schema version 2 with projects, ordered lanes, issues,
checks, feedback, optional Kit progress, scoped warnings, and scoped errors. It
never emits ANSI or additional stdout logging, making it safe for the macOS app
and automation.

Common workflows:

- Run `beacon` for the colorful project dashboard.
- Run `beacon --include-idle` when auditing quiet projects.
- Run `beacon select` to curate the tracked project set interactively.
- Run `beacon projects --untracked` to inspect deliberately quieted projects.
- Run `beacon refresh [project]` to request background work without blocking.
- Run `beacon agent status` to inspect the process, socket, cache count, and active refresh.
- Run `beacon open-next` to open the highest-priority review or action item.
- Run `beacon scan --repo NAME` to focus on one configured project.
- Run `beacon scan --json` for scripts or diagnostics.
- Click a macOS-app lane to open its pull request, issue, or local worktree.
- Run `beacon init --source <new-root> --yes` to merge another persistent source
  into the existing configuration without removing current entries.

## Updating

Merges to `main` create one semantic version for both the CLI and macOS app.
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

## Background Agent

Beacon uses the same executable for interactive commands and background work:

```bash
beacon agent install
beacon agent status
beacon agent stop
beacon agent uninstall
```

On macOS, installation creates the current-user LaunchAgent at
`~/Library/LaunchAgents/com.jamesonstone.beacon.agent.plist`. It runs only as
the current user, uses the authenticated `gh` CLI, and never stores GitHub
credentials. Authenticate `gh` persistently with `gh auth login`; environment-
only tokens are not copied into the LaunchAgent.

Operational files are user-only:

```text
~/.local/state/beacon/tracking.json
~/.cache/beacon/projects/*.json
~/.cache/beacon/agent.sock
~/.cache/beacon/agent.pid
~/Library/Logs/Beacon/agent.log
~/Library/Logs/Beacon/agent-error.log
```

One bounded worker pool scans tracked projects independently. Untracked
projects receive lightweight probes at the slower configured interval; Beacon
runs a full scan only after a material delta is detected. Duplicate refreshes
coalesce, and a project never has overlapping jobs.

## Read-only boundary

Scanning may run a timeout-bounded `git fetch --prune --no-tags` to refresh
remote-tracking metadata. Beacon never edits working files, changes branches,
pushes commits, creates pull requests, changes reviews, or merges work. Beacon
writes only its own configuration during confirmed `beacon init` operations and
its own user-scoped tracking state, cache, PID/socket, LaunchAgent, and rotated
logs. New evidence may automatically update tracking state when a previously
untracked project is reactivated.

## Architecture

- `cmd/beacon` and `internal/` implement config, source discovery, Git/GitHub/Kit evidence collection, lane correlation, managed tracking state, cache/protocol/scheduling, policy, and output.
- `macos/Beacon` contains the shared SwiftUI menu/dashboard app, embedded login
  item, app icon, and tests.
- The Xcode build embeds the Go executable as `Contents/MacOS/beacon-cli`; the standalone executable remains `beacon`.
- `.github/workflows/release.yml` validates, versions, packages, and publishes both products after a merge reaches `main`.
- Work lanes are active Git worktrees, scoped open pull requests, and scoped open issues. Unattached local branches are not scanned.

## Troubleshooting

Run `beacon doctor` first. It checks `git`, `gh`, authentication, configuration,
tracking-state migration, the background agent, local repositories, and GitHub
access. A repository-specific failure is reported without suppressing healthy
cached or freshly scanned projects.

If the agent is unavailable, inspect `beacon agent status` and the files under
`~/Library/Logs/Beacon/`. Reinstalling the LaunchAgent does not remove caches or
tracking choices. To reset only cached evidence, stop the agent, remove
`~/.cache/beacon/projects/`, and start it again with `beacon agent install`;
do not remove `tracking.json` unless you intentionally want to restore every
project to Tracked.

## Maintainers

Maintained with 🪖 and ❤️ by [Jameson](https://github.com/jamesonstone) (`jamesonstone`).
