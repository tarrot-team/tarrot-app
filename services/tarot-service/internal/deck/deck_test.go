package deck

import (
	"errors"
	"testing"
)

func TestAll(t *testing.T) {
	cards := All()
	if len(cards) != 78 {
		t.Fatalf("len(All()) = %d; want 78", len(cards))
	}

	seen := make(map[string]bool)
	majorCount := 0
	minorCount := 0
	for _, c := range cards {
		if seen[c.ID] {
			t.Fatalf("duplicate card ID: %s", c.ID)
		}
		seen[c.ID] = true
		if c.Type == "major" {
			majorCount++
		} else if c.Type == "minor" {
			minorCount++
		}
	}

	if majorCount != 22 || minorCount != 56 {
		t.Fatalf("major = %d, minor = %d; want 22 and 56", majorCount, minorCount)
	}
}

func TestDraw(t *testing.T) {
	cards, err := Draw(10)
	if err != nil {
		t.Fatalf("Draw(10) error: %v", err)
	}
	if len(cards) != 10 {
		t.Fatalf("len(Draw(10)) = %d; want 10", len(cards))
	}

	seen := make(map[string]bool)
	for _, c := range cards {
		if seen[c.ID] {
			t.Fatalf("duplicate drawn card: %s", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestDrawInvalid(t *testing.T) {
	for _, count := range []int{-1, 0, 79} {
		_, err := Draw(count)
		if !errors.Is(err, ErrInvalidCount) {
			t.Errorf("Draw(%d) error = %v; want ErrInvalidCount", count, err)
		}
	}
}

func TestGetByID(t *testing.T) {
	c, err := GetByID("minor_cups_13")
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if c.Name != "Королева Кубков (Queen of Cups)" {
		t.Errorf("name = %q; want Королева Кубков (Queen of Cups)", c.Name)
	}

	_, err = GetByID("unknown_card")
	if !errors.Is(err, ErrCardNotFound) {
		t.Errorf("error = %v; want ErrCardNotFound", err)
	}
}
