package interpreter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/deck"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/matrix"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/spread"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func sampleInput() Input {
	return Input{
		Question: "Что ждёт меня в новом проекте?",
		Spread: spread.Spread{
			Type:  "daily",
			Title: "Карта дня",
			Cards: []spread.PositionCard{
				{
					Position: "Карта дня",
					DrawnCard: deck.DrawnCard{
						Card: deck.Card{
							ID:        "major_1",
							Type:      "major",
							Number:    1,
							Name:      "Маг (The Magician)",
							ShortDesc: "Воля, инициатива, мастерство и реализация идей",
						},
						Reversed: false,
					},
				},
			},
		},
	}
}

func sampleMatrixInput(t *testing.T) MatrixInput {
	t.Helper()
	m, err := matrix.Calculate(time.Date(1997, 1, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("matrix.Calculate error: %v", err)
	}
	return MatrixInput{
		BirthDate: "1997-01-21",
		Focus:     "general",
		Matrix:    m,
	}
}

func TestFallbackInterpreter(t *testing.T) {
	res, err := (FallbackInterpreter{}).InterpretSpread(context.Background(), sampleInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != "fallback" {
		t.Errorf("source = %q; want fallback", res.Source)
	}
	if !strings.Contains(res.Text, "Маг (The Magician)") {
		t.Errorf("expected card name in text, got: %s", res.Text)
	}
}

func TestFallbackInterpretMatrix(t *testing.T) {
	res, err := (FallbackInterpreter{}).InterpretMatrix(context.Background(), sampleMatrixInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != "fallback" {
		t.Errorf("source = %q; want fallback", res.Source)
	}
	if !strings.Contains(res.Text, "3-12-9") || !strings.Contains(res.Text, "Солнце (The Sun)") {
		t.Errorf("unexpected matrix fallback text: %s", res.Text)
	}
}

func TestOpenAICompatClient(t *testing.T) {
	client := NewOpenAICompatClient("https://api.groq.com/openai/v1", "test-key", "llama-3.3-70b-versatile")
	client.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://api.groq.com/openai/v1/chat/completions" {
				t.Errorf("url = %q; want groq chat/completions", r.URL.String())
			}
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("auth header = %q; want Bearer test-key", r.Header.Get("Authorization"))
			}

			body, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]string{"role": "assistant", "content": "Отличное время для старта."}},
				},
			})

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		}),
	}

	res, err := client.InterpretSpread(context.Background(), sampleInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != "ai" || res.Text != "Отличное время для старта." {
		t.Errorf("got = %+v; want ai response", res)
	}

	mres, err := client.InterpretMatrix(context.Background(), sampleMatrixInput(t))
	if err != nil {
		t.Fatalf("unexpected matrix error: %v", err)
	}
	if mres.Source != "ai" || mres.Text != "Отличное время для старта." {
		t.Errorf("got matrix = %+v; want ai response", mres)
	}
}

func TestWithFallback(t *testing.T) {
	client := NewOpenAICompatClient("https://api.groq.com/openai/v1", "key", "model")
	client.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("rate limit")),
			}, nil
		}),
	}

	interp := WithFallback{
		Primary:  client,
		Fallback: FallbackInterpreter{},
	}

	res, err := interp.InterpretSpread(context.Background(), sampleInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != "fallback" {
		t.Errorf("source = %q; want fallback", res.Source)
	}

	mres, err := interp.InterpretMatrix(context.Background(), sampleMatrixInput(t))
	if err != nil {
		t.Fatalf("unexpected matrix error: %v", err)
	}
	if mres.Source != "fallback" {
		t.Errorf("matrix source = %q; want fallback", mres.Source)
	}
}
