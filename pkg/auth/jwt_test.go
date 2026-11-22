package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	secret := []byte("super-secret-key")
	token, err := Sign(secret, Claims{
		Sub:  "user-42",
		Role: "user",
		Exp:  time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	got, err := Verify(secret, token)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if got.Sub != "user-42" || got.Role != "user" {
		t.Errorf("got claims = %+v; want sub=user-42, role=user", got)
	}

	if _, err := Verify([]byte("wrong-secret"), token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("wrong secret err = %v; want ErrInvalidToken", err)
	}

	expiredToken, err := Sign(secret, Claims{
		Sub: "user-42",
		Iat: time.Now().Add(-2 * time.Hour).Unix(),
		Exp: time.Now().Add(-time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("Sign expired error: %v", err)
	}
	if _, err := Verify(secret, expiredToken); !errors.Is(err, ErrExpiredToken) {
		t.Errorf("expired err = %v; want ErrExpiredToken", err)
	}
}

func TestMiddleware(t *testing.T) {
	secret := []byte("test-secret")
	token, err := Sign(secret, Claims{Sub: "user-1", Role: "admin"})
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	h := Middleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserID(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(uid))
	}))

	t.Run("spoofed X-User-ID without jwt is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d; want 401", rec.Code)
		}
	})

	t.Run("valid bearer jwt succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || rec.Body.String() != "user-1" {
			t.Fatalf("status = %d, body = %q; want 200 user-1", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid bearer jwt returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer broken.jwt.token")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d; want 401", rec.Code)
		}
	})
}
