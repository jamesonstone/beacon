.PHONY: build test test-race release-test vet fmt fmt-check install scan scan-json doctor agent-install agent-status agent-stop agent-uninstall macos-build macos-test macos-run macos-hyperlite-run hyper

BEACON_DERIVED_DATA ?= $(TMPDIR)beacon-derived-data

build:
	go build -o bin/beacon ./cmd/beacon
	go build -o bin/bctl ./cmd/bctl

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
	go install ./cmd/beacon ./cmd/bctl

scan:
	go run ./cmd/bctl

scan-json:
	go run ./cmd/bctl --json

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
	xcodebuild -project macos/Beacon/Beacon.xcodeproj -scheme Beacon -configuration Debug -destination 'generic/platform=macOS' -derivedDataPath "$(BEACON_DERIVED_DATA)" CODE_SIGNING_ALLOWED=NO build
	xcodebuild -project macos/Beacon/Beacon.xcodeproj -scheme Hyperlite -configuration Debug -destination 'generic/platform=macOS' -derivedDataPath "$(BEACON_DERIVED_DATA)" CODE_SIGNING_ALLOWED=NO build

macos-test:
	xcodebuild -project macos/Beacon/Beacon.xcodeproj -scheme Beacon -configuration Debug -destination 'platform=macOS' -derivedDataPath "$(BEACON_DERIVED_DATA)" CODE_SIGNING_ALLOWED=NO ONLY_ACTIVE_ARCH=YES test

macos-run: macos-build
	@if pgrep -x Beacon >/dev/null; then \
		osascript -e 'tell application id "com.jamesonstone.beacon" to quit' >/dev/null 2>&1 || true; \
		attempt=0; \
		while pgrep -x Beacon >/dev/null && [ "$$attempt" -lt 50 ]; do \
			sleep 0.1; \
			attempt=$$((attempt + 1)); \
		done; \
		if pgrep -x Beacon >/dev/null; then \
			echo "Beacon did not stop cleanly; forcing termination."; \
			pkill -TERM -x Beacon 2>/dev/null || true; \
		fi; \
		attempt=0; \
		while pgrep -x Beacon >/dev/null && [ "$$attempt" -lt 20 ]; do \
			sleep 0.1; \
			attempt=$$((attempt + 1)); \
		done; \
		if pgrep -x Beacon >/dev/null; then \
			echo "Beacon is still running; refusing to start another instance." >&2; \
			exit 1; \
		fi; \
	fi
	open "$(BEACON_DERIVED_DATA)/Build/Products/Debug/Beacon.app"

macos-hyperlite-run: macos-build
	@if pgrep -x Hyperlite >/dev/null; then \
		osascript -e 'tell application id "com.jamesonstone.beacon.hyperlite" to quit' >/dev/null 2>&1 || true; \
		attempt=0; \
		while pgrep -x Hyperlite >/dev/null && [ "$$attempt" -lt 50 ]; do \
			sleep 0.1; \
			attempt=$$((attempt + 1)); \
		done; \
		if pgrep -x Hyperlite >/dev/null; then \
			echo "Hyperlite did not stop cleanly; forcing termination."; \
			pkill -TERM -x Hyperlite 2>/dev/null || true; \
		fi; \
		attempt=0; \
		while pgrep -x Hyperlite >/dev/null && [ "$$attempt" -lt 20 ]; do \
			sleep 0.1; \
			attempt=$$((attempt + 1)); \
		done; \
		if pgrep -x Hyperlite >/dev/null; then \
			echo "Hyperlite is still running; refusing to start another instance." >&2; \
			exit 1; \
		fi; \
	fi
	open "$(BEACON_DERIVED_DATA)/Build/Products/Debug/Hyperlite.app"

hyper: macos-hyperlite-run
