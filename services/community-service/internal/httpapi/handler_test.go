package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tarrot-team/tarrot-app/services/community-service/internal/profile"
)

func newTestHandler() http.Handler {
	store := profile.NewMemoryStore()
	svc := profile.NewService(store)
	return NewHandler(svc, nil)
}

func TestProfileLifecycle(t *testing.T) {
	h := newTestHandler()

	body := `{"username":"tarot_queen","bio":"читаю расклады","birth_date":"2007-09-05"}`
	req := httptest.NewRequest(http.MethodPost, "/profiles", bytes.NewBufferString(body))
	req.Header.Set("X-User-ID", "u-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created profile.Profile
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Username != "tarot_queen" || created.Level != 1 {
		t.Fatalf("unexpected profile: %+v", created)
	}

	dupBody := `{"username":"TAROT_QUEEN"}`
	reqDup := httptest.NewRequest(http.MethodPost, "/profiles", bytes.NewBufferString(dupBody))
	reqDup.Header.Set("X-User-ID", "u-2")
	recDup := httptest.NewRecorder()
	h.ServeHTTP(recDup, reqDup)

	if recDup.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recDup.Code)
	}

	patchBody := `{"bio":"взломал"}`
	reqForbidden := httptest.NewRequest(http.MethodPatch, "/profiles/u-1", bytes.NewBufferString(patchBody))
	reqForbidden.Header.Set("X-User-ID", "u-2")
	recForbidden := httptest.NewRecorder()
	h.ServeHTTP(recForbidden, reqForbidden)

	if recForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recForbidden.Code)
	}

	reqPatch := httptest.NewRequest(http.MethodPatch, "/profiles/u-1", bytes.NewBufferString(`{"bio":"обновлено"}`))
	reqPatch.Header.Set("X-User-ID", "u-1")
	recPatch := httptest.NewRecorder()
	h.ServeHTTP(recPatch, reqPatch)

	if recPatch.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recPatch.Code, recPatch.Body.String())
	}
}

func TestInvalidBirthDate(t *testing.T) {
	h := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/profiles", bytes.NewBufferString(`{"username":"anya_go","birth_date":"05-09-2007"}`))
	req.Header.Set("X-User-ID", "u-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}