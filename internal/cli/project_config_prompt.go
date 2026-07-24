package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
)

func (p huhPrompter) ChooseConfiguredProject(
	ctx context.Context,
	view projectBrowserView,
) (projectBrowserAction, error) {
	options := make([]huh.Option[projectBrowserAction], 0, len(view.Entries)+3)
	if view.Current != view.Root {
		options = append(options, huh.NewOption("← ..", projectBrowserAction{Kind: projectBrowserUp}))
	}
	for _, entry := range view.Entries {
		label := "→ " + entry.Name + "/"
		action := projectBrowserAction{Kind: projectBrowserEnter, Path: entry.Path}
		if entry.Repository {
			marker := "[ ]"
			if entry.Selected {
				marker = "[x]"
			}
			label = marker + " " + entry.Name
			action.Kind = projectBrowserToggle
		}
		options = append(options, huh.NewOption(label, action))
	}
	options = append(options,
		huh.NewOption(
			fmt.Sprintf("Save %d selected project(s)", view.Selected),
			projectBrowserAction{Kind: projectBrowserSave},
		),
		huh.NewOption("Cancel", projectBrowserAction{Kind: projectBrowserCancel}),
	)
	description := "Enter opens a directory or toggles a repository."
	if view.SelectedOutside > 0 {
		description += fmt.Sprintf(" %d selected outside this root will be preserved.", view.SelectedOutside)
	}
	var action projectBrowserAction
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[projectBrowserAction]().
			Title(view.Current).
			Description(description).
			Options(options...).
			Value(&action),
	)).WithTheme(huh.ThemeCatppuccin())
	if err := p.run(ctx, form); err != nil {
		return projectBrowserAction{}, err
	}
	return action, nil
}
