package assistant

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const defaultOllamaURL = "http://localhost:11434/v1"
const defaultLocalModel = "llama3"

// LocalAssistant generates answers using a local Ollama model via its
// OpenAI-compatible API endpoint.
type LocalAssistant struct {
	client       *openai.Client
	model        string
	systemPrompt string
	history      []openai.ChatCompletionMessage
	maxTurns     int
}

// EnsureOllamaModel starts Ollama if not running, then pulls the model if absent.
func EnsureOllamaModel(model string) error {
	if _, err := exec.LookPath("ollama"); err != nil {
		return fmt.Errorf("ollama not found in PATH — install it from https://ollama.com")
	}

	if err := ensureOllamaRunning(); err != nil {
		return err
	}

	// `ollama show` exits non-zero when the model is absent.
	check := exec.Command("ollama", "show", model)
	check.Stderr = nil
	if check.Run() == nil {
		return nil // already present
	}

	fmt.Printf("Model %q not found locally. Pulling via ollama (this may take a while)…\n", model)
	pull := exec.Command("ollama", "pull", model)
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		return fmt.Errorf("ollama pull %s: %w", model, err)
	}
	return nil
}

// ensureOllamaRunning checks if Ollama is responding and starts it if not.
func ensureOllamaRunning() error {
	if ollamaResponding() {
		return nil
	}

	fmt.Println("Ollama not running. Starting it…")
	cmd := exec.Command("ollama", "serve")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama: %w", err)
	}

	// Wait up to 10 seconds for the server to become ready.
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if ollamaResponding() {
			fmt.Println("Ollama started.")
			return nil
		}
	}
	return fmt.Errorf("ollama did not become ready in time")
}

func ollamaResponding() bool {
	c := http.Client{Timeout: 500 * time.Millisecond}
	resp, err := c.Get("http://localhost:11434/")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// NewLocal creates a LocalAssistant. baseURL defaults to the standard Ollama
// endpoint; model defaults to "llama3".
func NewLocal(model, profileText, companyText, baseURL string) *LocalAssistant {
	if baseURL == "" {
		baseURL = defaultOllamaURL
	}
	if model == "" {
		model = defaultLocalModel
	}
	// Ollama's API requires a tag (e.g. "llama3:latest"); add one if absent.
	if !strings.Contains(model, ":") {
		model += ":latest"
	}
	cfg := openai.DefaultConfig("ollama") // Ollama ignores the API key
	cfg.BaseURL = baseURL

	prompt := buildSystemPrompt(profileText, companyText)
	return &LocalAssistant{
		client:       openai.NewClientWithConfig(cfg),
		model:        model,
		systemPrompt: prompt,
		maxTurns:     20,
	}
}

// Answer generates a suggested answer using the local Ollama model.
func (a *LocalAssistant) Answer(ctx context.Context, question string) (*Suggestion, error) {
	a.history = append(a.history, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: question,
	})
	if len(a.history) > a.maxTurns*2 {
		a.history = a.history[len(a.history)-a.maxTurns*2:]
	}

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: a.systemPrompt},
	}
	messages = append(messages, a.history...)

	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     a.model,
		Messages:  messages,
		MaxTokens: 150,
	})
	if err != nil {
		return nil, fmt.Errorf("local model (%s): %w", a.model, err)
	}

	answer := strings.TrimSpace(resp.Choices[0].Message.Content)
	a.history = append(a.history, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: answer,
	})
	return &Suggestion{Question: question, Answer: answer}, nil
}
