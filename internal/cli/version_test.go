package cli

import (
	"bytes"
	"context"
	"runtime/debug"
	"strings"
	"testing"
)

func TestResolveBuildDetailsUsesEmbeddedVCSMetadata(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.0.0-20260710135309-c2b174e5068a"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "c2b174e5068ae4e7d282b71ab72e6b8b213f232b"},
			{Key: "vcs.time", Value: "2026-07-10T13:53:09Z"},
			{Key: "vcs.modified", Value: "false"},
		},
	}

	details := resolveBuildDetails("dev", "unknown", "unknown", info)
	if details.version != "0.0.0-20260710135309-c2b174e5068a" ||
		details.commit != "c2b174e5068ae4e7d282b71ab72e6b8b213f232b" ||
		details.date != "2026-07-10T13:53:09Z" {
		t.Fatalf("details = %+v", details)
	}
}

func TestResolveBuildDetailsPreservesReleaseMetadata(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.0.0-20260710135309-c2b174e5068a"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "embedded"},
			{Key: "vcs.time", Value: "embedded"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	details := resolveBuildDetails("1.2.3", "release-commit", "release-date", info)
	if details != (buildDetails{version: "1.2.3", commit: "release-commit", date: "release-date"}) {
		t.Fatalf("details = %+v", details)
	}
}

func TestResolveBuildDetailsMarksDevelopmentBuildDirty(t *testing.T) {
	info := &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "c2b174e5068ae4e7d282b71ab72e6b8b213f232b"},
		{Key: "vcs.time", Value: "2026-07-10T13:53:09Z"},
		{Key: "vcs.modified", Value: "true"},
	}}

	details := resolveBuildDetails("dev", "unknown", "unknown", info)
	if details.version != "dev-c2b174e5068a-dirty" {
		t.Fatalf("version = %q", details.version)
	}
}

func TestResolveBuildDetailsDoesNotDuplicateEmbeddedDirtyMarker(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.0.0-20260710135309-c2b174e5068a+dirty"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.modified", Value: "true"},
		},
	}

	details := resolveBuildDetails("dev", "unknown", "unknown", info)
	if details.version != "0.0.0-20260710135309-c2b174e5068a+dirty" {
		t.Fatalf("version = %q", details.version)
	}
}

func TestVersionCommandUsesExecutableName(t *testing.T) {
	var output bytes.Buffer
	command := versionCommand(&output, "bctl")
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(output.String(), "bctl ") {
		t.Fatalf("output = %q", output.String())
	}
}
