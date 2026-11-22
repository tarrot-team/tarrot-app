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

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/arcana"
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
	return c.complete(ctx, systemPrompt, buildUserPrompt(input), 800)
}

func (c *OpenAICompatClient) InterpretMatrix(ctx context.Context, input MatrixInput) (Result, error) {
	systemPrompt := "Ты специалист по матрице судьбы (метод Ладини, 22 аркана в марсельском порядке). Дай понятный, тёплый и структурированный разбор на русском языке: опиши сильные стороны (плюс), зоны роста (минус) и практичные рекомендации без фатализма и без медицинских или финансовых гарантий."
	return c.complete(ctx, systemPrompt, buildMatrixPrompt(input), 1000)
}

func (c *OpenAICompatClient) complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (Result, error) {
	payload := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   maxTokens,
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

func buildMatrixPrompt(input MatrixInput) string {
	m := input.Matrix
	focusLabels := map[string]string{
		"general": "общий разбор (визитная карточка, зона комфорта, отношения, финансы, кармический хвост, предназначение)",
		"love":    "фокус на отношениях и линии любви",
		"money":   "фокус на финансах, карьере и реализации",
		"karmic":  "фокус на кармическом хвосте и точках роста",
		"purpose": "фокус на личном, социальном и духовном предназначении",
	}
	focusText := focusLabels[input.Focus]
	if focusText == "" {
		focusText = focusLabels["general"]
	}

	nameOf := func(n int) string {
		if a, err := arcana.Get(n); err == nil {
			return fmt.Sprintf("%d (%s)", n, a.Name)
		}
		return fmt.Sprintf("%d", n)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Дата рождения: %s\n", input.BirthDate))
	b.WriteString(fmt.Sprintf("Фокус разбора: %s\n", focusText))
	b.WriteString(fmt.Sprintf("Личный квадрат: A=%s, B=%s, C=%s, D=%s\n", nameOf(m.A), nameOf(m.B), nameOf(m.C), nameOf(m.D)))
	b.WriteString(fmt.Sprintf("Зона комфорта (центр E): %s\n", nameOf(m.E)))
	b.WriteString(fmt.Sprintf("Родовой квадрат: F=%s, G=%s, H=%s, I=%s\n", nameOf(m.F), nameOf(m.G), nameOf(m.H), nameOf(m.I)))
	b.WriteString(fmt.Sprintf("Кармический хвост: %d-%d-%d\n", m.KarmicTail[0], m.KarmicTail[1], m.KarmicTail[2]))
	b.WriteString(fmt.Sprintf("Линия отношений и денег: вход K=%s, вход L=%s, центр=%s, любовь=%s, деньги=%s\n",
		nameOf(m.LoveMoney.K), nameOf(m.LoveMoney.L), nameOf(m.LoveMoney.Center), nameOf(m.LoveMoney.Love), nameOf(m.LoveMoney.Money)))
	b.WriteString(fmt.Sprintf("Предназначение: личное=%s, социальное=%s, духовное=%s\n",
		nameOf(m.Purpose.Personal), nameOf(m.Purpose.Social), nameOf(m.Purpose.Spiritual)))
	return b.String()
}
