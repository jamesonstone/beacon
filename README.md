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
short notes and tags to answer: what am I working on, what changed, and what
should I do next?

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

Project-following choices are stored separately in the strict, versioned
`$HOME/.local/state/beacon/tracking.json`. Configuration defines what Beacon can
discover; this managed state defines which discovered repositories you
deliberately follow. New discoveries begin in Quiet. Existing version-1 JSON
and sibling `tracking.yaml` choices migrate automatically without changing the
current followed set; migrated YAML is archived with a `.migrated` suffix.

`tracked_refresh_interval` defaults to one minute and controls complete
local observations. These frequent observations do not fetch or contact
GitHub. `untracked_probe_interval` defaults to ten minutes and
controls inexpensive local/GitHub summary probes for projects outside
Following. If you manage a large quiet inventory, increase it to `1h` to reduce
background GitHub traffic while retaining activity awareness. New evidence
never changes Following membership. The agent
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
beacon --include-idle
beacon --color=always
beacon doctor
beacon lanes
beacon lanes --parked
beacon pin 'gh:jamesonstone/beacon#5'
beacon park 'git:jamesonstone/beacon@GH-5'
beacon resume 'git:jamesonstone/beacon@GH-5'
beacon note 'git:jamesonstone/beacon@GH-5' 'finish the macOS smoke test'
beacon notes
beacon notes append 'Retest the merged release before lunch.'
printf '# Signal Log\n\n- verify PR #10\n' | beacon notes set
beacon notes edit
beacon notes path
beacon tag 'git:jamesonstone/beacon@GH-5' 'manual test'
beacon untag 'git:jamesonstone/beacon@GH-5' 'manual test'
beacon seen 'git:jamesonstone/beacon@GH-5'
beacon add --manual 'Research a smaller cache format'
beacon scan
beacon scan --include-idle
beacon scan --json
beacon scan --color=never
beacon scan --repo beacon
beacon sync
beacon sync check --no-fetch
beacon sync check --json
beacon sync apply owner/repository --yes
beacon limits
beacon limits --json
beacon projects
beacon select
beacon projects --followed
beacon projects --recent
beacon projects --quiet
beacon follow owner/important-project
beacon unfollow owner/old-project
beacon unfollow owner/one owner/two owner/three
# Compatibility aliases for existing scripts:
beacon track owner/important-project
beacon untrack owner/old-project
beacon refresh
beacon refresh beacon
beacon agent status
beacon open 'gh:jamesonstone/beacon#2'
beacon open-next
beacon config path
beacon config open
beacon version
```

Bare `beacon` is a manual refresh every time: it asks the background agent to
check current Git and GitHub evidence, waits for the coalesced scan to finish,
and renders the updated working set. If the agent is unavailable, Beacon runs
the same blocking foreground scan instead. Opening or reconnecting the macOS
app remains cache-only; scheduled background collection retains its
conservative cadence. Use `beacon refresh [project]` when you want to queue
background work without waiting, or `beacon scan` for the complete diagnostic
inventory.

`beacon scan` remains the explicit, blocking diagnostic path and returns the
complete repository inventory. `scan --json` remains deterministic, ANSI-free, and
does not require the agent. `--color=auto|always|never` controls human styling;
auto requires a TTY and honors `NO_COLOR`.

`beacon sync` is the explicit Git-only path for finding configured repositories
whose checked-out branch or local default branch is behind its fetched remote
default branch. In a terminal it fetches only the configured default-branch ref,
preselects every safe candidate, and asks once before updating one or many
repositories. Use `beacon sync check --no-fetch` for a network-free view of
existing local refs, or `beacon sync check` to run bounded
`git fetch --prune --no-tags` checks. `sync apply` requires named repositories
and `--yes` outside an interactive terminal.

Automatic sync is intentionally narrow: Beacon can fast-forward a clean
checked-out default branch, or return a clean fully merged feature branch to
the default branch and fast-forward it. Dirty, detached, diverged, missing-ref,
unmerged, and multi-worktree cases remain manual. This workflow never invokes
`gh` or the GitHub API and never rebases, resets, stashes, deletes, commits,
pushes, or changes GitHub state.

`beacon limits` is an explicit snapshot of the external rate-limited dependency
Beacon currently uses: authenticated `gh`. One invocation runs one bounded
`gh api rate_limit` request and shows GraphQL, REST Core, and Search usage,
remaining allowance, and reset time. Beacon never runs this command at startup
or on a schedule; JSON output is available for the bundled macOS helper.

The default working-set view groups lanes as **Active**, **Waiting**,
**Recently Active**, and **Parked**. Dirty or unpublished work, recent local
commits, recent authored PRs, pinned lanes, and manual lanes can enter the
working set. Old authored PRs stay out unless pinned. `beacon lanes --parked`
reveals parked lanes without allowing a large historical inventory to consume
the primary view.

Lane notes and tags are optional memory cues, never status truth. Beacon stores them
with attention and last-seen observations in the user-only strict JSON file
`$HOME/.local/state/beacon/lanes.json`. When Git or GitHub evidence changes
after a note, both clients label that note as stale and show a factual delta
such as `new commit observed`, `PR #5 opened`, or `CI changed from pending to
success`. Tags are short, deduplicated labels that can be added or removed from
the CLI and macOS lane cards; they never affect Beacon's attention or action
policy. Manual lanes support planning or research without requiring Git,
GitHub, Kit, or a Codex task API.

`beacon notes` is a separate global Markdown scratchpad for real-time thoughts
that span lanes. `show`, `set`, `append`, `edit`, and `path` subcommands all use
`$XDG_DATA_HOME/beacon/notes.md`, defaulting to
`$HOME/.local/share/beacon/notes.md`. The document is size-bounded, atomically
saved with user-only permissions, and never interpreted as Git/GitHub evidence.
On macOS, `beacon notes edit` waits for the editor to close and then publishes
the saved document to running Beacon clients; other platforms can use `set`,
`append`, or edit the path directly while Beacon is stopped.

Idle work inside Following is treated as inventory instead of queue content.
Human output hides idle followed projects by default and replaces them with a
compact count; pass `--include-idle` to list that inventory. `beacon scan
--repo NAME` always shows the selected project even when it is idle. An idle base lane is omitted
when its project already has active work. JSON remains complete regardless of
these presentation filters.

Following is an explicit repository-level choice, independent of lane attention.
Use `beacon select` (or the compatible bare `beacon projects`) for a colorful,
searchable multi-select of every discovered project. Followed projects start
highlighted; use the arrow keys to scroll, Space to toggle a project, `/` to
filter, and Enter to confirm. `beacon follow` and `beacon unfollow` accept a
stable `owner/repository` identity or a unique discovered name. The established
`track` and `untrack` commands remain compatibility aliases. Multiple arguments
are applied as one atomic agent mutation without a GitHub scan.

Projects outside Following retain an evidence baseline. Later local or GitHub
changes move them to **Recently Updated** for `settings.stale_after`—24 hours by
default—without silently following them. You can inspect the factual reason and
decide whether to Follow. When the window expires, they return to **Quiet**.
Incomplete evidence is never compared as if it were a material change.
Repository-scoped diagnostics remain in JSON without flooding the focused lane
dashboard.

Follow and Stop Following actions are optimistic and nonblocking. Each
selection moves the project immediately and joins a visible background queue.
The Go agent
acknowledges from cached evidence without a network probe; incomplete evidence
creates a pending baseline that a later complete scan initializes. The next
scheduled muted probe establishes the compact comparison baseline. Beacon
processes the queue in selection order, keeps navigation and scanning
available, and rolls back only the affected project if a request fails.

## macOS Dashboard

Beacon remains in the menu bar and also runs as a regular macOS application,
so its neon-space icon is available in the Dock and Command-Tab when a camera
notch or a crowded menu bar hides the menu item. Ordinary launches open one
dashboard window at a focused 580-point width and the full usable screen
height. Close the window to keep Beacon running quietly;
choose **Open Dashboard** in the top-right Settings menu or activate Beacon from the Dock or
Command-Tab to reopen the same window.

The menu and detached window are two views over one shared background-agent
connection. They show the same Following, Parking Lot, Recently Updated, and
Quiet views and never
start duplicate repository scans. Secondary actions live in the top-right gear
menu so lane evidence receives the full height. The adjacent view button
switches between the default stacked list, horizontal state tiles, and an
experimental kanban board; the selection persists across launches.

A dedicated neon refresh button in the top-right of both surfaces performs
**Scan Now**. Use it after merging one or several pull requests to bypass the
normal evidence cache, run one coalesced batched refresh, and update both views.
Repeated clicks cannot start overlapping scans.

The adjacent **Repository Sync** button first shows a network-free comparison
against existing remote-tracking refs. Its badge counts repositories that need
attention. **Check for Updates** explicitly fetches only each configured remote
default branch; row, selected, and all-safe buttons then run the same guarded
fast-forward behavior as `beacon sync`. Both macOS surfaces delegate this work
to Go and never execute Git or `gh` directly.

The next **Dependency Limits** button is also explicit-only. Selecting it asks
the bundled Go helper for one `gh api rate_limit` snapshot and shows the
GraphQL, REST Core, and Search buckets. After the first check, the button shows
the highest usage percentage in mint below 50%, gold from 50% through 75%, and
coral above 75%; zero usage retains the gauge icon. No startup request or
background polling is added.

A compact tab row keeps repository attention one click away. **Following** is
selected whenever a dashboard surface opens and contains Active, Waiting, and
Recently Active lanes. **Parking Lot** is the next peer tab, followed by
**Recently Updated**, the outside-activity inbox, and **Quiet**, the remaining
discovered inventory. Both outside views are searchable and provide a
nonblocking Follow action. Settings keeps only the Following manager instead of
duplicating those primary tabs. Any destination control opens its page on the
first selection and returns to **Following** when selected again; selecting a
different destination switches directly to it. This applies to the peer tabs,
Repository Sync, Dependency Limits, and Manage Following.

When Following has no work in progress and no projects are still loading, the
blank lane area becomes a lightweight **All caught up** backsplash. Its native
SwiftUI orbit adapts to the compact menu extra and the detached window, respects
Reduce Motion, and describes lane state without claiming local Git refs are current.

The Beacon wordmark carries a modest neon/pastel color wave. It uses a shared,
deterministic time phase in the menu and detached window and becomes a static
neon gradient when Reduce Motion is enabled.

The menu-bar item keeps a colored beacon-light glyph visible in every state.
An in-progress lane count appears beside it in a separate gold-to-coral badge,
so the app remains recognizable without burying the count among other status
items.

Beacon defaults to a 12-point system monospaced design. Settings provides
System, Rounded, Monospaced, and Serif designs plus 11, 12, 13, 14, and 16-point
base sizes; both surfaces share the persisted choice. Lane notation appears as
compact tag chips: use **Tag** to add context and the chip's close control to remove it.
Evidence badges such as **Dirty**, **CI None**, and **Review None** also reveal
a trailing close control on hover. Hiding a badge is local presentation state:
it does not change the underlying evidence or next action, and a changed signal
appears again. Use **Restore Hidden Badges** in Settings to clear all dismissals.

The whimsical **Signal Notes** panel sits at the bottom of both surfaces and is
expanded by default to roughly half the surface, while a manual collapse choice
persists. One live editor applies headings, emphasis, lists, quotes, inline code,
links, and dividers as Markdown is entered while retaining the exact plain-text
source. It autosaves three seconds after the latest edit, and Save and Revert
remain available for immediate control. All writes travel through the Go agent
authority so the menu, detached window, and `beacon notes` stay synchronized.

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

`scan --json` emits schema version 3 with projects, ordered lanes, issues,
checks, feedback, optional Kit progress, lane attention and tags, scoped
warnings, and scoped errors. It
never emits ANSI or additional stdout logging, making it safe for the macOS app
and automation.

Common workflows:

- Run `beacon` for the colorful project dashboard.
- Run `beacon --include-idle` when auditing idle work inside Following.
- Run `beacon select` to curate Following interactively.
- Run `beacon projects --recent` to inspect outside activity.
- Run `beacon projects --quiet` to inspect the remaining discovered inventory.
- Run `beacon refresh [project]` to request background work without blocking.
- Run `beacon sync` after merged pull requests to select safe local updates.
- Run `beacon sync check --no-fetch` for a network-free stale-branch check.
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
~/.local/share/beacon/notes.md
~/.cache/beacon/projects/*.json
~/.cache/beacon/agent.sock
~/.cache/beacon/agent.pid
~/Library/Logs/Beacon/agent.log
~/Library/Logs/Beacon/agent-error.log
```

One bounded worker pool scans followed projects independently. Projects outside
Following receive lightweight probes at the slower configured interval; Beacon
records material deltas as recent activity without changing membership.
Duplicate refreshes
coalesce, and a project never has overlapping jobs.

## Read-only boundary

Scanning may run a timeout-bounded `git fetch --prune --no-tags` to refresh
remote-tracking metadata. Beacon never edits working files, changes branches,
pushes commits, creates pull requests, changes reviews, or merges work. Beacon
writes only its own configuration during confirmed `beacon init` operations and
its own user-scoped following state, Markdown signal notes, cache, PID/socket,
LaunchAgent, and rotated logs. New evidence may update a non-followed project's activity timestamp but
never changes whether the user follows it.

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
following choices. To reset only cached evidence, stop the agent, remove
`~/.cache/beacon/projects/`, and start it again with `beacon agent install`;
do not remove `tracking.json` unless you intentionally want every discovered
project to restart in Quiet and rebuild Following manually.

## Maintainers

Maintained with 🪖 and ❤️ by [Jameson](https://github.com/jamesonstone) (`jamesonstone`).
