package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/reading"
)

func TestReadingRejectsUnverifiedUserID(t *testing.T) {
	store := reading.NewMemoryStore()
	err := store.Save(context.Background(), reading.Reading{
		ID: "reading-1", UserID: "alice", Question: "Личный вопрос",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(nil, store, testJWTSecret)

	tests := []struct {
		name   string
		userID string
	}{
		{"guest", ""},
		{"forged owner header", "alice"},
		{"forged other user header", "bob"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/spreads/reading-1", nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401; body: %s",
					rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "Личный вопрос") {
				t.Error("private reading was disclosed in the response")
			}
		})
	}
}
