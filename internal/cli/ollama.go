package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/ollama"
	"github.com/spf13/cobra"
)

const ollamaInputLimit = ollama.MaxContextBytes + ollama.MaxPromptBytes + 4096

type ollamaClient interface {
	ListModels(context.Context) ([]ollama.Model, error)
	Chat(context.Context, ollama.ChatInput) (ollama.ChatResult, error)
}

type ollamaStatus struct {
	Models          []ollama.Model `json:"models"`
	ConfiguredModel string         `json:"configured_model"`
}

type ollamaDefault struct {
	ConfiguredModel string `json:"configured_model"`
}

func (a App) ollamaCommand(configPath *string) *cobra.Command {
	command := &cobra.Command{Use: "ollama", Short: "Use locally installed Ollama models"}
	command.AddCommand(
		a.ollamaModelsCommand(configPath),
		a.ollamaChatCommand(),
		a.ollamaSetDefaultCommand(configPath),
	)
	return command
}

func (a App) ollamaModelsCommand(configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "models", Short: "List locally installed Ollama models", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(*configPath)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()
			models, err := a.localOllamaClient().ListModels(ctx)
			if err != nil {
				return err
			}
			status := ollamaStatus{Models: models, ConfiguredModel: cfg.Settings.OllamaModel}
			if jsonOutput {
				return json.NewEncoder(a.Out).Encode(status)
			}
			if len(models) == 0 {
				_, err = fmt.Fprintln(a.Out, "No local Ollama models are installed.")
				return err
			}
			for _, model := range models {
				marker := ""
				if model.Name == status.ConfiguredModel {
					marker = " (default)"
				}
				if _, err := fmt.Fprintf(a.Out, "%s%s\n", model.Name, marker); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	return command
}

func (a App) ollamaChatCommand() *cobra.Command {
	var model string
	var jsonOutput bool
	command := &cobra.Command{
		Use: "chat", Short: "Ask a local Ollama model with optional Notes context", Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			input, err := decodeOllamaInput(a.input())
			if err != nil {
				return err
			}
			input.Model = strings.TrimSpace(model)
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			result, err := a.localOllamaClient().Chat(ctx, input)
			if err != nil {
				return err
			}
			if jsonOutput {
				return json.NewEncoder(a.Out).Encode(result)
			}
			_, err = fmt.Fprintln(a.Out, result.Content)
			return err
		},
	}
	command.Flags().StringVar(&model, "model", "", "installed local Ollama model")
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	if err := command.MarkFlagRequired("model"); err != nil {
		panic(err)
	}
	return command
}

func (a App) ollamaSetDefaultCommand(configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "set-default MODEL", Short: "Set the default Notes Ollama model", Args: exactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			model := strings.TrimSpace(args[0])
			if model == "" {
				return usageError{errors.New("MODEL is required")}
			}
			cfg, err := config.Load(*configPath)
			if err != nil {
				return err
			}
			cfg.Settings.OllamaModel = model
			if err := (config.AtomicWriter{}).Write(cfg.Path, cfg); err != nil {
				return err
			}
			result := ollamaDefault{ConfiguredModel: model}
			if jsonOutput {
				return json.NewEncoder(a.Out).Encode(result)
			}
			_, err = fmt.Fprintf(a.Out, "Default Ollama model set to %s.\n", model)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON only")
	return command
}

func (a App) localOllamaClient() ollamaClient {
	if a.ollamaClientSource != nil {
		return a.ollamaClientSource()
	}
	return ollama.New()
}

func decodeOllamaInput(reader io.Reader) (ollama.ChatInput, error) {
	limited := io.LimitReader(reader, ollamaInputLimit+1)
	contents, err := io.ReadAll(limited)
	if err != nil {
		return ollama.ChatInput{}, fmt.Errorf("read Ollama chat input: %w", err)
	}
	if len(contents) > ollamaInputLimit {
		return ollama.ChatInput{}, fmt.Errorf("Ollama chat input exceeds the %d-byte limit", ollamaInputLimit)
	}
	var input ollama.ChatInput
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return ollama.ChatInput{}, usageError{fmt.Errorf("decode Ollama chat input: %w", err)}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return ollama.ChatInput{}, usageError{errors.New("Ollama chat input must contain one JSON document")}
		}
		return ollama.ChatInput{}, usageError{fmt.Errorf("decode Ollama chat input: %w", err)}
	}
	return input, nil
}
