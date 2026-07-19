package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListModelsFiltersCloudAndSortsLocalArtifacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/tags" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"models":[
			{"name":"zeta:latest","size":10,"details":{"format":"gguf"}},
			{"name":"remote:cloud","size":340,"details":{"format":""}},
			{"name":"metadata-only","size":1,"details":{"format":""}},
			{"name":"alpha:latest","size":20,"details":{"format":"gguf"}}
		]}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].Name != "alpha:latest" || models[1].Name != "zeta:latest" {
		t.Fatalf("models = %#v", models)
	}
}

func TestChatValidatesModelAndSendsBoundedNonStreamingMessages(t *testing.T) {
	var requestBody struct {
		Model     string    `json:"model"`
		Messages  []message `json:"messages"`
		Stream    bool      `json:"stream"`
		Think     bool      `json:"think"`
		KeepAlive string    `json:"keep_alive"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/tags":
			_, _ = writer.Write([]byte(`{"models":[{"name":"local:latest","size":42,"details":{"format":"gguf"}}]}`))
		case "/api/chat":
			if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
				t.Fatal(err)
			}
			_, _ = writer.Write([]byte(`{"model":"local:latest","message":{"role":"assistant","content":"  useful answer  "}}`))
		default:
			t.Fatalf("path = %q", request.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Chat(context.Background(), ChatInput{
		Model: "local:latest", Context: "selected secret", Prompt: "summarize it",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "useful answer" || result.Model != "local:latest" {
		t.Fatalf("result = %#v", result)
	}
	if requestBody.Stream || requestBody.Think || requestBody.KeepAlive != "5m" {
		t.Fatalf("request options = %#v", requestBody)
	}
	if len(requestBody.Messages) != 2 || !strings.Contains(requestBody.Messages[1].Content, "selected secret") || !strings.Contains(requestBody.Messages[1].Content, "summarize it") {
		t.Fatalf("messages = %#v", requestBody.Messages)
	}
}

func TestChatAllowsPromptWithoutNotesContext(t *testing.T) {
	var userMessage string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/tags":
			_, _ = writer.Write([]byte(`{"models":[{"name":"local:latest","size":42,"details":{"format":"gguf"}}]}`))
		case "/api/chat":
			var body struct {
				Messages []message `json:"messages"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			userMessage = body.Messages[1].Content
			_, _ = writer.Write([]byte(`{"model":"local:latest","message":{"role":"assistant","content":"answer"}}`))
		default:
			t.Fatalf("path = %q", request.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Chat(context.Background(), ChatInput{
		Model: "local:latest", Prompt: "brainstorm next steps",
	}); err != nil {
		t.Fatal(err)
	}
	if userMessage != "User request:\nbrainstorm next steps" {
		t.Fatalf("user message = %q", userMessage)
	}
}

func TestChatRejectsUnavailableAndOversizedInputBeforeGeneration(t *testing.T) {
	chatCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/chat" {
			chatCalls++
		}
		_, _ = writer.Write([]byte(`{"models":[{"name":"local:latest","size":42,"details":{"format":"gguf"}}]}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []ChatInput{
		{Model: "missing:latest", Context: "context", Prompt: "question"},
		{Model: "local:latest", Context: strings.Repeat("x", MaxContextBytes+1), Prompt: "question"},
		{Model: "local:latest", Context: "context", Prompt: "  "},
	} {
		if _, err := client.Chat(context.Background(), input); err == nil {
			t.Fatalf("input %#v unexpectedly succeeded", input)
		}
	}
	if chatCalls != 0 {
		t.Fatalf("chat calls = %d", chatCalls)
	}
}

func TestClientReportsOllamaAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte(`{"error":"model runner unavailable"}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListModels(context.Background())
	if err == nil || !strings.Contains(err.Error(), "model runner unavailable") {
		t.Fatalf("error = %v", err)
	}
}
