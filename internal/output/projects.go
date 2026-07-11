package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/jamesonstone/beacon/internal/model"
)

func Projects(writer io.Writer, snapshot model.Snapshot, state model.TrackingState, options TerminalOptions) error {
	if options.Width <= 0 {
		options.Width = 120
	}
	style := terminalStyles(writer, options.Color)
	title := "Tracked Projects"
	if state == model.TrackingUntracked {
		title = "Untracked Projects"
	}
	projects := make([]model.Project, 0)
	for _, project := range snapshot.Projects {
		if project.TrackingState == state {
			projects = append(projects, project)
		}
	}
	countLabel := fmt.Sprintf("%d project%s", len(projects), pluralSuffix(len(projects)))
	if _, err := fmt.Fprintf(writer, "%s  %s\n\n", style.heading.Render(title), style.dim.Render(countLabel)); err != nil {
		return err
	}
	if len(projects) == 0 {
		_, err := fmt.Fprintln(writer, style.dim.Render("No projects in this view."))
		return err
	}
	if options.Width < narrowWidth {
		for _, project := range projects {
			if err := writeWrapped(writer, "  ", style.wrap.Width(max(1, options.Width-2)).Render(style.project.Render(project.Name)+"  "+project.GitHub)); err != nil {
				return err
			}
			if err := writeWrapped(writer, "    ", style.wrap.Width(max(1, options.Width-4)).Render(style.dim.Render(project.Path))); err != nil {
				return err
			}
		}
		return nil
	}

	var buffer bytes.Buffer
	table := tabwriter.NewWriter(&buffer, 2, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "PROJECT\tGITHUB\tPATH"); err != nil {
		return err
	}
	for _, project := range projects {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\n", project.Name, project.GitHub, project.Path); err != nil {
			return err
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}
	lines := strings.SplitAfter(buffer.String(), "\n")
	for index, project := range projects {
		lineIndex := index + 1
		if lineIndex >= len(lines) {
			break
		}
		lines[lineIndex] = replaceFirst(lines[lineIndex], project.Name, style.project.Render(project.Name))
	}
	_, err := io.WriteString(writer, strings.Join(lines, ""))
	return err
}
