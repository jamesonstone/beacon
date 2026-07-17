package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/jamesonstone/beacon/internal/model"
)

func visibleDiagnostics(snapshot model.Snapshot, diagnostics []model.ScanError) []model.ScanError {
	untracked := make(map[string]struct{})
	for _, project := range snapshot.Projects {
		if project.TrackingState != model.TrackingUntracked {
			continue
		}
		untracked[project.Name] = struct{}{}
		untracked[project.GitHub] = struct{}{}
	}
	visible := make([]model.ScanError, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if _, hidden := untracked[diagnostic.Repository]; diagnostic.Repository != "" && hidden {
			continue
		}
		visible = append(visible, diagnostic)
	}
	return visible
}

func renderTable(writer io.Writer, lanes []model.Lane, style styles) error {
	var buffer bytes.Buffer
	table := tabwriter.NewWriter(&buffer, 2, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "PROJECT\tWORK ITEM\tSTATUS\tLAST PROGRESS\tNEXT ACTION"); err != nil {
		return err
	}
	for _, lane := range lanes {
		status := statusLabel(lane)
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
			lane.Repository,
			workItem(lane),
			status,
			progressLabel(lane),
			actionLabel(lane.NextAction)); err != nil {
			return err
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}
	lines := strings.SplitAfter(buffer.String(), "\n")
	for index, lane := range lanes {
		lineIndex := index + 1 // line zero is the table header
		if lineIndex >= len(lines) {
			break
		}
		line := lines[lineIndex]
		line = replaceFirst(line, lane.Repository, style.project.Render(lane.Repository))
		status := statusLabel(lane)
		line = replaceFirst(line, status, statusStyle(style, lane).Render(status))
		if lane.Progress == nil {
			progress := progressLabel(lane)
			line = replaceFirst(line, progress, style.dim.Render(progress))
		}
		action := actionLabel(lane.NextAction)
		line = replaceLast(line, action, actionStyle(style, lane.NextAction).Render(action))
		lines[lineIndex] = line
	}
	_, err := io.WriteString(writer, strings.Join(lines, ""))
	return err
}

func renderNarrow(writer io.Writer, lanes []model.Lane, style styles, width int) error {
	titleIndent := indentForWidth(width, "  ")
	detailIndent := indentForWidth(width, "    ")
	titleWidth := max(1, width-lipgloss.Width(titleIndent))
	detailWidth := max(1, width-lipgloss.Width(detailIndent))
	for _, lane := range lanes {
		title := style.project.Render(lane.Repository) + "  " + statusStyle(style, lane).Render(workItem(lane)+" · "+statusLabel(lane))
		if err := writeWrapped(writer, titleIndent, style.wrap.Width(titleWidth).Render(title)); err != nil {
			return err
		}
		next := style.dim.Render(progressLabel(lane)) + "  " + actionStyle(style, lane.NextAction).Render("Next: "+actionLabel(lane.NextAction))
		if err := writeWrapped(writer, detailIndent, style.wrap.Width(detailWidth).Render(next)); err != nil {
			return err
		}
		if evidence := evidenceLabel(lane); evidence != "" {
			if err := writeWrapped(writer, detailIndent, style.wrap.Width(detailWidth).Render(evidence)); err != nil {
				return err
			}
		}
	}
	return nil
}

func nextActionLine(snapshot model.Snapshot, lanes map[string]model.Lane, style styles) string {
	lane, found := firstActionLane(snapshot, lanes)
	if !found {
		return ""
	}
	action := lane.NextAction
	if action == model.ActionNone && lane.ReviewReady {
		action = model.ActionReviewPR
	}
	return style.heading.Render("Next:") + " " +
		actionStyle(style, action).Render(actionLabel(action)) +
		" · " + style.project.Render(lane.Repository) +
		" · " + workItem(lane)
}

func firstActionLane(snapshot model.Snapshot, lanes map[string]model.Lane) (model.Lane, bool) {
	for _, group := range [][]string{snapshot.Groups.Ready, snapshot.Groups.Action} {
		for _, id := range group {
			lane, found := lanes[id]
			if found {
				return lane, true
			}
		}
	}
	return model.Lane{}, false
}

func indentForWidth(width int, preferred string) string {
	if width <= 1 {
		return ""
	}
	for lipgloss.Width(preferred) >= width {
		preferred = strings.TrimSuffix(preferred, " ")
	}
	return preferred
}

func writeWrapped(writer io.Writer, indent, value string) error {
	_, err := fmt.Fprintln(writer, indent+strings.ReplaceAll(value, "\n", "\n"+indent))
	return err
}

func replaceFirst(value, old, replacement string) string {
	index := strings.Index(value, old)
	if index < 0 {
		return value
	}
	return value[:index] + replacement + value[index+len(old):]
}

func replaceLast(value, old, replacement string) string {
	index := strings.LastIndex(value, old)
	if index < 0 {
		return value
	}
	return value[:index] + replacement + value[index+len(old):]
}
