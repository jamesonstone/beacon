package cli

import (
	"context"
	"errors"

	"github.com/charmbracelet/huh"
	"github.com/jamesonstone/beacon/internal/reposync"
)

func (p huhPrompter) SelectRepositoryUpdates(ctx context.Context, repositories []reposync.Repository) ([]string, error) {
	if len(repositories) == 0 {
		return nil, errors.New("no repositories can be updated automatically")
	}
	options := make([]huh.Option[string], 0, len(repositories))
	selected := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		label := repository.Name + "  " + repository.CurrentBranch + " → " + repository.Base
		options = append(options, huh.NewOption(label, repository.ProjectID).Selected(true))
		selected = append(selected, repository.ProjectID)
	}
	form := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Repositories to bring up to date").
			Description("Only fast-forward-safe updates are selectable.").
			Options(options...).
			Validate(func(values []string) error {
				if len(values) == 0 {
					return errors.New("select at least one repository")
				}
				return nil
			}).
			Value(&selected),
	))
	if err := p.run(ctx, form.WithTheme(huh.ThemeCatppuccin())); err != nil {
		return nil, err
	}
	return selected, nil
}
