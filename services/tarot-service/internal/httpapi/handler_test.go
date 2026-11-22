package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tarrot-team/tarrot-app/pkg/auth"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/arcana"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/deck"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/interpreter"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/reading"
)

var testJWTSecret = []byte("test-jwt-secret")

type mockInterpreter struct {
	text string
}

func (m mockInterpreter) InterpretSpread(_ context.Context, _ interpreter.Input) (interpreter.Result, error) {
	return interpreter.Result{Text: m.text, Source: "ai"}, nil
}

func (m mockInterpreter) InterpretMatrix(_ context.Context, _ interpreter.MatrixInput) (interpreter.Result, error) {
	return interpreter.Result{Text: m.text, Source: "ai"}, nil
}

func bearerFor(t *testing.T, userID string) string {
	t.Helper()
	tok, err := auth.Sign(testJWTSecret, auth.Claims{Sub: userID, Role: "user"})
	if err != nil {
		t.Fatalf("auth.Sign: %v", err)
	}
	return "Bearer " + tok
}

func TestHandleHealth(t *testing.T) {
	h := NewHandler(nil, nil, testJWTSecret)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleAllArcana(t *testing.T) {
	h := NewHandler(nil, nil, testJWTSecret)
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
	h := NewHandler(nil, nil, testJWTSecret)
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
	h := NewHandler(nil, nil, testJWTSecret)

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
	h := NewHandler(nil, nil, testJWTSecret)

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
	h := NewHandler(mockInterpreter{text: "Тестовая трактовка"}, reading.NewMemoryStore(), testJWTSecret)

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
		if got.ID != "" {
			t.Errorf("expected empty id without jwt, got %q", got.ID)
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

func TestSpreadHistoryFlow(t *testing.T) {
	h := NewHandler(mockInterpreter{text: "Сохранённая трактовка"}, reading.NewMemoryStore(), testJWTSecret)
	user1Auth := bearerFor(t, "user-1")
	user2Auth := bearerFor(t, "user-2")

	createReq := httptest.NewRequest(http.MethodPost, "/spread", bytes.NewBufferString(`{"type":"daily","question":"Как пройдёт день?"}`))
	createReq.Header.Set("Authorization", user1Auth)
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d; want 200", createRec.Code)
	}
	var created spreadResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID == "" || created.CreatedAt == nil {
		t.Fatalf("expected id and created_at, got %+v", created)
	}

	spoofedList := httptest.NewRequest(http.MethodGet, "/spreads", nil)
	spoofedList.Header.Set("X-User-ID", "user-1")
	spoofedRec := httptest.NewRecorder()
	h.ServeHTTP(spoofedRec, spoofedList)
	if spoofedRec.Code != http.StatusUnauthorized {
		t.Fatalf("spoofed X-User-ID list status = %d; want 401", spoofedRec.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/spreads", nil)
	listReq.Header.Set("Authorization", user1Auth)
	listRec := httptest.NewRecorder()
	h.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d; want 200", listRec.Code)
	}
	var items []reading.Reading
	if err := json.NewDecoder(listRec.Body).Decode(&items); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("unexpected list: %+v", items)
	}

	guestGetRec := httptest.NewRecorder()
	h.ServeHTTP(guestGetRec, httptest.NewRequest(http.MethodGet, "/spreads/"+created.ID, nil))
	if guestGetRec.Code != http.StatusUnauthorized {
		t.Fatalf("guest get status = %d; want 401", guestGetRec.Code)
	}

	otherGetReq := httptest.NewRequest(http.MethodGet, "/spreads/"+created.ID, nil)
	otherGetReq.Header.Set("Authorization", user2Auth)
	otherGetRec := httptest.NewRecorder()
	h.ServeHTTP(otherGetRec, otherGetReq)
	if otherGetRec.Code != http.StatusForbidden {
		t.Fatalf("other user get status = %d; want 403", otherGetRec.Code)
	}

	ownerGetReq := httptest.NewRequest(http.MethodGet, "/spreads/"+created.ID, nil)
	ownerGetReq.Header.Set("Authorization", user1Auth)
	ownerGetRec := httptest.NewRecorder()
	h.ServeHTTP(ownerGetRec, ownerGetReq)
	if ownerGetRec.Code != http.StatusOK {
		t.Fatalf("owner get status = %d; want 200", ownerGetRec.Code)
	}

	forbidDel := httptest.NewRequest(http.MethodDelete, "/spreads/"+created.ID, nil)
	forbidDel.Header.Set("Authorization", user2Auth)
	forbidRec := httptest.NewRecorder()
	h.ServeHTTP(forbidRec, forbidDel)
	if forbidRec.Code != http.StatusForbidden {
		t.Fatalf("forbidden delete status = %d; want 403", forbidRec.Code)
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/spreads/"+created.ID, nil)
	delReq.Header.Set("Authorization", user1Auth)
	delRec := httptest.NewRecorder()
	h.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d; want 204", delRec.Code)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/spreads/"+created.ID, nil)
	missingReq.Header.Set("Authorization", user1Auth)
	missingRec := httptest.NewRecorder()
	h.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("get deleted status = %d; want 404", missingRec.Code)
	}
}

func TestHandleMatrixInterpret(t *testing.T) {
	h := NewHandler(mockInterpreter{text: "Разбор матрицы"}, nil, testJWTSecret)

	t.Run("valid request", func(t *testing.T) {
		body := bytes.NewBufferString(`{"birth_date":"1997-01-21","focus":"money"}`)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/matrix/interpret", body))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; want 200", rec.Code)
		}
		var got matrixInterpretResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Focus != "money" || got.Source != "ai" || got.Interpretation != "Разбор матрицы" {
			t.Errorf("unexpected response: %+v", got)
		}
	})

	t.Run("invalid focus", func(t *testing.T) {
		body := bytes.NewBufferString(`{"birth_date":"1997-01-21","focus":"health"}`)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/matrix/interpret", body))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want 400", rec.Code)
		}
	})
}
