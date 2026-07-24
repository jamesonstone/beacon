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

func WorkTerminal(writer io.Writer, scan model.WorkScan, options TerminalOptions) error {
	if options.Width <= 0 {
		options.Width = 120
	}
	style := terminalStyles(writer, options.Color)
	if _, err := fmt.Fprintln(writer, style.heading.Render("Beacon v2")); err != nil {
		return err
	}
	summary := fmt.Sprintf("%d project%s · %d active · %d work item%s",
		scan.Summary.Projects, pluralSuffix(scan.Summary.Projects),
		scan.Summary.ActiveProjects,
		scan.Summary.WorkItems, pluralSuffix(scan.Summary.WorkItems))
	if scan.Summary.UnknownProjects > 0 {
		summary += fmt.Sprintf(" · %d unknown", scan.Summary.UnknownProjects)
	}
	if _, err := fmt.Fprintln(writer, summary); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer); err != nil {
		return err
	}

	if len(scan.Items) == 0 {
		if _, err := fmt.Fprintln(writer, style.dim.Render("No work in progress.")); err != nil {
			return err
		}
	} else if options.Width < narrowWidth {
		if err := renderNarrowWork(writer, scan.Items, style, options.Width); err != nil {
			return err
		}
	} else if err := renderWorkTable(writer, scan.Items, style); err != nil {
		return err
	}

	if !options.IncludeIdle && scan.Summary.IdleProjects > 0 {
		if _, err := fmt.Fprintf(writer, "\n%s\n", style.dim.Render(fmt.Sprintf(
			"%d idle project%s hidden · use --include-idle to show",
			scan.Summary.IdleProjects, pluralSuffix(scan.Summary.IdleProjects),
		))); err != nil {
			return err
		}
	}
	if err := renderWorkDiagnostics(writer, "Warnings", scan.Warnings, style.yellow); err != nil {
		return err
	}
	return renderWorkDiagnostics(writer, "Errors", scan.Errors, style.red)
}

func renderWorkTable(writer io.Writer, items []model.WorkItem, style styles) error {
	var buffer bytes.Buffer
	table := tabwriter.NewWriter(&buffer, 2, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "PROJECT\tWORK\tSTATE\tNEXT"); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\n",
			item.Repository, workLabel(item), workStateLabel(item.State), workActionLabel(item.NextAction)); err != nil {
			return err
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}
	lines := strings.SplitAfter(buffer.String(), "\n")
	for index, item := range items {
		lineIndex := index + 1
		if lineIndex >= len(lines) {
			break
		}
		line := replaceFirst(lines[lineIndex], item.Repository, style.project.Render(item.Repository))
		state := workStateLabel(item.State)
		line = replaceFirst(line, state, workStateStyle(style, item.State).Render(state))
		action := workActionLabel(item.NextAction)
		line = replaceLast(line, action, actionStyle(style, item.NextAction).Render(action))
		lines[lineIndex] = line
	}
	_, err := io.WriteString(writer, strings.Join(lines, ""))
	return err
}

func renderNarrowWork(writer io.Writer, items []model.WorkItem, style styles, width int) error {
	titleIndent := indentForWidth(width, "  ")
	detailIndent := indentForWidth(width, "    ")
	titleWidth := max(1, width-lipgloss.Width(titleIndent))
	detailWidth := max(1, width-lipgloss.Width(detailIndent))
	for _, item := range items {
		title := style.project.Render(item.Repository) + "  " +
			workStateStyle(style, item.State).Render(workLabel(item)+" · "+workStateLabel(item.State))
		if err := writeWrapped(writer, titleIndent, style.wrap.Width(titleWidth).Render(title)); err != nil {
			return err
		}
		next := style.dim.Render("Next: ") + actionStyle(style, item.NextAction).Render(workActionLabel(item.NextAction))
		if err := writeWrapped(writer, detailIndent, style.wrap.Width(detailWidth).Render(next)); err != nil {
			return err
		}
	}
	return nil
}

func renderWorkDiagnostics(
	writer io.Writer,
	title string,
	diagnostics []model.ScanError,
	style lipgloss.Style,
) error {
	if len(diagnostics) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(writer, "\n%s\n", style.Render(title)); err != nil {
		return err
	}
	for _, diagnostic := range diagnostics {
		prefix := diagnostic.Repository
		if prefix != "" {
			prefix += ": "
		}
		if _, err := fmt.Fprintf(writer, "  %s\n", style.Render(prefix+diagnostic.Stage+": "+diagnostic.Message)); err != nil {
			return err
		}
	}
	return nil
}

func workLabel(item model.WorkItem) string {
	if item.PullRequest != nil {
		return fmt.Sprintf("PR #%d · %s", item.PullRequest.Number, item.Branch)
	}
	if item.Branch != "" {
		return item.Branch
	}
	return item.Base
}

func workStateLabel(state model.WorkState) string {
	switch state {
	case model.WorkConflict:
		return "conflict"
	case model.WorkCIFailed:
		return "CI failed"
	case model.WorkFeedback:
		return "feedback"
	case model.WorkDirty:
		return "local changes"
	case model.WorkUnpublished:
		return "unpublished"
	case model.WorkDraft:
		return "draft PR"
	case model.WorkPullRequest:
		return "open PR"
	case model.WorkBranch:
		return "branch"
	default:
		return "idle"
	}
}

func workStateStyle(style styles, state model.WorkState) lipgloss.Style {
	switch state {
	case model.WorkConflict, model.WorkCIFailed, model.WorkFeedback:
		return style.red
	case model.WorkDirty, model.WorkUnpublished, model.WorkDraft,
		model.WorkPullRequest, model.WorkBranch:
		return style.yellow
	default:
		return style.dim
	}
}

func workActionLabel(action model.Action) string {
	if action == model.ActionNone {
		return "—"
	}
	return actionLabel(action)
}
