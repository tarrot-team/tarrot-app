package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tarrot-team/tarrot-app/services/community-service/internal/profile"
)

func TestForgedUserIDCannotEditProfile(t *testing.T) {
	store := profile.NewMemoryStore()
	svc := profile.NewService(store)
	_, err := svc.Create(context.Background(), profile.CreateInput{
		UserID: "alice", Username: "alice",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/profiles/alice",
		strings.NewReader(`{"bio":"changed"}`))
	req.Header.Set("X-User-ID", "alice") // Поддельный заголовок, без сессии.
	rec := httptest.NewRecorder()

	NewHandler(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401; body: %s", rec.Code, rec.Body.String())
	}
	got, err := store.GetByID(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if got.Bio != "" {
		t.Fatalf("profile changed without authentication: bio = %q", got.Bio)
	}
}
