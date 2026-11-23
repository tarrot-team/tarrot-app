package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tarrot-team/tarrot-app/pkg/auth"
	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/hub"
	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/store"
)

var testJWTSecret = []byte("test-jwt-secret")

func bearerFor(t *testing.T, userID string) string {
	t.Helper()
	tok, err := auth.Sign(testJWTSecret, auth.Claims{Sub: userID, Role: "user"})
	if err != nil {
		t.Fatalf("auth.Sign: %v", err)
	}
	return "Bearer " + tok
}

func TestHandleHealth(t *testing.T) {
	api := NewHandler(hub.New(), store.NewMemoryStore(), testJWTSecret)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rec.Code)
	}
}

func TestPostAndGetRoomMessages(t *testing.T) {
	api := NewHandler(hub.New(), store.NewMemoryStore(), testJWTSecret)

	spoofed := httptest.NewRequest(http.MethodPost, "/rooms/tarot/messages", bytes.NewBufferString(`{"text":"hi"}`))
	spoofed.Header.Set("X-User-ID", "user-1")
	spoofedRec := httptest.NewRecorder()
	api.ServeHTTP(spoofedRec, spoofed)
	if spoofedRec.Code != http.StatusUnauthorized {
		t.Fatalf("spoofed status = %d; want 401", spoofedRec.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/rooms/tarot/messages", bytes.NewBufferString(`{"text":"Привет, чат!"}`))
	req.Header.Set("Authorization", bearerFor(t, "user-1"))
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("post status = %d; want 201", rec.Code)
	}

	getRec := httptest.NewRecorder()
	api.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/rooms/tarot/messages", nil))
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d; want 200", getRec.Code)
	}

	var resp roomMessagesResponse
	if err := json.NewDecoder(getRec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Messages) != 1 || resp.Messages[0].Text != "Привет, чат!" {
		t.Fatalf("unexpected messages: %+v", resp)
	}
}

func TestRoomSSEStream(t *testing.T) {
	h := hub.New()
	api := NewHandler(h, store.NewMemoryStore(), testJWTSecret)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/rooms/tarot/stream", nil).WithContext(ctx)
	req.Header.Set("Authorization", bearerFor(t, "user-1"))
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		api.ServeHTTP(rec, req)
		close(done)
	}()

	deadline := time.Now().Add(time.Second)
	for h.RoomSubscribers("tarot") == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if h.RoomSubscribers("tarot") != 1 {
		cancel()
		t.Fatal("subscriber did not register in time")
	}

	h.PublishMessage("tarot", "user-2", "Живое сообщение")
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	if !strings.Contains(rec.Body.String(), "Живое сообщение") {
		t.Fatalf("expected SSE body to contain message, got: %s", rec.Body.String())
	}
}

func TestSendNotification(t *testing.T) {
	h := hub.New()
	api := NewHandler(h, store.NewMemoryStore(), testJWTSecret)

	ch, cancel := h.SubscribeUser("user-9")
	defer cancel()

	body := bytes.NewBufferString(`{"to_user_id":"user-9","from_user_id":"user-2","text":"Лайкнул твой пост"}`)
	req := httptest.NewRequest(http.MethodPost, "/notifications", body)
	req.Header.Set("Authorization", "Bearer "+signNotificationToken(t, "community-service", "service"))

	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; want 201", rec.Code)
	}

	select {
	case ev := <-ch:
		if ev.Text != "Лайкнул твой пост" {
			t.Errorf("got text = %q", ev.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notification")
	}
}
