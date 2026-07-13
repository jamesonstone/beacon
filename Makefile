.PHONY: build test test-race release-test vet fmt fmt-check install scan scan-json doctor agent-install agent-status agent-stop agent-uninstall macos-build macos-test macos-run

BEACON_DERIVED_DATA ?= $(TMPDIR)beacon-derived-data

build:
	go build -o bin/beacon ./cmd/beacon

test:
	go test ./...

test-race:
	go test -race ./...

release-test:
	./scripts/test-next-version.sh

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

agent-install:
	go run ./cmd/beacon agent install

agent-status:
	go run ./cmd/beacon agent status

agent-stop:
	go run ./cmd/beacon agent stop

agent-uninstall:
	go run ./cmd/beacon agent uninstall

macos-build:
	xcodebuild -project macos/Beacon/Beacon.xcodeproj -scheme Beacon -configuration Debug -derivedDataPath "$(BEACON_DERIVED_DATA)" CODE_SIGNING_ALLOWED=NO build

macos-test:
	xcodebuild -project macos/Beacon/Beacon.xcodeproj -scheme Beacon -configuration Debug -destination 'platform=macOS' -derivedDataPath "$(BEACON_DERIVED_DATA)" CODE_SIGNING_ALLOWED=NO test

macos-run: macos-build
	open -n "$(BEACON_DERIVED_DATA)/Build/Products/Debug/Beacon.app"
