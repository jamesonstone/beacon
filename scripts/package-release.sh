#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd "$script_dir/.." && pwd)"
version="${BEACON_VERSION:?BEACON_VERSION is required}"
commit="${BEACON_COMMIT:?BEACON_COMMIT is required}"
build_date="${BEACON_BUILD_DATE:?BEACON_BUILD_DATE is required}"
build_number="${BEACON_BUILD_NUMBER:-1}"
output_directory="${BEACON_OUTPUT_DIR:-$repository_root/dist}"
derived_data="${BEACON_DERIVED_DATA:-${TMPDIR:-/tmp}/beacon-release-derived-data}"
staging_directory="$(mktemp -d "${TMPDIR:-/tmp}/beacon-release-stage.XXXXXX")"
trap 'rm -rf "$staging_directory"' EXIT

if [[ "$(uname -s)" != Darwin ]]; then
  printf 'macOS is required to package the Beacon application\n' >&2
  exit 1
fi

if [[ -e "$output_directory" && ! -d "$output_directory" ]]; then
  printf 'release output exists and is not a directory: %s\n' "$output_directory" >&2
  exit 1
fi
if [[ -d "$output_directory" ]] &&
  [[ -n "$(find "$output_directory" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
  printf 'release output directory must be empty: %s\n' "$output_directory" >&2
  exit 1
fi
mkdir -p "$output_directory"

ldflags="-s -w -X github.com/jamesonstone/beacon/internal/cli.Version=$version -X github.com/jamesonstone/beacon/internal/cli.Commit=$commit -X github.com/jamesonstone/beacon/internal/cli.Date=$build_date"

for platform in darwin/arm64 darwin/amd64 linux/arm64 linux/amd64; do
  os="${platform%/*}"
  architecture="${platform#*/}"
  archive_name="beacon_${version}_${os}_${architecture}"
  archive_directory="$staging_directory/$archive_name"
  mkdir -p "$archive_directory"
  (
    cd "$repository_root"
    CGO_ENABLED=0 GOOS="$os" GOARCH="$architecture" \
      go build -trimpath -ldflags "$ldflags" -o "$archive_directory/beacon" ./cmd/beacon
    CGO_ENABLED=0 GOOS="$os" GOARCH="$architecture" \
      go build -trimpath -ldflags "$ldflags" -o "$archive_directory/bctl" ./cmd/bctl
  )
  tar -C "$staging_directory" -czf "$output_directory/$archive_name.tar.gz" "$archive_name"
done

BEACON_VERSION="$version" \
BEACON_COMMIT="$commit" \
BEACON_BUILD_DATE="$build_date" \
xcodebuild \
  -project "$repository_root/macos/Beacon/Beacon.xcodeproj" \
  -scheme Beacon \
  -configuration Release \
  -destination 'generic/platform=macOS' \
  -derivedDataPath "$derived_data" \
  CODE_SIGNING_ALLOWED=NO \
  MARKETING_VERSION="$version" \
  CURRENT_PROJECT_VERSION="$build_number" \
  BEACON_VERSION="$version" \
  BEACON_COMMIT="$commit" \
  BEACON_BUILD_DATE="$build_date" \
  build

xcodebuild \
  -project "$repository_root/macos/Beacon/Beacon.xcodeproj" \
  -scheme Hyperlite \
  -configuration Release \
  -destination 'generic/platform=macOS' \
  -derivedDataPath "$derived_data" \
  CODE_SIGNING_ALLOWED=NO \
  MARKETING_VERSION="$version" \
  CURRENT_PROJECT_VERSION="$build_number" \
  BEACON_VERSION="$version" \
  BEACON_COMMIT="$commit" \
  BEACON_BUILD_DATE="$build_date" \
  build

application="$derived_data/Build/Products/Release/Beacon.app"
helper="$application/Contents/MacOS/beacon-cli"
login_item="$application/Contents/Library/LoginItems/BeaconLoginItem.app"
info_plist="$application/Contents/Info.plist"

[[ -d "$application" ]]
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$info_plist")" == "$version" ]]
[[ "$(/usr/bin/lipo -archs "$application/Contents/MacOS/Beacon")" == *arm64* ]]
[[ "$(/usr/bin/lipo -archs "$application/Contents/MacOS/Beacon")" == *x86_64* ]]
[[ "$(/usr/bin/lipo -archs "$helper")" == *arm64* ]]
[[ "$(/usr/bin/lipo -archs "$helper")" == *x86_64* ]]
[[ -x "$login_item/Contents/MacOS/BeaconLoginItem" ]]
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$login_item/Contents/Info.plist")" == "com.jamesonstone.beacon.login-item" ]]
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIconFile' "$info_plist")" == "AppIcon" ]]
if /usr/libexec/PlistBuddy -c 'Print :LSUIElement' "$info_plist" >/dev/null 2>&1; then
  printf 'Beacon.app must not set LSUIElement\n' >&2
  exit 1
fi
"$helper" version | grep -F "beacon $version ($commit, $build_date)"

/usr/bin/codesign --force --deep --sign - --timestamp=none "$application"
/usr/bin/codesign --verify --deep --strict "$application"

/usr/bin/ditto -c -k --sequesterRsrc --keepParent \
  "$application" \
  "$output_directory/Beacon_${version}_macos_universal.zip"

hyperlite_application="$derived_data/Build/Products/Release/Hyperlite.app"
hyperlite_helper="$hyperlite_application/Contents/MacOS/beacon-cli"
hyperlite_bctl="$hyperlite_application/Contents/MacOS/bctl"
hyperlite_info_plist="$hyperlite_application/Contents/Info.plist"
[[ -d "$hyperlite_application" ]]
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$hyperlite_info_plist")" == "com.jamesonstone.beacon.hyperlite" ]]
[[ "$(/usr/libexec/PlistBuddy -c 'Print :LSUIElement' "$hyperlite_info_plist")" == "true" ]]
[[ -x "$hyperlite_helper" ]]
[[ -x "$hyperlite_bctl" ]]
[[ "$(/usr/bin/lipo -archs "$hyperlite_helper")" == *arm64* ]]
[[ "$(/usr/bin/lipo -archs "$hyperlite_helper")" == *x86_64* ]]
[[ "$(/usr/bin/lipo -archs "$hyperlite_bctl")" == *arm64* ]]
[[ "$(/usr/bin/lipo -archs "$hyperlite_bctl")" == *x86_64* ]]
/usr/bin/codesign --force --deep --sign - --timestamp=none "$hyperlite_application"
/usr/bin/codesign --verify --deep --strict "$hyperlite_application"
/usr/bin/ditto -c -k --sequesterRsrc --keepParent \
  "$hyperlite_application" \
  "$output_directory/Hyperlite_${version}_macos_universal.zip"

(
  cd "$output_directory"
  for artifact in ./*.tar.gz ./*.zip; do
    shasum -a 256 "${artifact#./}"
  done
) >"$output_directory/checksums.txt"

printf 'release artifacts written to %s\n' "$output_directory"
