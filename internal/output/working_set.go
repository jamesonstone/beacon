package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func workingSetTerminal(writer io.Writer, snapshot model.Snapshot, options TerminalOptions, style styles) error {
	if _, err := fmt.Fprintln(writer, style.heading.Render("Beacon Working Set")+"  "+style.dim.Render(snapshot.GeneratedAt.Local().Format("2006-01-02 15:04:05 MST"))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "%d active · %d recent · %d parked\n\n", snapshot.Summary.ActiveLanes, snapshot.Summary.RecentLanes, snapshot.Summary.ParkedLanes); err != nil {
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
		{"Active", snapshot.WorkingSet.Active}, {"Waiting", snapshot.WorkingSet.Waiting},
		{"Recently Active", snapshot.WorkingSet.Recent},
	}
	if options.IncludeParked {
		sections = append(sections, struct {
			title string
			ids   []string
		}{"Parked", snapshot.WorkingSet.Parked})
	}
	visible := 0
	for _, section := range sections {
		if len(section.ids) == 0 {
			continue
		}
		visible += len(section.ids)
		if _, err := fmt.Fprintln(writer, style.heading.Render(section.title)); err != nil {
			return err
		}
		for _, id := range section.ids {
			lane, found := byID[id]
			if !found || lane.Attention == nil {
				continue
			}
			title := workItem(lane)
			if lane.Attention.Title != "" {
				title = lane.Attention.Title
			}
			identity := lane.Repository + "/" + lane.Branch
			if lane.Attention.Manual {
				identity = "manual"
			}
			identity += " · " + relativeAge(snapshot.GeneratedAt, lane.UpdatedAt)
			line := style.project.Render(title) + "  " + style.dim.Render(identity)
			if lane.Attention.Pinned {
				line += "  " + style.yellow.Render("pinned")
			}
			if _, err := fmt.Fprintln(writer, "  "+line); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(writer, "    "+lane.Attention.Delta+"  "+style.dim.Render("Next: "+actionLabel(lane.NextAction))); err != nil {
				return err
			}
			if lane.Attention.Note != "" {
				note := lane.Attention.Note
				if lane.Attention.NoteStale {
					note += " (evidence changed since note)"
				}
				if _, err := fmt.Fprintln(writer, "    "+style.yellow.Render("Note: "+note)); err != nil {
					return err
				}
			}
			if len(lane.Attention.Tags) > 0 {
				if _, err := fmt.Fprintln(writer, "    "+style.project.Render("Tags: "+strings.Join(lane.Attention.Tags, ", "))); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}
	if visible == 0 {
		if _, err := fmt.Fprintln(writer, style.dim.Render("No active lanes · use beacon add --manual, beacon pin, or beacon scan for full diagnostics")); err != nil {
			return err
		}
	}
	if !options.IncludeParked && len(snapshot.WorkingSet.Parked) > 0 {
		_, err := fmt.Fprintf(writer, "%s\n", style.dim.Render(fmt.Sprintf("%d parked lane%s · use beacon lanes --parked to view", len(snapshot.WorkingSet.Parked), pluralSuffix(len(snapshot.WorkingSet.Parked)))))
		return err
	}
	return nil
}

func relativeAge(now, updated time.Time) string {
	if updated.IsZero() {
		return "activity unknown"
	}
	age := now.Sub(updated)
	if age < 0 {
		age = 0
	}
	switch {
	case age < time.Minute:
		return "active now"
	case age < time.Hour:
		return fmt.Sprintf("active %dm ago", int(age/time.Minute))
	case age < 24*time.Hour:
		return fmt.Sprintf("active %dh ago", int(age/time.Hour))
	default:
		return fmt.Sprintf("active %dd ago", int(age/(24*time.Hour)))
	}
}
