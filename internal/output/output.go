package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/jamesonstone/beacon/internal/model"
)

const narrowWidth = 100

type TerminalOptions struct {
	Color         bool
	Width         int
	IncludeIdle   bool
	WorkingSet    bool
	IncludeParked bool
}

type styles struct {
	project lipgloss.Style
	green   lipgloss.Style
	yellow  lipgloss.Style
	red     lipgloss.Style
	dim     lipgloss.Style
	heading lipgloss.Style
	wrap    lipgloss.Style
}

func JSON(writer io.Writer, snapshot model.Snapshot) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(snapshot)
}

func Terminal(writer io.Writer, snapshot model.Snapshot) error {
	return TerminalWithOptions(writer, snapshot, TerminalOptions{Width: 120})
}

func TerminalWithOptions(writer io.Writer, snapshot model.Snapshot, options TerminalOptions) error {
	if options.Width <= 0 {
		options.Width = 120
	}
	style := terminalStyles(writer, options.Color)
	if options.WorkingSet {
		return workingSetTerminal(writer, snapshot, options, style)
	}
	header := style.heading.Render("Beacon") + "  " + style.dim.Render(snapshot.GeneratedAt.Local().Format("2006-01-02 15:04:05 MST"))
	if options.Width < narrowWidth {
		if err := writeWrapped(writer, "", style.wrap.Width(options.Width).Render(header)); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintln(writer, header); err != nil {
		return err
	}
	summary := fmt.Sprintf("%d projects · %d work items · %d ready · %d issues · %d unresolved feedback",
		snapshot.Summary.TrackedProjects, snapshot.Summary.Total, snapshot.Summary.ReviewReady, snapshot.Summary.OpenIssues, snapshot.Summary.UnresolvedFeedback)
	visibleWarnings := visibleDiagnostics(snapshot, snapshot.Warnings)
	if len(visibleWarnings) > 0 {
		summary += fmt.Sprintf(" · %d warnings", len(visibleWarnings))
	}
	if options.Width < narrowWidth {
		if err := writeWrapped(writer, "", style.wrap.Width(options.Width).Render(summary)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintf(writer, "%s\n\n", summary); err != nil {
		return err
	}

	byID := make(map[string]model.Lane, len(snapshot.Lanes))
	for _, lane := range snapshot.Lanes {
		byID[lane.ID] = lane
	}
	idleIDs, idleProjects := idleFollowingInventory(snapshot)
	sections := []struct {
		title string
		ids   []string
	}{
		{"Ready for Review", snapshot.Groups.Ready},
		{"Needs Action", snapshot.Groups.Action},
		{"Waiting", snapshot.Groups.Waiting},
	}
	if options.IncludeIdle && len(idleIDs) > 0 {
		sections = append(sections, struct {
			title string
			ids   []string
		}{"Idle Following Projects", idleIDs})
	}
	for _, section := range sections {
		if len(section.ids) == 0 {
			continue
		}
		sectionTitle := style.heading.Render(section.title)
		if options.Width < narrowWidth {
			if err := writeWrapped(writer, "", style.wrap.Width(options.Width).Render(sectionTitle)); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintln(writer, sectionTitle); err != nil {
			return err
		}
		lanes := make([]model.Lane, 0, len(section.ids))
		for _, id := range section.ids {
			if lane, ok := byID[id]; ok {
				lanes = append(lanes, lane)
			}
		}
		if options.Width < narrowWidth {
			if err := renderNarrow(writer, lanes, style, options.Width); err != nil {
				return err
			}
		} else if err := renderTable(writer, lanes, style); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}
	if !options.IncludeIdle && idleProjects > 0 {
		message := style.dim.Render(fmt.Sprintf("%d idle following project%s hidden · use --include-idle to show", idleProjects, pluralSuffix(idleProjects)))
		if options.Width < narrowWidth {
			if err := writeWrapped(writer, "", style.wrap.Width(options.Width).Render(message)); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintln(writer, message); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}
	if snapshot.Summary.RecentProjects > 0 {
		message := style.dim.Render(fmt.Sprintf("%d recently updated project%s outside Following · run beacon projects --recent to view", snapshot.Summary.RecentProjects, pluralSuffix(snapshot.Summary.RecentProjects)))
		if options.Width < narrowWidth {
			if err := writeWrapped(writer, "", style.wrap.Width(options.Width).Render(message)); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintln(writer, message); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}
	if snapshot.Summary.QuietProjects > 0 {
		message := style.dim.Render(fmt.Sprintf("%d quiet project%s outside Following · run beacon projects --quiet to view", snapshot.Summary.QuietProjects, pluralSuffix(snapshot.Summary.QuietProjects)))
		if options.Width < narrowWidth {
			if err := writeWrapped(writer, "", style.wrap.Width(options.Width).Render(message)); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintln(writer, message); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}
	visibleErrors := visibleDiagnostics(snapshot, snapshot.Errors)
	if len(visibleErrors) > 0 {
		errorTitle := style.red.Render("Errors")
		if options.Width < narrowWidth {
			if err := writeWrapped(writer, "", style.wrap.Width(options.Width).Render(errorTitle)); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintln(writer, errorTitle); err != nil {
			return err
		}
		for _, scanError := range visibleErrors {
			prefix := scanError.Repository
			if prefix != "" {
				prefix += ": "
			}
			message := style.red.Render(prefix + scanError.Stage + ": " + scanError.Message)
			if options.Width < narrowWidth {
				indent := indentForWidth(options.Width, "  ")
				if err := writeWrapped(writer, indent, style.wrap.Width(max(1, options.Width-lipgloss.Width(indent))).Render(message)); err != nil {
					return err
				}
			} else if _, err := fmt.Fprintf(writer, "  %s\n", message); err != nil {
				return err
			}
		}
	}
	return nil
}
