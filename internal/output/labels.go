package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/muesli/termenv"
)

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
	case model.ActionContinueWork:
		return "continue work"
	default:
		return "none"
	}
}
