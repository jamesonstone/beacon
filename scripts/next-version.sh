#!/usr/bin/env bash

set -euo pipefail

strict_tags() {
  git tag --list 'v*' --sort=-version:refname |
    while IFS= read -r tag; do
      if [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        printf '%s\n' "$tag"
      fi
    done
}

tag_at_head="$({
  git tag --points-at HEAD --list 'v*' --sort=-version:refname |
    while IFS= read -r tag; do
      if [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        printf '%s\n' "$tag"
        break
      fi
    done
} || true)"

if [[ -n "$tag_at_head" ]]; then
  printf '%s\n' "$tag_at_head"
  exit 0
fi

latest_tag="$(strict_tags | head -n 1 || true)"
if [[ -n "$latest_tag" ]]; then
  range="$latest_tag..HEAD"
  version="${latest_tag#v}"
  IFS=. read -r major minor patch <<<"$version"
else
  range="HEAD"
  major=0
  minor=0
  patch=0
fi

messages="$(git log --format='%s%n%b' "$range")"
if [[ -z "$messages" ]]; then
  printf 'no commits found after %s\n' "${latest_tag:-repository start}" >&2
  exit 1
fi

if printf '%s\n' "$messages" | grep -Eq '(^|[[:space:]])BREAKING([ -])CHANGE:' ||
  printf '%s\n' "$messages" | grep -Eq '^[[:alpha:]][[:alnum:]_-]*(\([^)]*\))?!:'; then
  major=$((major + 1))
  minor=0
  patch=0
elif printf '%s\n' "$messages" | grep -Eq '^feat(\([^)]*\))?:'; then
  minor=$((minor + 1))
  patch=0
else
  patch=$((patch + 1))
fi

printf 'v%d.%d.%d\n' "$major" "$minor" "$patch"
