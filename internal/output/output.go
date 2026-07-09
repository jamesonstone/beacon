package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jamesonstone/beacon/internal/model"
)

func JSON(writer io.Writer, snapshot model.Snapshot) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(snapshot)
}

func Terminal(writer io.Writer, snapshot model.Snapshot) error {
	if _, err := fmt.Fprintf(writer, "Beacon\nGenerated %s\n\n", snapshot.GeneratedAt.Local().Format("2006-01-02 15:04:05 MST")); err != nil {
		return err
	}
	byID := make(map[string]model.Lane, len(snapshot.Lanes))
	for _, lane := range snapshot.Lanes {
		byID[lane.ID] = lane
	}
	sections := []struct {
		title  string
		symbol string
		ids    []string
	}{
		{"Ready for Review", "✓", snapshot.Groups.Ready},
		{"Needs Action", "!", snapshot.Groups.Action},
		{"Waiting", "…", snapshot.Groups.Waiting},
		{"Idle", "·", snapshot.Groups.Idle},
	}
	for _, section := range sections {
		if len(section.ids) == 0 {
			continue
		}
		if _, err := fmt.Fprintln(writer, section.title); err != nil {
			return err
		}
		for _, id := range section.ids {
			lane := byID[id]
			if _, err := fmt.Fprintf(writer, "  %s %-18s %-24s %s\n", section.symbol, lane.Repository, lane.Branch, describe(lane)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(writer, "    Action: %s\n", actionLabel(lane.NextAction)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}
	if len(snapshot.Errors) > 0 {
		if _, err := fmt.Fprintln(writer, "Errors"); err != nil {
			return err
		}
		for _, scanError := range snapshot.Errors {
			prefix := scanError.Repository
			if prefix != "" {
				prefix += ": "
			}
			if _, err := fmt.Fprintf(writer, "  ! %s%s: %s\n", prefix, scanError.Stage, scanError.Message); err != nil {
				return err
			}
		}
	}
	return nil
}

func describe(lane model.Lane) string {
	parts := []string{string(lane.Signals.Worktree), string(lane.Signals.Publication)}
	if lane.PullRequest != nil {
		parts = append(parts, fmt.Sprintf("PR #%d", lane.PullRequest.Number), "CI "+string(lane.Signals.CI))
	}
	return strings.Join(parts, ", ")
}

func actionLabel(action model.Action) string {
	switch action {
	case model.ActionReviewPR:
		return "review pull request"
	case model.ActionResolveConflict:
		return "resolve conflicts"
	case model.ActionFixCI:
		return "fix failing CI"
	case model.ActionAddressReview:
		return "address requested changes"
	case model.ActionInspectLocal:
		return "inspect local changes"
	case model.ActionPushBranch:
		return "push branch"
	case model.ActionCreatePR:
		return "create pull request"
	case model.ActionMarkReady:
		return "mark pull request ready"
	case model.ActionRefreshState:
		return "refresh or reconcile state"
	case model.ActionResumeOrClose:
		return "resume or close stale work"
	default:
		return "none"
	}
}
