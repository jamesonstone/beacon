---
kit_metadata_version: 1
artifact: spec
workflow_version: 2
phase: deliver
delivery_intent: issue_branch_pr_in_progress
clarification:
  status: ready
  confidence: 100
  unresolved_questions: 0
feature:
  id: "0003"
  slug: beacon-github-releases
  dir: 0003-beacon-github-releases
relationships:
  - type: builds_on
    target: 0002-beacon-init-dashboard
references:
  - id: issue-1
    name: Implement Beacon work-lane review radar
    type: github-issue
    target: https://github.com/jamesonstone/beacon/issues/1
    relation: supports
    read_policy: conditional
    used_for: existing delivery lane selected for the first releasable merge
    status: active
  - id: github-release-notes
    name: GitHub automatically generated release notes
    type: web
    target: https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes
    relation: informs
    read_policy: conditional
    used_for: release-note generation
    status: active
  - id: constitution
    name: Beacon constitution
    type: doc
    target: docs/CONSTITUTION.md
    relation: constrains
    read_policy: must
    used_for: packaging, versioning, and distribution boundaries
    status: active
skills: []
---

# Beacon GitHub Releases

## Thesis

Every accepted merge to `main` should produce one traceable semantic version of Beacon, with the CLI and macOS menu app built from the same commit, carrying the same version, and downloadable from a GitHub release with generated notes and checksums.

## Context

Beacon currently builds locally and in pull-request CI but does not publish installable artifacts. The Go CLI already exposes link-time version variables, while the macOS application has static bundle versions and embeds an unversioned helper. This feature adds deterministic SemVer calculation, reproducible packaging, synchronized version injection, GitHub release publication, and user-facing installation and configuration documentation.

The user requested that this ship with the current first-release work. Because the menu application is not yet present on `main`, delivery continues on issue `#1`, branch `GH-1`, and ready PR `#2`; no release is created before that PR is merged by a human.

## Clarifications

- A push to `main` is the repository's merge-to-main release event.
- The CLI and macOS application share one SemVer because they ship from one source commit and the app bundles that CLI.
- Conventional commits since the latest strict `vMAJOR.MINOR.PATCH` tag determine the bump: breaking change or `!` is major, `feat` is minor, and every other change is patch.
- With no prior release, breaking work starts at `v1.0.0`, feature work at `v0.1.0`, and other work at `v0.0.1`.
- Rerunning a release for an already tagged commit reuses that version and replaces assets only while GitHub permits it; it never invents another version for the same commit.
- CLI archives target macOS and Linux on `amd64` and `arm64`.
- The macOS archive contains one universal application and universal embedded helper, both versioned from the release.
- The app is ad-hoc signed for bundle integrity but is not Developer ID signed or notarized. Gatekeeper guidance must remain explicit.
- GitHub-generated notes describe the release; a checksum manifest covers every uploaded binary archive.
- Release automation never merges, commits back to `main`, changes branches, or modifies user configuration.

## Requirements

1. Trigger release automation only for pushes to `main`, serialize release jobs, and grant only `contents: write`.
2. Calculate and test the next strict SemVer from conventional commit history without a third-party release service.
3. Build version-injected CLI archives for macOS and Linux on `amd64` and `arm64`.
4. Build a Release-configuration universal macOS app whose bundle version and embedded CLI report the release version and commit.
5. Ad-hoc sign and verify the macOS bundle, create a zip with macOS metadata preserved, and generate SHA-256 checksums.
6. Create or safely resume a GitHub release at the calculated tag with generated release notes and all artifacts.
7. Update the README with release installation, first-run initialization, multi-source configuration, everyday usage, upgrades, and unsigned-app guidance.
8. Update the user-authorized default config non-destructively to version 2 with the five requested persistent source roots.

## Assumptions

- Hosted `macos-14` runners retain the Xcode and architecture support already used by Beacon CI.
- GitHub's provided token may create releases and tags when the workflow grants `contents: write`.
- The repository continues using Conventional Commit PR and commit titles.
- GitHub releases are public and do not require a package registry or installer.
- User-specific configuration is validated locally but is never committed to the repository.

## Acceptance Criteria

- [x] AC1: The workflow runs on `main` pushes only, serializes executions, and has no permission broader than `contents: write`.
- [x] AC2: Version tests cover initial patch/minor/major releases, subsequent bumps, breaking bodies, exact-tag reuse, and invalid tag exclusion.
- [x] AC3: Every CLI archive receives the calculated version, commit, and build date, and the native host archive reports those values through `beacon version`.
- [x] AC4: The universal macOS app and helper contain both architectures, expose the calculated version, and pass bundle/code-sign verification.
- [ ] AC5: One GitHub release receives generated notes, four CLI archives, one macOS app zip, and a checksum manifest.
- [ ] AC6: A rerun for the same tagged commit is idempotent and does not calculate or publish a second version.
- [x] AC7: README instructions take a new user from release download through initialization, scanning, and opening the menu app, including Gatekeeper limitations.
- [x] AC8: The personal config contains exactly the requested persistent source roots in valid version-2 form while preserving existing settings and explicit repository metadata.
- [x] AC9: Go, race, version-script, macOS, workflow-lint, config-validation, and Kit project checks pass.

## Implementation Plan

1. Add the release contract to the constitution and this specification.
2. Implement a portable, tested SemVer script and version-aware macOS helper build.
3. Add the serialized, least-privilege release workflow and artifact verification.
4. Rewrite README installation, onboarding, source configuration, daily-use, and upgrade guidance around GitHub releases.
5. Migrate the personal config with Beacon's own merge path where possible, then validate it without scanning or rewriting repositories.
6. Run the complete validation map, record evidence, and update the existing ready PR.

## Task Checklist

- [x] T1: Implement and test SemVer calculation.
- [x] T2: Inject release metadata into CLI and macOS builds.
- [x] T3: Build and validate release archives and checksums.
- [x] T4: Publish through a `main`-push GitHub Actions workflow with generated notes.
- [x] T5: Update README and durable release contracts.
- [x] T6: Migrate and validate the personal configuration.
- [ ] T7: Complete local, workflow, project, and delivery validation; observe the first release after human merge.

## Validation Map

| Criterion | Validation |
| --- | --- |
| AC1, AC5, AC6 | `actionlint`, workflow inspection, and release-command dry-path inspection |
| AC2 | Temporary-repository shell tests for every bump and reuse case |
| AC3 | Cross-build inventory plus executable host-architecture version smoke test |
| AC4 | Release Xcode build, `lipo -info`, `plutil`, embedded-helper version, and `codesign --verify` |
| AC7 | README command and path review against current CLI help and config schema |
| AC8 | `beacon config validate` against the default personal config and exact source comparison |
| AC9 | `make fmt-check vet test test-race release-test build macos-test`, `actionlint`, `git diff --check`, and `kit check --all --project` |

## Reflection Notes

- A single macOS job can cross-compile the pure-Go CLI and build the only platform-specific client, keeping all artifacts on one runner and eliminating artifact handoff or version drift.
- Repository-local version calculation avoids a release-service dependency while remaining deterministic under the existing Conventional Commit contract.
- Workflow-level serialization is necessary so a second rapid merge observes the tag created for the first merge before calculating its own version.
- Ad-hoc signing verifies internal bundle consistency but does not provide Developer ID trust; README quarantine guidance therefore follows checksum verification and remains explicit.
- User glob notation is represented as canonical directory roots because Beacon performs recursive discovery itself and strict configuration does not expand shell wildcards.

## Documentation Updates

- Update `docs/CONSTITUTION.md` with synchronized versioning, GitHub release, and ad-hoc-signing boundaries.
- Update `docs/PROJECT_PROGRESS_SUMMARY.md` with feature `0003`.
- Update `README.md` around release-first installation and multi-source onboarding.

## Delivery Decision

Continue on issue `#1`, branch `GH-1`, and ready PR `#2` so the first merge containing the menu app also contains its release path. Do not merge the PR or create a release manually; the workflow becomes active only after a human merges to `main`.

## Evidence

- `shellcheck scripts/*.sh` passed for the SemVer, test, helper, and packaging scripts.
- `actionlint v1.7.12` passed for CI and release workflows; release actions are pinned to the full commits for `actions/checkout v6.0.2` and `actions/setup-go v6.4.0`.
- `make fmt-check`, `make vet`, `make test`, `make test-race`, `make release-test`, and `make build` passed.
- `make macos-test` passed all 14 Swift tests for both target architectures; Xcode's non-fatal local `linkd.autoShortcut` host messages remain unrelated to the suite result.
- A release dry run produced four CLI archives, a 4.6 MB universal app zip, and `checksums.txt`; `shasum -a 256 -c` verified all five archives.
- The native CLI archive and embedded helper reported `beacon 0.1.0` with the injected commit and UTC date.
- `lipo` confirmed `arm64` and `x86_64` in the application and embedded helper, bundle versions were `0.1.0` and build `1`, and `codesign --verify --deep --strict` passed.
- `beacon init` atomically migrated `/Users/jamesonstone/.config/beacon/config.yaml` to version 2 with the five requested sources, preserved the explicit Beacon repository, and reported 79 accessible discoveries. `beacon config validate` passed; `jamesonstone/sigint` remains a scoped warning because GitHub cannot resolve that repository.
- `git diff --check` and `kit check --all --project` passed.
- AC5, AC6, and the final part of T7 intentionally remain pending until a human merges PR #2 and the first live release workflow completes; no release was created during implementation.
