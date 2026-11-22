package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/arcana"
)

func TestHandleHealth(t *testing.T) {
	h := NewHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleAllArcana(t *testing.T) {
	h := NewHandler()
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

func TestHandleArcanaByNumber(t *testing.T) {
	h := NewHandler()

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
	h := NewHandler()

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
