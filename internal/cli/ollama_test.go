package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/ollama"
)

type fakeOllamaClient struct {
	models []ollama.Model
	result ollama.ChatResult
	input  ollama.ChatInput
}

func (client *fakeOllamaClient) ListModels(context.Context) ([]ollama.Model, error) {
	return client.models, nil
}

func (client *fakeOllamaClient) Chat(_ context.Context, input ollama.ChatInput) (ollama.ChatResult, error) {
	client.input = input
	return client.result, nil
}

func TestOllamaModelsReturnsLocalStatusWithoutStartingAgent(t *testing.T) {
	configPath := writeOllamaTestConfig(t, "gpt-oss:20b")
	fake := &fakeOllamaClient{models: []ollama.Model{{Name: "gpt-oss:20b", Size: 42, Details: ollama.ModelDetails{Format: "gguf"}}}}
	var output bytes.Buffer
	app := App{
		In: strings.NewReader(""), Out: &output, Err: &bytes.Buffer{},
		ollamaClientSource: func() ollamaClient { return fake }, autoStartAgent: true,
	}
	root := app.Root()
	root.SetArgs([]string{"--config", configPath, "ollama", "models", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var status ollamaStatus
	if err := json.Unmarshal(output.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.ConfiguredModel != "gpt-oss:20b" || len(status.Models) != 1 {
		t.Fatalf("status = %#v", status)
	}
}

func TestOllamaChatReadsContextAndPromptFromStdin(t *testing.T) {
	fake := &fakeOllamaClient{result: ollama.ChatResult{Model: "local:latest", Content: "answer"}}
	var output bytes.Buffer
	app := App{
		In:  strings.NewReader(`{"context":"private note","prompt":"summarize"}`),
		Out: &output, Err: &bytes.Buffer{}, ollamaClientSource: func() ollamaClient { return fake },
	}
	root := app.Root()
	root.SetArgs([]string{"ollama", "chat", "--model", "local:latest", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if fake.input.Context != "private note" || fake.input.Prompt != "summarize" || fake.input.Model != "local:latest" {
		t.Fatalf("input = %#v", fake.input)
	}
	if strings.Contains(strings.Join(root.Flags().Args(), " "), "private note") {
		t.Fatal("Notes context leaked into command arguments")
	}
	var result ollama.ChatResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil || result.Content != "answer" {
		t.Fatalf("result = %#v, %v", result, err)
	}
}

func TestOllamaChatAllowsPromptWithoutNotesContext(t *testing.T) {
	fake := &fakeOllamaClient{result: ollama.ChatResult{Model: "local:latest", Content: "answer"}}
	app := App{
		In: strings.NewReader(`{"prompt":"brainstorm"}`), Out: &bytes.Buffer{}, Err: &bytes.Buffer{},
		ollamaClientSource: func() ollamaClient { return fake },
	}
	root := app.Root()
	root.SetArgs([]string{"ollama", "chat", "--model", "local:latest", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if fake.input.Context != "" || fake.input.Prompt != "brainstorm" {
		t.Fatalf("input = %#v", fake.input)
	}
}

func TestOllamaChatReadsCompleteConversationFromStdin(t *testing.T) {
	fake := &fakeOllamaClient{result: ollama.ChatResult{Model: "local:latest", Content: "answer"}}
	app := App{
		In:  strings.NewReader(`{"context":"private note","messages":[{"role":"user","content":"first"},{"role":"assistant","content":"answer"},{"role":"user","content":"follow up"}]}`),
		Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, ollamaClientSource: func() ollamaClient { return fake },
	}
	root := app.Root()
	root.SetArgs([]string{"ollama", "chat", "--model", "local:latest", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if fake.input.Context != "private note" || fake.input.Prompt != "" || fake.input.Model != "local:latest" {
		t.Fatalf("input = %#v", fake.input)
	}
	want := []ollama.ChatMessage{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "answer"},
		{Role: "user", Content: "follow up"},
	}
	if len(fake.input.Messages) != len(want) {
		t.Fatalf("messages = %#v", fake.input.Messages)
	}
	for index := range want {
		if fake.input.Messages[index] != want[index] {
			t.Fatalf("messages = %#v", fake.input.Messages)
		}
	}
	if strings.Contains(strings.Join(root.Flags().Args(), " "), "private note") {
		t.Fatal("conversation leaked into command arguments")
	}
}

func TestOllamaSetDefaultAtomicallyPreservesConfig(t *testing.T) {
	configPath := writeOllamaTestConfig(t, "")
	var output bytes.Buffer
	app := App{In: strings.NewReader(""), Out: &output, Err: &bytes.Buffer{}}
	root := app.Root()
	root.SetArgs([]string{"--config", configPath, "ollama", "set-default", "llama3.2:latest", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "ollama_model: llama3.2:latest") || !strings.Contains(string(contents), "path: ") {
		t.Fatalf("config contents = %s", contents)
	}
}

func TestDecodeOllamaInputRejectsUnknownAndMultipleDocuments(t *testing.T) {
	for _, input := range []string{
		`{"context":"x","prompt":"y","unknown":true}`,
		"{\"context\":\"x\",\"prompt\":\"y\"}\n{}",
	} {
		if _, err := decodeOllamaInput(strings.NewReader(input)); err == nil {
			t.Fatalf("input %q unexpectedly succeeded", input)
		}
	}
}

func writeOllamaTestConfig(t *testing.T, model string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "config.yaml")
	settings := ""
	if model != "" {
		settings = "settings:\n  ollama_model: " + model + "\n"
	}
	contents := "version: 2\n" + settings + "sources:\n  - path: " + root + "\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
