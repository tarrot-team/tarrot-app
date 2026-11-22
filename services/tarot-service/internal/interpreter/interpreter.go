package interpreter

import (
	"context"
	"fmt"
	"strings"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/spread"
)

type Input struct {
	Spread   spread.Spread
	Question string
}

type Result struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

type Interpreter interface {
	InterpretSpread(ctx context.Context, input Input) (Result, error)
}

type FallbackInterpreter struct{}

func (FallbackInterpreter) InterpretSpread(_ context.Context, input Input) (Result, error) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Расклад «%s»", input.Spread.Title))
	if input.Question != "" {
		b.WriteString(fmt.Sprintf(" на вопрос: «%s»", input.Question))
	}
	b.WriteString(".\n\n")

	for _, c := range input.Spread.Cards {
		orientation := "прямое положение"
		if c.Reversed {
			orientation = "перевёрнутое положение"
		}
		b.WriteString(fmt.Sprintf("- %s: %s (%s) — %s.\n", c.Position, c.Name, orientation, c.ShortDesc))
	}

	return Result{
		Text:   strings.TrimSpace(b.String()),
		Source: "fallback",
	}, nil
}

type WithFallback struct {
	Primary  Interpreter
	Fallback Interpreter
}

func (w WithFallback) InterpretSpread(ctx context.Context, input Input) (Result, error) {
	if w.Primary != nil {
		res, err := w.Primary.InterpretSpread(ctx, input)
		if err == nil {
			return res, nil
		}
	}
	return w.Fallback.InterpretSpread(ctx, input)
}
