package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tarrot-team/tarrot-app/pkg/auth"
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

func signNotificationToken(t *testing.T, sub, role string) string {
	t.Helper()
	token, err := auth.Sign(testJWTSecret, auth.Claims{
		Sub:  sub,
		Role: role,
	})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func TestNotificationForbiddenForRegularUser(t *testing.T) {
	h := hub.New()
	ch, cancel := h.SubscribeUser("recipient")
	defer cancel()

	token := signNotificationToken(t, "alice", "user")
	req := httptest.NewRequest(
		http.MethodPost,
		"/notifications",
		strings.NewReader(`{"to_user_id":"recipient","text":"spam"}`),
	)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	NewHandler(h, store.NewMemoryStore(), testJWTSecret).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body: %s", rec.Code, rec.Body.String())
	}
	select {
	case ev := <-ch:
		t.Fatalf("unexpected notification delivered: %+v", ev)
	default:
	}
}

func TestNotificationAdminCannotSpoofFromUserID(t *testing.T) {
	h := hub.New()
	ch, cancel := h.SubscribeUser("recipient")
	defer cancel()

	token := signNotificationToken(t, "admin-1", "admin")
	req := httptest.NewRequest(
		http.MethodPost,
		"/notifications",
		strings.NewReader(`{"to_user_id":"recipient","from_user_id":"alice","text":"spoofed"}`),
	)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	NewHandler(h, store.NewMemoryStore(), testJWTSecret).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body: %s", rec.Code, rec.Body.String())
	}
	select {
	case ev := <-ch:
		t.Fatalf("unexpected spoofed notification delivered: %+v", ev)
	default:
	}
}

func TestNotificationServiceAllowedAndRateLimited(t *testing.T) {
	h := hub.New()
	ch, cancel := h.SubscribeUser("recipient")
	defer cancel()

	api := NewHandler(h, store.NewMemoryStore(), testJWTSecret)
	token := signNotificationToken(t, "community-service", "service")
	body := `{"to_user_id":"recipient","from_user_id":"user-2","text":"new reply"}`

	for i := 0; i < notifyRateLimit; i++ {
		req := httptest.NewRequest(http.MethodPost, "/notifications", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		api.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want 201; body: %s", i+1, rec.Code, rec.Body.String())
		}
		select {
		case ev := <-ch:
			if ev.FromUserID != "user-2" {
				t.Fatalf("from_user_id = %q, want user-2", ev.FromUserID)
			}
		default:
			t.Fatalf("request %d: expected notification event", i+1)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/notifications", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("over-limit status = %d, want 429; body: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header on 429 response")
	}
}
