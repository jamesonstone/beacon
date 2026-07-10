package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/muesli/termenv"
)

const narrowWidth = 100

type TerminalOptions struct {
	Color bool
	Width int
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
	header := style.heading.Render("Beacon") + "  " + style.dim.Render(snapshot.GeneratedAt.Local().Format("2006-01-02 15:04:05 MST"))
	if options.Width < narrowWidth {
		if err := writeWrapped(writer, "", style.wrap.Width(options.Width).Render(header)); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintln(writer, header); err != nil {
		return err
	}
	summary := fmt.Sprintf("%d projects · %d work items · %d ready · %d issues · %d unresolved feedback",
		snapshot.Summary.Projects, snapshot.Summary.Total, snapshot.Summary.ReviewReady, snapshot.Summary.OpenIssues, snapshot.Summary.UnresolvedFeedback)
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
	sections := []struct {
		title string
		ids   []string
	}{
		{"Ready for Review", snapshot.Groups.Ready},
		{"Needs Action", snapshot.Groups.Action},
		{"Waiting", snapshot.Groups.Waiting},
		{"Idle", snapshot.Groups.Idle},
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
	if len(snapshot.Errors) > 0 {
		errorTitle := style.red.Render("Errors")
		if options.Width < narrowWidth {
			if err := writeWrapped(writer, "", style.wrap.Width(options.Width).Render(errorTitle)); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintln(writer, errorTitle); err != nil {
			return err
		}
		for _, scanError := range snapshot.Errors {
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

func terminalStyles(writer io.Writer, color bool) styles {
	renderer := lipgloss.NewRenderer(writer)
	if color {
		renderer.SetColorProfile(termenv.ANSI)
	} else {
		renderer.SetColorProfile(termenv.Ascii)
	}
	return styles{
		project: renderer.NewStyle().Foreground(lipgloss.Color("6")),
		green:   renderer.NewStyle().Foreground(lipgloss.Color("2")),
		yellow:  renderer.NewStyle().Foreground(lipgloss.Color("3")),
		red:     renderer.NewStyle().Foreground(lipgloss.Color("1")),
		dim:     renderer.NewStyle().Faint(true),
		heading: renderer.NewStyle().Bold(true),
		wrap:    renderer.NewStyle(),
	}
}

func statusStyle(style styles, lane model.Lane) lipgloss.Style {
	if lane.NextAction == model.ActionResolveConflict || lane.NextAction == model.ActionFixCI || lane.NextAction == model.ActionAddressReview {
		return style.red
	}
	if lane.ReviewReady && lane.NextAction != model.ActionWaitForCI && lane.NextAction != model.ActionManualTestMerge {
		return style.green
	}
	if lane.NextAction != model.ActionNone {
		return style.yellow
	}
	return style.dim
}

func actionStyle(style styles, action model.Action) lipgloss.Style {
	switch action {
	case model.ActionResolveConflict, model.ActionFixCI, model.ActionAddressReview:
		return style.red
	case model.ActionReviewPR, model.ActionMergePR:
		return style.green
	case model.ActionNone:
		return style.dim
	default:
		return style.yellow
	}
}

func workItem(lane model.Lane) string {
	if lane.PullRequest != nil {
		return fmt.Sprintf("PR #%d %s", lane.PullRequest.Number, lane.PullRequest.Title)
	}
	if lane.Issue != nil {
		return fmt.Sprintf("Issue #%d %s", lane.Issue.Number, lane.Issue.Title)
	}
	if lane.Branch != "" {
		return lane.Branch
	}
	return lane.ID
}

func statusLabel(lane model.Lane) string {
	if lane.ReviewReady {
		if lane.Signals.CI == model.CIPending {
			return "ready · CI pending"
		}
		return "ready"
	}
	switch lane.NextAction {
	case model.ActionResolveConflict:
		return "conflict"
	case model.ActionFixCI:
		return "CI failed"
	case model.ActionAddressReview:
		return "feedback"
	case model.ActionInspectLocal:
		return "local changes"
	case model.ActionPushBranch:
		return "unpublished"
	case model.ActionStartIssue:
		return "queued"
	case model.ActionNone:
		return "idle"
	default:
		return "waiting"
	}
}

func progressLabel(lane model.Lane) string {
	if lane.Progress != nil {
		label := lane.Progress.Phase
		if lane.Progress.Feature != "" {
			label += " · " + lane.Progress.Feature
		}
		if label != "" {
			return label
		}
	}
	if lane.UpdatedAt.IsZero() {
		return "unknown"
	}
	return lane.UpdatedAt.Local().Format("Jan 02 15:04")
}

func evidenceLabel(lane model.Lane) string {
	parts := []string{string(lane.Signals.Worktree), "CI " + string(lane.Signals.CI), "review " + string(lane.Signals.Review), string(lane.Signals.Freshness)}
	if lane.Progress != nil && lane.Progress.Summary != "" {
		parts = append(parts, lane.Progress.Summary)
	}
	return strings.Join(parts, " · ")
}

func actionLabel(action model.Action) string {
	switch action {
	case model.ActionReviewPR:
		return "review manually"
	case model.ActionResolveConflict:
		return "resolve conflicts"
	case model.ActionFixCI:
		return "fix failing CI"
	case model.ActionAddressReview:
		return "address review feedback"
	case model.ActionInspectLocal:
		return "inspect local changes"
	case model.ActionPushBranch:
		return "push branch"
	case model.ActionCreatePR:
		return "create pull request"
	case model.ActionMarkReady:
		return "mark pull request ready"
	case model.ActionWaitForCI:
		return "wait for CI"
	case model.ActionManualTestMerge:
		return "manual test, then merge"
	case model.ActionMergePR:
		return "merge pull request"
	case model.ActionStartIssue:
		return "start issue"
	case model.ActionRefreshState:
		return "refresh or reconcile state"
	case model.ActionResumeOrClose:
		return "resume or close stale work"
	default:
		return "none"
	}
}
