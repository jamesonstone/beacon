package cli

import (
	"context"
	"errors"
	"time"

	"github.com/jamesonstone/beacon/internal/agent"
)

func waitForAgentRefresh(ctx context.Context, client agentRequestClient) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		event, err := client.Request(ctx, agent.Request{Type: agent.RequestGetAgentStatus})
		if err != nil {
			return err
		}
		if event.Status == nil {
			return errors.New("agent returned no status during manual refresh")
		}
		if !event.Status.Refreshing {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
