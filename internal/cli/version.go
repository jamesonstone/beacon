package cli

import (
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

func versionCommand(writer io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			info, _ := debug.ReadBuildInfo()
			details := resolveBuildDetails(Version, Commit, Date, info)
			_, err := fmt.Fprintf(writer, "beacon %s (%s, %s)\n", details.version, details.commit, details.date)
			return err
		},
	}
}

type buildDetails struct {
	version string
	commit  string
	date    string
}

func resolveBuildDetails(version, commit, date string, info *debug.BuildInfo) buildDetails {
	details := buildDetails{version: version, commit: commit, date: date}
	if info == nil {
		return details
	}

	useEmbeddedVersion := details.version == "dev"
	if useEmbeddedVersion && info.Main.Version != "" && info.Main.Version != "(devel)" {
		details.version = strings.TrimPrefix(info.Main.Version, "v")
	}

	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if details.commit == "unknown" && setting.Value != "" {
				details.commit = setting.Value
			}
		case "vcs.time":
			if details.date == "unknown" && setting.Value != "" {
				details.date = setting.Value
			}
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if useEmbeddedVersion && details.version == "dev" && details.commit != "unknown" {
		details.version = "dev-" + shortRevision(details.commit)
	}
	if useEmbeddedVersion && modified && !strings.Contains(details.version, "dirty") {
		details.version += "-dirty"
	}
	return details
}

func shortRevision(revision string) string {
	const length = 12
	if len(revision) <= length {
		return revision
	}
	return revision[:length]
}
