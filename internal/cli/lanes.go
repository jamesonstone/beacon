package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/output"
	"github.com/spf13/cobra"
)

func (a App) lanesCommand(configPath *string) *cobra.Command {
	var parked bool
	command := &cobra.Command{
		Use: "lanes", Short: "Show the current working set", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			event, err := a.laneRequest(cmd.Context(), *configPath, agent.Request{Type: agent.RequestGetSnapshot})
			if err != nil {
				return err
			}
			if event.Snapshot == nil {
				return errors.New("agent returned no working-set snapshot")
			}
			colorMode, _ := cmd.Flags().GetString("color")
			color, err := a.resolveColor(colorMode)
			if err != nil {
				return err
			}
			return output.TerminalWithOptions(a.Out, *event.Snapshot, output.TerminalOptions{Color: color, Width: a.terminalWidth(), WorkingSet: true, IncludeParked: parked})
		},
	}
	command.Flags().BoolVar(&parked, "parked", false, "include parked lanes")
	return command
}

func (a App) laneAttentionCommand(configPath *string, name, requestType string) *cobra.Command {
	var off bool
	command := &cobra.Command{
		Use: name + " <lane-id>", Short: commandTitle(name) + " a working-set lane", Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			request := agent.Request{Type: requestType, LaneID: args[0]}
			switch name {
			case "pin":
				request.Pinned = !off
			case "park":
				request.AttentionState = string(model.AttentionParked)
			case "resume":
				request.AttentionState = string(model.AttentionActive)
			}
			_, err := a.laneRequest(cmd.Context(), *configPath, request)
			if err == nil {
				_, err = fmt.Fprintf(a.Out, "%s %s\n", name, args[0])
			}
			return err
		},
	}
	if name == "pin" {
		command.Flags().BoolVar(&off, "off", false, "remove the lane pin")
	}
	return command
}

func commandTitle(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func (a App) laneNoteCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "note <lane-id> [text]", Short: "Set or clear a lane note", Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageError{fmt.Errorf("%s requires a lane id", cmd.CommandPath())}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			note := ""
			if len(args) > 1 {
				note = strings.Join(args[1:], " ")
			}
			_, err := a.laneRequest(cmd.Context(), *configPath, agent.Request{Type: agent.RequestSetLaneNote, LaneID: args[0], Note: note})
			if err == nil {
				_, err = fmt.Fprintf(a.Out, "noted %s\n", args[0])
			}
			return err
		},
	}
}

func (a App) laneTagCommand(configPath *string, name, requestType string) *cobra.Command {
	return &cobra.Command{
		Use: name + " <lane-id> <tag>", Short: commandTitle(name) + " a working-set lane", Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return usageError{fmt.Errorf("%s requires a lane id and tag", cmd.CommandPath())}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := strings.Join(args[1:], " ")
			_, err := a.laneRequest(cmd.Context(), *configPath, agent.Request{Type: requestType, LaneID: args[0], Tag: tag})
			if err == nil {
				_, err = fmt.Fprintf(a.Out, "%s %s: %s\n", name, args[0], tag)
			}
			return err
		},
	}
}

func (a App) laneAddCommand(configPath *string) *cobra.Command {
	var manual bool
	command := &cobra.Command{
		Use: "add <title>", Short: "Add a manual working-set lane", Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageError{fmt.Errorf("%s requires a title", cmd.CommandPath())}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !manual {
				return usageError{errors.New("beacon add currently requires --manual")}
			}
			event, err := a.laneRequest(cmd.Context(), *configPath, agent.Request{Type: agent.RequestAddManualLane, Title: strings.Join(args, " ")})
			if err == nil {
				_, err = fmt.Fprintf(a.Out, "added %s\n", event.ProjectID)
			}
			return err
		},
	}
	command.Flags().BoolVar(&manual, "manual", false, "create a lane without Git evidence")
	return command
}

func (a App) laneSeenCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "seen <lane-id>", Short: "Acknowledge the current lane evidence", Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := a.laneRequest(cmd.Context(), *configPath, agent.Request{Type: agent.RequestMarkLaneSeen, LaneID: args[0]})
			if err == nil {
				_, err = fmt.Fprintf(a.Out, "seen %s\n", args[0])
			}
			return err
		},
	}
}

func (a App) laneRequest(ctx context.Context, configPath string, request agent.Request) (agent.Event, error) {
	_, paths, err := a.agentConfig(configPath)
	if err != nil {
		return agent.Event{}, err
	}
	event, err := (agent.Client{Socket: paths.Socket}).Request(ctx, request)
	if err != nil {
		return agent.Event{}, err
	}
	if event.Type == agent.EventProjectFailed {
		return agent.Event{}, errors.New(event.Message)
	}
	return event, nil
}
