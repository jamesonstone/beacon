package cli

import (
	"context"

	"github.com/charmbracelet/huh"
	"github.com/jamesonstone/beacon/internal/model"
)

func (p huhPrompter) SelectTrackedProjects(ctx context.Context, projects []model.Project) ([]string, error) {
	options := make([]huh.Option[string], 0, len(projects))
	selected := trackedProjectIDs(projects)
	for _, project := range projects {
		option := huh.NewOption(project.Name+"  "+project.GitHub, project.GitHub)
		if project.TrackingState != model.TrackingUntracked {
			option = option.Selected(true)
		}
		options = append(options, option)
	}
	form := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Projects Beacon should track").
			Description("Space toggles a project; type / to filter.").
			Options(options...).
			Value(&selected),
	))
	form = form.WithTheme(huh.ThemeCatppuccin())
	if err := p.run(ctx, form); err != nil {
		return nil, err
	}
	return selected, nil
}
