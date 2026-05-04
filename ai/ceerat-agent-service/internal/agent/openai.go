package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type OpenAIClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	return &OpenAIClient{apiKey: apiKey, model: model, http: http.DefaultClient}
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Tools       []tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
	Temperature float64   `json:"temperature"`
}

type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type tool struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (c *OpenAIClient) Complete(ctx context.Context, messages []message, tools []tool) (message, error) {
	if c.apiKey == "" {
		return message{}, errors.New("OPENAI_API_KEY is not set")
	}

	body, err := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: 0.2,
	})
	if err != nil {
		return message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return message{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return message{}, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return message{}, fmt.Errorf("openai decode response: %w: %s", err, string(raw))
	}
	if out.Error != nil {
		return message{}, errors.New(out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return message{}, errors.New("openai returned no choices")
	}
	return out.Choices[0].Message, nil
}
