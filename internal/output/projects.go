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
	title := "Following Projects"
	projects := make([]model.Project, 0)
	for _, project := range snapshot.Projects {
		if project.TrackingState == state {
			projects = append(projects, project)
		}
	}
	if state == model.TrackingUntracked {
		title = "Projects Not Followed"
	}
	return renderProjects(writer, projects, title, false, options)
}

func FollowingProjects(writer io.Writer, snapshot model.Snapshot, state model.FollowState, options TerminalOptions) error {
	title := map[model.FollowState]string{
		model.FollowFollowing: "Following Projects",
		model.FollowRecent:    "Recently Updated Projects",
		model.FollowQuiet:     "Quiet Projects",
	}[state]
	projects := make([]model.Project, 0)
	for _, project := range snapshot.Projects {
		if normalizedFollowState(project) == state {
			projects = append(projects, project)
		}
	}
	return renderProjects(writer, projects, title, state == model.FollowRecent, options)
}

func renderProjects(writer io.Writer, projects []model.Project, title string, showActivity bool, options TerminalOptions) error {
	if options.Width <= 0 {
		options.Width = 120
	}
	style := terminalStyles(writer, options.Color)
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
			if showActivity {
				activity := project.ActivityReason + " · " + activityTime(project)
				if err := writeWrapped(writer, "    ", style.wrap.Width(max(1, options.Width-4)).Render(style.dim.Render(activity))); err != nil {
					return err
				}
			}
		}
		return nil
	}

	var buffer bytes.Buffer
	table := tabwriter.NewWriter(&buffer, 2, 4, 2, ' ', 0)
	header := "PROJECT\tGITHUB\tPATH"
	if showActivity {
		header += "\tACTIVITY\tUPDATED"
	}
	if _, err := fmt.Fprintln(table, header); err != nil {
		return err
	}
	for _, project := range projects {
		row := fmt.Sprintf("%s\t%s\t%s", project.Name, project.GitHub, project.Path)
		if showActivity {
			row += fmt.Sprintf("\t%s\t%s", project.ActivityReason, activityTime(project))
		}
		if _, err := fmt.Fprintln(table, row); err != nil {
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

func activityTime(project model.Project) string {
	if project.LastActivityAt.IsZero() {
		return "unknown"
	}
	return project.LastActivityAt.Format("2006-01-02 15:04Z07:00")
}

func normalizedFollowState(project model.Project) model.FollowState {
	if project.FollowState != "" {
		return project.FollowState
	}
	if project.TrackingState == model.TrackingUntracked {
		return model.FollowQuiet
	}
	return model.FollowFollowing
}
