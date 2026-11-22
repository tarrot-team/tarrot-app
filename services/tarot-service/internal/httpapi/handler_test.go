package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/arcana"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/deck"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/interpreter"
)

type mockInterpreter struct {
	text string
}

func (m mockInterpreter) InterpretSpread(_ context.Context, _ interpreter.Input) (interpreter.Result, error) {
	return interpreter.Result{Text: m.text, Source: "ai"}, nil
}

func TestHandleHealth(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleAllArcana(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/arcana", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}

	var got []arcana.Arcana
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != 22 {
		t.Fatalf("len(arcana) = %d; want 22", len(got))
	}
}

func TestHandleDeck(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/deck", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}

	var got []deck.Card
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != 78 {
		t.Fatalf("len(deck) = %d; want 78", len(got))
	}
}

func TestHandleArcanaByNumber(t *testing.T) {
	h := NewHandler(nil)

	t.Run("valid number", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/arcana/8", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
		}

		var got arcana.Arcana
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got.Number != 8 || got.Name != "Справедливость (Justice)" {
			t.Errorf("got = %+v; want Justice (8)", got)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/arcana/99", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("not a number", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/arcana/abc", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestHandleMatrix(t *testing.T) {
	h := NewHandler(nil)

	t.Run("valid birth date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/matrix?birth_date=1999-12-29", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
		}

		var got matrixResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got.BirthDate != "1999-12-29" {
			t.Errorf("birth_date = %q; want 1999-12-29", got.BirthDate)
		}
		if got.Matrix.A != 11 || got.Matrix.G != 22 || got.Matrix.M != 18 {
			t.Errorf("unexpected matrix: %+v", got.Matrix)
		}
		if _, ok := got.Arcana["22"]; !ok {
			t.Errorf("expected arcana 22 in response map")
		}
	})

	t.Run("missing birth date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/matrix", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/matrix?birth_date=29.12.1999", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestHandleSpread(t *testing.T) {
	h := NewHandler(mockInterpreter{text: "Тестовая трактовка"})

	t.Run("valid three cards", func(t *testing.T) {
		body := bytes.NewBufferString(`{"type":"three_cards","question":"Что будет на работе?"}`)
		req := httptest.NewRequest(http.MethodPost, "/spread", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
		}

		var got spreadResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(got.Spread.Cards) != 3 {
			t.Errorf("cards = %d; want 3", len(got.Spread.Cards))
		}
		if got.Interpretation != "Тестовая трактовка" || got.Source != "ai" {
			t.Errorf("unexpected interpretation: %+v", got)
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		body := bytes.NewBufferString(`{"type":"unknown"}`)
		req := httptest.NewRequest(http.MethodPost, "/spread", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("too long question", func(t *testing.T) {
		longQ := strings.Repeat("а", 501)
		body := bytes.NewBufferString(`{"type":"daily","question":"` + longQ + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/spread", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("manual physical deck cards", func(t *testing.T) {
		body := bytes.NewBufferString(`{"type":"custom","question":"Стоит ли менять работу?","cards":[{"card_id":"major_1","reversed":false,"position":"Сейчас"},{"card_id":"minor_pentacles_10","reversed":true}]}`)
		req := httptest.NewRequest(http.MethodPost, "/spread", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
		}

		var got spreadResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got.Mode != "manual" || len(got.Spread.Cards) != 2 {
			t.Errorf("unexpected response: %+v", got)
		}
		if got.Spread.Cards[0].ID != "major_1" || !got.Spread.Cards[1].Reversed {
			t.Errorf("unexpected manual cards: %+v", got.Spread.Cards)
		}
	})

	t.Run("manual duplicate card", func(t *testing.T) {
		body := bytes.NewBufferString(`{"type":"custom","cards":[{"card_id":"major_1"},{"card_id":"major_1"}]}`)
		req := httptest.NewRequest(http.MethodPost, "/spread", body)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
