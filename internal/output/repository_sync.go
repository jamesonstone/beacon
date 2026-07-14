package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/jamesonstone/beacon/internal/reposync"
)

func RepositorySyncJSON(writer io.Writer, report reposync.Report) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(report)
}

func RepositorySync(writer io.Writer, report reposync.Report, options TerminalOptions) error {
	style := terminalStyles(writer, options.Color)
	attention, safe := 0, 0
	for _, repository := range report.Repositories {
		if repository.NeedsUpdate {
			attention++
		}
		if repository.CanUpdate {
			safe++
		}
	}
	header := fmt.Sprintf("Repository Sync  %d need attention · %d safe to update", attention, safe)
	if report.FetchAttempt {
		header += " · remote refs checked"
	} else {
		header += " · local refs only"
	}
	if _, err := fmt.Fprintf(writer, "%s\n\n", style.heading.Render(header)); err != nil {
		return err
	}
	if len(report.Repositories) == 0 {
		_, err := fmt.Fprintln(writer, style.dim.Render("No configured repositories were found."))
		return err
	}
	var buffer bytes.Buffer
	table := tabwriter.NewWriter(&buffer, 2, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "PROJECT\tBRANCH\tDEFAULT\tSTATE\tACTION"); err != nil {
		return err
	}
	for _, repository := range report.Repositories {
		branch := repository.CurrentBranch
		if branch == "" {
			branch = "detached"
		}
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s/%s\t%s\t%s\n",
			repository.ProjectID, branch, repository.Remote, repository.Base,
			repository.State, syncActionLabel(repository)); err != nil {
			return err
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}
	if _, err := io.Copy(writer, &buffer); err != nil {
		return err
	}
	for _, repository := range report.Repositories {
		if repository.State == reposync.StateCurrent || repository.State == reposync.StateAhead {
			continue
		}
		message := "  " + repository.ProjectID + ": " + repository.Reason
		if repository.Error != "" {
			message += " (" + repository.Error + ")"
		}
		if _, err := fmt.Fprintln(writer, style.dim.Render(message)); err != nil {
			return err
		}
	}
	return nil
}

func syncActionLabel(repository reposync.Repository) string {
	if repository.Updated {
		return "updated"
	}
	if !repository.CanUpdate {
		if repository.NeedsUpdate || repository.State == reposync.StateBlocked || repository.State == reposync.StateDiverged {
			return "manual"
		}
		return "none"
	}
	return strings.ReplaceAll(string(repository.Action), "_", " ")
}
