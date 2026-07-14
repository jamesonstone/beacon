package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/jamesonstone/beacon/internal/githubapi"
)

func DependencyLimitsJSON(writer io.Writer, report githubapi.RateLimitReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(report)
}

func DependencyLimits(writer io.Writer, report githubapi.RateLimitReport, options TerminalOptions) error {
	style := terminalStyles(writer, options.Color)
	if _, err := fmt.Fprintf(writer, "%s\n\n", style.heading.Render("Dependency Limits")); err != nil {
		return err
	}
	if len(report.Dependencies) == 0 {
		_, err := fmt.Fprintln(writer, style.dim.Render("No rate-limited dependencies were reported."))
		return err
	}

	for index, dependency := range report.Dependencies {
		if index > 0 {
			if _, err := fmt.Fprintln(writer); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(writer, "%s\n", style.heading.Render(dependency.Name)); err != nil {
			return err
		}
		var buffer bytes.Buffer
		table := tabwriter.NewWriter(&buffer, 2, 4, 2, ' ', 0)
		if _, err := fmt.Fprintln(table, "BUCKET\tUSED\tLIMIT\tREMAINING\tRESET"); err != nil {
			return err
		}
		for _, bucket := range dependency.Buckets {
			reset := "unknown"
			if !bucket.ResetAt.IsZero() {
				reset = bucket.ResetAt.Local().Format("15:04:05")
			}
			if _, err := fmt.Fprintf(table, "%s\t%d\t%d\t%d\t%s\n",
				bucket.Name, bucket.Used, bucket.Limit, bucket.Remaining, reset); err != nil {
				return err
			}
		}
		if err := table.Flush(); err != nil {
			return err
		}
		if _, err := io.Copy(writer, &buffer); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(writer, "\n%s\n", style.dim.Render("Checked explicitly at "+report.CheckedAt.Local().Format("15:04:05")))
	return err
}
