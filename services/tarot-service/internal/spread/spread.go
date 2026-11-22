package spread

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/deck"
)

const maxManualCards = 10

type PositionCard struct {
	Position string `json:"position"`
	deck.DrawnCard
}

type ManualCardInput struct {
	CardID   string `json:"card_id"`
	Reversed bool   `json:"reversed"`
	Position string `json:"position,omitempty"`
}

type Spread struct {
	Type  string         `json:"type"`
	Title string         `json:"title"`
	Cards []PositionCard `json:"cards"`
}

var (
	ErrUnknownType      = errors.New("unknown spread type: supported are daily, three_cards, love, custom")
	ErrInvalidCardCount = errors.New("invalid number of cards for this spread type")
	ErrDuplicateCard    = errors.New("duplicate card in spread")
)

type spreadDef struct {
	title     string
	positions []string
}

var definitions = map[string]spreadDef{
	"daily": {
		title:     "Карта дня",
		positions: []string{"Карта дня"},
	},
	"three_cards": {
		title:     "Прошлое / Настоящее / Будущее",
		positions: []string{"Прошлое", "Настоящее", "Будущее"},
	},
	"love": {
		title:     "Отношения",
		positions: []string{"Ты", "Партнёр", "Перспектива"},
	},
}

func Draw(spreadType string) (Spread, error) {
	def, ok := definitions[spreadType]
	if !ok {
		return Spread{}, ErrUnknownType
	}

	drawn, err := deck.Draw(len(def.positions))
	if err != nil {
		return Spread{}, err
	}

	cards := make([]PositionCard, len(def.positions))
	for i, pos := range def.positions {
		cards[i] = PositionCard{
			Position:  pos,
			DrawnCard: drawn[i],
		}
	}

	return Spread{
		Type:  spreadType,
		Title: def.title,
		Cards: cards,
	}, nil
}

func BuildManual(spreadType string, inputs []ManualCardInput) (Spread, error) {
	if len(inputs) == 0 || len(inputs) > maxManualCards {
		return Spread{}, ErrInvalidCardCount
	}

	var title string
	var defaultPositions []string

	if spreadType == "custom" {
		title = "Свободный расклад"
	} else {
		def, ok := definitions[spreadType]
		if !ok {
			return Spread{}, ErrUnknownType
		}
		if len(inputs) != len(def.positions) {
			return Spread{}, ErrInvalidCardCount
		}
		title = def.title
		defaultPositions = def.positions
	}

	seen := make(map[string]bool, len(inputs))
	cards := make([]PositionCard, len(inputs))

	for i, in := range inputs {
		id := strings.TrimSpace(in.CardID)
		if seen[id] {
			return Spread{}, ErrDuplicateCard
		}
		seen[id] = true

		card, err := deck.GetByID(id)
		if err != nil {
			return Spread{}, err
		}

		pos := strings.TrimSpace(in.Position)
		if pos == "" {
			if i < len(defaultPositions) {
				pos = defaultPositions[i]
			} else {
				pos = fmt.Sprintf("Карта %d", i+1)
			}
		}

		cards[i] = PositionCard{
			Position: pos,
			DrawnCard: deck.DrawnCard{
				Card:     card,
				Reversed: in.Reversed,
			},
		}
	}

	return Spread{
		Type:  spreadType,
		Title: title,
		Cards: cards,
	}, nil
}
