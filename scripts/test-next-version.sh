#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
version_script="$script_dir/next-version.sh"
workspace="$(mktemp -d "${TMPDIR:-/tmp}/beacon-version-test.XXXXXX")"
trap 'rm -rf "$workspace"' EXIT

new_repository() {
  local name="$1"
  local repository="$workspace/$name"
  mkdir -p "$repository"
  git -C "$repository" init -q -b main
  git -C "$repository" config user.name 'Beacon Release Test'
  git -C "$repository" config user.email 'release-test@example.invalid'
  printf '%s\n' "$repository"
}

commit_change() {
  local repository="$1"
  local subject="$2"
  local body="${3:-}"
  printf '%s\n' "$subject" >>"$repository/history.txt"
  git -C "$repository" add history.txt
  if [[ -n "$body" ]]; then
    git -C "$repository" commit -q -m "$subject" -m "$body"
  else
    git -C "$repository" commit -q -m "$subject"
  fi
}

assert_version() {
  local repository="$1"
  local expected="$2"
  local actual
  actual="$(cd "$repository" && "$version_script")"
  if [[ "$actual" != "$expected" ]]; then
    printf 'expected %s, got %s in %s\n' "$expected" "$actual" "$repository" >&2
    exit 1
  fi
}

patch_repository="$(new_repository patch)"
commit_change "$patch_repository" 'docs: add usage guide'
assert_version "$patch_repository" 'v0.0.1'
git -C "$patch_repository" tag v0.0.1
commit_change "$patch_repository" 'fix: correct scan output'
assert_version "$patch_repository" 'v0.0.2'

feature_repository="$(new_repository feature)"
commit_change "$feature_repository" 'feat(GH-1): add dashboard'
assert_version "$feature_repository" 'v0.1.0'
git -C "$feature_repository" tag v0.1.0
commit_change "$feature_repository" 'feat: add release downloads'
assert_version "$feature_repository" 'v0.2.0'

breaking_repository="$(new_repository breaking)"
commit_change "$breaking_repository" 'feat: establish public contract' 'BREAKING CHANGE: replace the previous schema'
assert_version "$breaking_repository" 'v1.0.0'
git -C "$breaking_repository" tag v1.0.0
assert_version "$breaking_repository" 'v1.0.0'

bang_repository="$(new_repository bang)"
commit_change "$bang_repository" 'refactor!: replace the command contract'
assert_version "$bang_repository" 'v1.0.0'

invalid_repository="$(new_repository invalid-tag)"
commit_change "$invalid_repository" 'chore: start repository'
git -C "$invalid_repository" tag v999-invalid
assert_version "$invalid_repository" 'v0.0.1'

printf 'semantic version tests passed\n'
