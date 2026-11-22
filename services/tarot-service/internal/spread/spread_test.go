package spread

import (
	"errors"
	"testing"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/deck"
)

func TestDraw(t *testing.T) {
	tests := []struct {
		spreadType string
		wantCount  int
	}{
		{spreadType: "daily", wantCount: 1},
		{spreadType: "three_cards", wantCount: 3},
		{spreadType: "love", wantCount: 3},
	}

	for _, tc := range tests {
		got, err := Draw(tc.spreadType)
		if err != nil {
			t.Fatalf("Draw(%q) error: %v", tc.spreadType, err)
		}
		if got.Type != tc.spreadType || len(got.Cards) != tc.wantCount {
			t.Errorf("Draw(%q) = {type: %q, cards: %d}; want %d cards",
				tc.spreadType, got.Type, len(got.Cards), tc.wantCount)
		}
	}
}

func TestDrawUnknown(t *testing.T) {
	_, err := Draw("celtic_cross")
	if !errors.Is(err, ErrUnknownType) {
		t.Fatalf("error = %v; want ErrUnknownType", err)
	}
}

func TestBuildManualPreset(t *testing.T) {
	got, err := BuildManual("three_cards", []ManualCardInput{
		{CardID: "major_1", Reversed: false},
		{CardID: "minor_cups_1", Reversed: true},
		{CardID: "major_21", Reversed: false},
	})
	if err != nil {
		t.Fatalf("BuildManual error: %v", err)
	}
	if len(got.Cards) != 3 {
		t.Fatalf("len(cards) = %d; want 3", len(got.Cards))
	}
	if got.Cards[0].Position != "Прошлое" || !got.Cards[1].Reversed || got.Cards[2].ID != "major_21" {
		t.Errorf("unexpected cards: %+v", got.Cards)
	}
}

func TestBuildManualCustom(t *testing.T) {
	got, err := BuildManual("custom", []ManualCardInput{
		{CardID: "major_8", Position: "Суть ситуации"},
		{CardID: "minor_swords_10", Reversed: true},
	})
	if err != nil {
		t.Fatalf("BuildManual custom error: %v", err)
	}
	if got.Cards[0].Position != "Суть ситуации" || got.Cards[1].Position != "Карта 2" {
		t.Errorf("unexpected positions: %+v", got.Cards)
	}
}

func TestBuildManualErrors(t *testing.T) {
	_, err := BuildManual("three_cards", []ManualCardInput{{CardID: "major_1"}})
	if !errors.Is(err, ErrInvalidCardCount) {
		t.Errorf("wrong count error = %v; want ErrInvalidCardCount", err)
	}

	_, err = BuildManual("custom", []ManualCardInput{
		{CardID: "major_1"},
		{CardID: "major_1"},
	})
	if !errors.Is(err, ErrDuplicateCard) {
		t.Errorf("duplicate error = %v; want ErrDuplicateCard", err)
	}

	_, err = BuildManual("custom", []ManualCardInput{{CardID: "nope"}})
	if !errors.Is(err, deck.ErrCardNotFound) {
		t.Errorf("missing card error = %v; want deck.ErrCardNotFound", err)
	}
}
