#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 2 || $# -gt 3 ]]; then
	printf 'usage: %s <helper-directory> <destination> [bctl-destination]\n' "$0" >&2
  exit 2
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd "$script_dir/.." && pwd)"
helper_directory="$1"
destination="$2"
bctl_destination="${3:-}"
version="${BEACON_VERSION:-dev}"
commit="${BEACON_COMMIT:-unknown}"
build_date="${BEACON_BUILD_DATE:-unknown}"
ldflags="-s -w -X github.com/jamesonstone/beacon/internal/cli.Version=$version -X github.com/jamesonstone/beacon/internal/cli.Commit=$commit -X github.com/jamesonstone/beacon/internal/cli.Date=$build_date"

mkdir -p "$helper_directory" "$(dirname "$destination")"

for architecture in arm64 amd64; do
  output="$helper_directory/beacon-$architecture"
  (
    cd "$repository_root"
    CGO_ENABLED=0 GOOS=darwin GOARCH="$architecture" \
      go build -trimpath -ldflags "$ldflags" -o "$output" ./cmd/beacon
  )
done

/usr/bin/lipo -create \
  "$helper_directory/beacon-arm64" \
  "$helper_directory/beacon-amd64" \
  -output "$destination"

# Xcode validates nested executables while signing the containing app. Sign the
# universal helper first, using Xcode's selected identity when available and an
# ad-hoc identity for local developer builds.
signing_identity="${EXPANDED_CODE_SIGN_IDENTITY:--}"
/usr/bin/codesign --force --sign "$signing_identity" --timestamp=none "$destination"

if [[ -n "$bctl_destination" ]]; then
	for architecture in arm64 amd64; do
		output="$helper_directory/bctl-$architecture"
		(
			cd "$repository_root"
			CGO_ENABLED=0 GOOS=darwin GOARCH="$architecture" \
				go build -trimpath -ldflags "$ldflags" -o "$output" ./cmd/bctl
		)
	done
	/usr/bin/lipo -create \
		"$helper_directory/bctl-arm64" \
		"$helper_directory/bctl-amd64" \
		-output "$bctl_destination"
	/usr/bin/codesign --force --sign "$signing_identity" --timestamp=none "$bctl_destination"
fi
