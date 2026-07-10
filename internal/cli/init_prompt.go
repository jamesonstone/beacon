package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/jamesonstone/beacon/internal/config"
)

type initMode string

const (
	initModeWatch  initMode = "watch"
	initModeSelect initMode = "select"
)

type huhPrompter struct {
	input  io.Reader
	output io.Writer
}

func (p huhPrompter) ChooseMode(ctx context.Context) (initMode, error) {
	mode := initModeWatch
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[initMode]().
			Title("How should Beacon add repositories?").
			Options(
				huh.NewOption("Watch an entire directory", initModeWatch),
				huh.NewOption("Select individual repositories", initModeSelect),
			).
			Value(&mode),
	))
	if err := p.run(ctx, form); err != nil {
		return "", err
	}
	return mode, nil
}

func (p huhPrompter) Directory(ctx context.Context) (string, error) {
	var directory string
	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Directory to search").
			Placeholder("~/go/src/github.com").
			Validate(validateDirectory).
			Value(&directory),
	))
	if err := p.run(ctx, form); err != nil {
		return "", err
	}
	return directory, nil
}

func (p huhPrompter) SelectRepositories(ctx context.Context, repositories []config.Repository) ([]config.Repository, error) {
	if len(repositories) == 0 {
		return nil, errors.New("no accessible GitHub repositories were discovered")
	}
	options := make([]huh.Option[config.Repository], 0, len(repositories))
	for _, repository := range repositories {
		options = append(options, huh.NewOption(repository.GitHub+"  "+repository.Path, repository))
	}
	var selected []config.Repository
	form := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[config.Repository]().
			Title("Repositories to add").
			Options(options...).
			Validate(func(values []config.Repository) error {
				if len(values) == 0 {
					return errors.New("select at least one repository")
				}
				return nil
			}).
			Value(&selected),
	))
	if err := p.run(ctx, form); err != nil {
		return nil, err
	}
	return selected, nil
}

func (p huhPrompter) Confirm(ctx context.Context, title string) (bool, error) {
	confirmed := false
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Affirmative("Yes").Negative("No").Value(&confirmed),
	))
	if err := p.run(ctx, form); err != nil {
		return false, err
	}
	return confirmed, nil
}

func (p huhPrompter) run(ctx context.Context, form *huh.Form) error {
	return form.WithInput(p.input).WithOutput(p.output).RunWithContext(ctx)
}

func validateDirectory(value string) error {
	path, err := config.CanonicalizePath(value)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("source must not be a symbolic link")
	}
	if !info.IsDir() {
		return errors.New("path is not a directory")
	}
	return nil
}
