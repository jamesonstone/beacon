.PHONY: build test test-race vet fmt fmt-check install scan scan-json doctor macos-build macos-test macos-run

BEACON_DERIVED_DATA ?= $(TMPDIR)beacon-derived-data

build:
	go build -o bin/beacon ./cmd/beacon

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal

fmt-check:
	test -z "$$(gofmt -l cmd internal)"

install:
	go install ./cmd/beacon

scan:
	go run ./cmd/beacon scan

scan-json:
	go run ./cmd/beacon scan --json

doctor:
	go run ./cmd/beacon doctor

macos-build:
	xcodebuild -project macos/Beacon/Beacon.xcodeproj -scheme Beacon -configuration Debug -derivedDataPath "$(BEACON_DERIVED_DATA)" CODE_SIGNING_ALLOWED=NO build

macos-test:
	xcodebuild -project macos/Beacon/Beacon.xcodeproj -scheme Beacon -configuration Debug -destination 'platform=macOS' -derivedDataPath "$(BEACON_DERIVED_DATA)" CODE_SIGNING_ALLOWED=NO test

macos-run: macos-build
	open -n "$(BEACON_DERIVED_DATA)/Build/Products/Debug/Beacon.app"
