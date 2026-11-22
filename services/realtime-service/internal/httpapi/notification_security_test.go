package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/hub"
	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/store"
)

func TestNotificationRejectsUnverifiedSender(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		body   string
	}{
		{
			name: "no identity",
			body: `{"to_user_id":"recipient","from_user_id":"alice","text":"test notification"}`,
		},
		{
			name:   "forged header and sender",
			userID: "alice",
			body:   `{"to_user_id":"recipient","from_user_id":"admin","text":"test notification"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := hub.New()
			ch, cancel := h.SubscribeUser("recipient")
			defer cancel()

			req := httptest.NewRequest(http.MethodPost, "/notifications", strings.NewReader(tt.body))
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			rec := httptest.NewRecorder()
			NewHandler(h, store.NewMemoryStore(), testJWTSecret).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401; body: %s", rec.Code, rec.Body.String())
			}
			select {
			case ev := <-ch:
				t.Errorf("notification delivered without verified sender: %+v", ev)
			default:
			}
		})
	}
}
