package cli

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

const openTargetTimeout = 5 * time.Second

func openTargetCommand(goos string, target string) (string, []string, error) {
	if target == "" {
		return "", nil, fmt.Errorf("open target is required")
	}
	switch goos {
	case "darwin":
		return "open", []string{target}, nil
	case "linux":
		return "xdg-open", []string{target}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", target}, nil
	default:
		return "", nil, fmt.Errorf("opening targets is unsupported on %s", goos)
	}
}

func (a App) openTarget(ctx context.Context, target string) error {
	name, args, err := openTargetCommand(runtime.GOOS, target)
	if err != nil {
		return err
	}
	commandContext, cancel := context.WithTimeout(ctx, openTargetTimeout)
	defer cancel()
	if _, err := a.Runner.Run(commandContext, "", name, args...); err != nil {
		return fmt.Errorf("open target %s: %w", target, err)
	}
	return nil
}
