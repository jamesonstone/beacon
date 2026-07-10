package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

func (s initService) checkPrerequisites(ctx context.Context) error {
	for _, executable := range []string{"git", "gh"} {
		if _, err := s.lookup(executable); err != nil {
			return fmt.Errorf("%s", installGuidance(executable))
		}
	}
	if err := s.checkGitHubAuth(ctx); err == nil {
		return nil
	} else if !s.isTTY() {
		return fmt.Errorf("GitHub authentication is required; run: gh auth login")
	}
	login, promptErr := s.prompter.Confirm(ctx, "GitHub CLI is not authenticated. Run gh auth login now?")
	if promptErr != nil {
		return fmt.Errorf("confirm GitHub authentication: %w", promptErr)
	}
	if !login {
		return fmt.Errorf("GitHub authentication is required; run: gh auth login")
	}
	if err := s.authRunner.Run(ctx, "gh", "auth", "login"); err != nil {
		return fmt.Errorf("authenticate GitHub CLI: %w", err)
	}
	if err := s.checkGitHubAuth(ctx); err != nil {
		return fmt.Errorf("verify GitHub authentication: %w", err)
	}
	return nil
}

func (s initService) checkGitHubAuth(ctx context.Context) error {
	commandContext, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err := s.runner.Run(commandContext, "", "gh", "auth", "status")
	return err
}

type inheritedCommand struct {
	input  io.Reader
	output io.Writer
	errOut io.Writer
}

func (r inheritedCommand) Run(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = r.input
	command.Stdout = r.output
	command.Stderr = r.errOut
	return command.Run()
}
