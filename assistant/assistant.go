// Package assistant uses GPT to generate answers shaped by the interviewee's profile.
package assistant

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// Suggestion is a question paired with a suggested answer.
type Suggestion struct {
	Question string
	Answer   string
}

// Answerer generates suggested answers for interview questions.
type Answerer interface {
	Answer(ctx context.Context, question string) (*Suggestion, error)
}

// Assistant maintains conversation context and generates profile-aware answers
// using the OpenAI API.
type Assistant struct {
	client       *openai.Client
	model        string
	systemPrompt string
	history      []openai.ChatCompletionMessage
	maxTurns     int
}

// New creates an Assistant. profileText and companyText may each be empty.
func New(apiKey, model, profileText, companyText string) *Assistant {
	prompt := buildSystemPrompt(profileText, companyText)
	return &Assistant{
		client:       openai.NewClient(apiKey),
		model:        model,
		systemPrompt: prompt,
		maxTurns:     20,
	}
}

func buildSystemPrompt(profileText, companyText string) string {
	prompt := basePrompt
	if strings.TrimSpace(profileText) != "" {
		prompt += fmt.Sprintf(profilePromptSuffix, strings.TrimSpace(profileText))
	}
	if strings.TrimSpace(companyText) != "" {
		prompt += fmt.Sprintf(companyPromptSuffix, strings.TrimSpace(companyText))
	}
	return prompt
}

// Answer generates a suggested answer for the given question text.
// Callers should check IsQuestion first to avoid unnecessary API calls.
func (a *Assistant) Answer(ctx context.Context, question string) (*Suggestion, error) {
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
		return nil, fmt.Errorf("OpenAI API (%s): %w", a.model, err)
	}

	answer := strings.TrimSpace(resp.Choices[0].Message.Content)

	a.history = append(a.history, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: answer,
	})

	return &Suggestion{
		Question: question,
		Answer:   answer,
	}, nil
}
