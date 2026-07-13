package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/notes"
	"github.com/spf13/cobra"
)

func (a App) notesCommand(configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "notes", Short: "Read and edit the local Markdown signal log", Args: noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return a.showNotes(jsonOutput)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes document as JSON")
	command.AddCommand(
		a.notesShowCommand(),
		a.notesWriteCommand(configPath, "set", false),
		a.notesWriteCommand(configPath, "append", true),
		a.notesEditCommand(configPath),
		a.notesPathCommand(),
	)
	return command
}

func (a App) notesShowCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "show", Short: "Print the Markdown signal log", Args: noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return a.showNotes(jsonOutput)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the notes document as JSON")
	return command
}

func (a App) showNotes(jsonOutput bool) error {
	path, err := notes.ResolvePath()
	if err != nil {
		return err
	}
	document, err := (notes.FileStore{}).Load(path)
	if err != nil {
		return err
	}
	if jsonOutput {
		encoder := json.NewEncoder(a.Out)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(document)
	}
	_, err = io.WriteString(a.Out, document.Content)
	return err
}

func (a App) notesWriteCommand(configPath *string, name string, appendContent bool) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: name + " [markdown]", Short: commandTitle(name) + " the Markdown signal log", Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := a.noteInput(args)
			if err != nil {
				return err
			}
			document, err := a.writeNotes(cmd.Context(), *configPath, content, appendContent)
			if err != nil {
				return err
			}
			if jsonOutput {
				return json.NewEncoder(a.Out).Encode(document)
			}
			_, err = fmt.Fprintln(a.Out, "saved signal notes")
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit the saved notes document as JSON")
	return command
}

func (a App) writeNotes(ctx context.Context, configPath, content string, appendContent bool) (notes.Document, error) {
	resolvedConfig, configErr := config.ResolvePath(configPath)
	if configErr == nil {
		paths, pathErr := agent.ResolvePaths(resolvedConfig)
		if pathErr == nil {
			requestType := agent.RequestSetNotes
			if appendContent {
				requestType = agent.RequestAppendNotes
			}
			event, requestErr := a.agentClient(paths.Socket).Request(ctx, agent.Request{Type: requestType, Content: content})
			if requestErr == nil {
				if event.Notes != nil {
					return *event.Notes, nil
				}
				if event.Message != "" {
					return notes.Document{}, errors.New(event.Message)
				}
				return notes.Document{}, errors.New("Beacon agent returned no notes document")
			}
			if !errors.Is(requestErr, agent.ErrUnavailable) {
				return notes.Document{}, requestErr
			}
		}
	}
	path, err := notes.ResolvePath()
	if err != nil {
		return notes.Document{}, err
	}
	if appendContent {
		return (notes.FileStore{}).Append(path, content)
	}
	return (notes.FileStore{}).Write(path, content)
}

func (a App) noteInput(args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	if a.inputIsTTY() {
		return "", usageError{fmt.Errorf("provide Markdown text or pipe it on standard input")}
	}
	contents, err := io.ReadAll(io.LimitReader(a.input(), notes.MaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read Markdown notes from standard input: %w", err)
	}
	if len(contents) > notes.MaxBytes {
		return "", fmt.Errorf("Beacon notes exceed the %d-byte limit", notes.MaxBytes)
	}
	return string(contents), nil
}

func (a App) notesPathCommand() *cobra.Command {
	return &cobra.Command{
		Use: "path", Short: "Print the local Markdown signal-log path", Args: noArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			path, err := notes.ResolvePath()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.Out, path)
			return err
		},
	}
}

func (a App) notesEditCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "edit", Short: "Open the Markdown signal log in an editor", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := notes.ResolvePath()
			if err != nil {
				return err
			}
			if runtime.GOOS != "darwin" {
				return fmt.Errorf("beacon notes edit is unsupported on %s; edit %s directly", runtime.GOOS, path)
			}
			store := notes.FileStore{}
			document, err := store.Load(path)
			if err != nil {
				return err
			}
			if document.UpdatedAt.IsZero() {
				if _, err := store.Write(path, document.Content); err != nil {
					return err
				}
			}
			_, err = a.Runner.Run(cmd.Context(), "", "open", "-W", "-t", path)
			if err != nil {
				return err
			}
			document, err = store.Load(path)
			if err != nil {
				return err
			}
			_, err = a.writeNotes(cmd.Context(), *configPath, document.Content, false)
			return err
		},
	}
}
