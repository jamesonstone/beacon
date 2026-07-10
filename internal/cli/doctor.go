package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type doctorReport struct {
	OK     bool          `json:"ok"`
	Checks []doctorCheck `json:"checks"`
}

func (a App) doctorCommand(configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "doctor", Short: "Validate dependencies, authentication, and configuration", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report := a.runDoctor(cmd.Context(), *configPath)
			if jsonOutput {
				encoder := json.NewEncoder(a.Out)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(report); err != nil {
					return err
				}
			} else {
				for _, check := range report.Checks {
					symbol := "✓"
					if !check.OK {
						symbol = "!"
					}
					if _, err := fmt.Fprintf(a.Out, "%s %-18s %s\n", symbol, check.Name, check.Message); err != nil {
						return err
					}
				}
			}
			if !report.OK {
				return errors.New("doctor found one or more problems")
			}
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	return command
}

func (a App) runDoctor(ctx context.Context, configPath string) doctorReport {
	report := doctorReport{OK: true}
	add := func(name string, ok bool, message string) {
		report.Checks = append(report.Checks, doctorCheck{Name: name, OK: ok, Message: message})
		if !ok {
			report.OK = false
		}
	}
	for _, executable := range []string{"git", "gh"} {
		path, err := exec.LookPath(executable)
		if err != nil {
			add(executable, false, err.Error())
		} else {
			add(executable, true, path)
		}
	}
	commandContext, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if _, err := a.Runner.Run(commandContext, "", "gh", "auth", "status"); err != nil {
		add("github auth", false, err.Error())
	} else {
		add("github auth", true, "authenticated")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		add("configuration", false, err.Error())
		return report
	}
	add("configuration", true, cfg.Path)
	configDirectory := filepath.Dir(cfg.Path)
	if info, statErr := os.Stat(configDirectory); statErr != nil {
		add("config directory", false, statErr.Error())
	} else if info.Mode().Perm()&0o200 == 0 {
		add("config directory", false, "directory is not writable: "+configDirectory)
	} else {
		add("config directory", true, configDirectory)
	}

	repositories := append([]config.Repository{}, cfg.Repositories...)
	if len(cfg.Sources) > 0 {
		result := (discovery.Discoverer{Runner: a.Runner}).Discover(ctx, cfg.Sources)
		add("source discovery", len(result.Repositories) > 0, fmt.Sprintf("%d repositories discovered", len(result.Repositories)))
		for _, warning := range result.Warnings {
			add("source "+warning.Stage, false, warning.Path+": "+warning.Message)
		}
		repositories = append(repositories, result.Repositories...)
	}
	seen := make(map[string]struct{}, len(repositories))
	for _, repository := range repositories {
		if _, exists := seen[repository.GitHub]; exists {
			continue
		}
		seen[repository.GitHub] = struct{}{}
		checkContext, checkCancel := context.WithTimeout(ctx, 20*time.Second)
		_, gitErr := a.Runner.Run(checkContext, repository.Path, "git", "worktree", "list", "--porcelain", "-z")
		checkCancel()
		if gitErr != nil {
			add(repository.Name+" git", false, gitErr.Error())
		} else {
			add(repository.Name+" git", true, repository.Path)
		}
		githubContext, githubCancel := context.WithTimeout(ctx, 20*time.Second)
		_, githubErr := a.Runner.Run(githubContext, "", "gh", "repo", "view", repository.GitHub, "--json", "nameWithOwner")
		githubCancel()
		if githubErr != nil {
			add(repository.Name+" github", false, githubErr.Error())
		} else {
			add(repository.Name+" github", true, repository.GitHub)
		}
	}
	return report
}
