package interpreter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAICompatClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func NewOpenAICompatClient(baseURL, apiKey, model string) *OpenAICompatClient {
	return &OpenAICompatClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *OpenAICompatClient) InterpretSpread(ctx context.Context, input Input) (Result, error) {
	systemPrompt := "Ты опытный и бережный таролог. Дай структурированную, понятную интерпретацию расклада на русском языке: кратко разбери каждую карту в её позиции с учётом прямого или перевёрнутого положения, затем сделай общий вывод и мягкий совет. Не давай медицинских, юридических или финансовых гарантий."
	userPrompt := buildUserPrompt(input)

	payload := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   800,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Result{}, err
	}

	endpoint := c.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Result{}, fmt.Errorf("llm api status %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Result{}, err
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return Result{}, errors.New("llm returned empty response")
	}

	return Result{
		Text:   strings.TrimSpace(parsed.Choices[0].Message.Content),
		Source: "ai",
	}, nil
}

func buildUserPrompt(input Input) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Расклад: %s\n", input.Spread.Title))
	if input.Question != "" {
		b.WriteString(fmt.Sprintf("Вопрос пользователя: %s\n", input.Question))
	}
	b.WriteString("Выпавшие карты:\n")
	for _, c := range input.Spread.Cards {
		orientation := "прямая"
		if c.Reversed {
			orientation = "перевёрнутая"
		}
		b.WriteString(fmt.Sprintf("- %s: %s (%s)\n", c.Position, c.Name, orientation))
	}
	return b.String()
}
