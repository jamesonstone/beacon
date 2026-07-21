package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/spf13/cobra"
)

func (a App) openCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "open <lane-id>",
		Short: "Open a lane's pull request or worktree",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snapshot, err := a.loadSnapshot(cmd.Context(), *configPath)
			if err != nil {
				return err
			}
			for _, lane := range snapshot.Lanes {
				if lane.ID == args[0] {
					return a.openLane(cmd.Context(), lane)
				}
			}
			return fmt.Errorf("lane not found: %s", args[0])
		},
	}
}

func (a App) openNextCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "open-next",
		Short: "Open the highest-priority lane",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot, err := a.loadSnapshot(cmd.Context(), *configPath)
			if err != nil {
				return err
			}
			lane, ok := nextActiveLane(snapshot)
			if !ok {
				return errors.New("no active work lanes found")
			}
			return a.openLane(cmd.Context(), lane)
		},
	}
}

func nextActiveLane(snapshot model.Snapshot) (model.Lane, bool) {
	byID := make(map[string]model.Lane, len(snapshot.Lanes))
	for _, lane := range snapshot.Lanes {
		byID[lane.ID] = lane
	}
	groups := [][]string{snapshot.WorkingSet.Active, snapshot.WorkingSet.Waiting, snapshot.WorkingSet.Recent}
	if len(snapshot.WorkingSet.Active)+len(snapshot.WorkingSet.Waiting)+len(snapshot.WorkingSet.Recent) == 0 {
		groups = [][]string{snapshot.Groups.Ready, snapshot.Groups.Action, snapshot.Groups.Waiting}
	}
	for _, group := range groups {
		for _, id := range group {
			if lane, ok := byID[id]; ok && laneHasOpenTarget(lane) {
				return lane, true
			}
		}
	}
	return model.Lane{}, false
}

func laneHasOpenTarget(lane model.Lane) bool {
	return lane.PullRequest != nil || lane.Issue != nil || lane.Worktree != nil
}

func (a App) loadSnapshot(ctx context.Context, path string) (model.Snapshot, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return model.Snapshot{}, err
	}
	if _, paths, agentErr := a.agentConfig(path); agentErr == nil {
		if event, requestErr := (agent.Client{Socket: paths.Socket}).Request(ctx, agent.Request{Type: agent.RequestGetSnapshot}); requestErr == nil && event.Snapshot != nil {
			return *event.Snapshot, nil
		}
	}
	return a.scanSnapshot(ctx, cfg, "", false)
}

func (a App) openLane(ctx context.Context, lane model.Lane) error {
	target := ""
	if lane.PullRequest != nil {
		target = lane.PullRequest.URL
	} else if lane.Issue != nil {
		target = lane.Issue.URL
	} else if lane.Worktree != nil {
		target = lane.Worktree.Path
	}
	if target == "" {
		return fmt.Errorf("lane has no openable target: %s", lane.ID)
	}
	return a.openTarget(ctx, target)
}
