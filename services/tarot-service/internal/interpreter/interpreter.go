package interpreter

import (
	"context"
	"fmt"
	"strings"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/arcana"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/matrix"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/spread"
)

type Input struct {
	Spread   spread.Spread
	Question string
}

type MatrixInput struct {
	BirthDate string
	Focus     string
	Matrix    matrix.Matrix
}

type Result struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

type Interpreter interface {
	InterpretSpread(ctx context.Context, input Input) (Result, error)
	InterpretMatrix(ctx context.Context, input MatrixInput) (Result, error)
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

func (FallbackInterpreter) InterpretMatrix(_ context.Context, input MatrixInput) (Result, error) {
	m := input.Matrix
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Разбор матрицы судьбы для даты %s:\n\n", input.BirthDate))

	if e, err := arcana.Get(m.E); err == nil {
		b.WriteString(fmt.Sprintf("Зона комфорта (центр %d — %s):\n+ В плюсе: %s.\n- В минусе: %s.\n\n",
			e.Number, e.Name, e.Plus, e.Minus))
	}
	if love, err := arcana.Get(m.LoveMoney.Love); err == nil {
		b.WriteString(fmt.Sprintf("Линия отношений (%d — %s): %s.\n\n",
			love.Number, love.Name, love.Love))
	}
	if money, err := arcana.Get(m.LoveMoney.Money); err == nil {
		b.WriteString(fmt.Sprintf("Линия финансов (%d — %s): %s.\n\n",
			money.Number, money.Name, money.Money))
	}
	b.WriteString(fmt.Sprintf("Кармический хвост: %d-%d-%d.\n", m.KarmicTail[0], m.KarmicTail[1], m.KarmicTail[2]))
	b.WriteString(fmt.Sprintf("Предназначение: личное %d, социальное %d, духовное %d.",
		m.Purpose.Personal, m.Purpose.Social, m.Purpose.Spiritual))

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

func (w WithFallback) InterpretMatrix(ctx context.Context, input MatrixInput) (Result, error) {
	if w.Primary != nil {
		res, err := w.Primary.InterpretMatrix(ctx, input)
		if err == nil {
			return res, nil
		}
	}
	return w.Fallback.InterpretMatrix(ctx, input)
}
