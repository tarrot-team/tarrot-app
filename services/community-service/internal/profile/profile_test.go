package profile

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestService() *Service {
	svc := NewService(NewMemoryStore())
	svc.now = func() time.Time {
		return time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	}
	return svc
}

func TestServiceCreateAndGet(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	created, err := svc.Create(ctx, CreateInput{
		UserID:    "  user-1  ",
		Username:  "  anya_go  ",
		Bio:       "  пишу на go  ",
		AvatarURL: "https://example.com/avatar.png",
		BirthDate: "2007-09-05",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created.UserID != "user-1" || created.Username != "anya_go" || created.Bio != "пишу на go" {
		t.Fatalf("fields were not trimmed properly: %+v", created)
	}
	if created.Level != 1 || created.Reputation != 0 {
		t.Fatalf("unexpected defaults: level=%d rep=%d", created.Level, created.Reputation)
	}

	got, err := svc.GetByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if got.Username != "anya_go" || got.BirthDate != "2007-09-05" {
		t.Fatalf("unexpected fetched profile: %+v", got)
	}

	if _, err := svc.GetByID(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceCreateValidation(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	tests := []struct {
		name    string
		input   CreateInput
		wantErr error
	}{
		{
			name:    "empty user_id",
			input:   CreateInput{UserID: " ", Username: "valid_user"},
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "short username",
			input:   CreateInput{UserID: "u1", Username: "ab"},
			wantErr: ErrInvalidUsername,
		},
		{
			name:    "long username",
			input:   CreateInput{UserID: "u1", Username: strings.Repeat("a", 33)},
			wantErr: ErrInvalidUsername,
		},
		{
			name:    "invalid username characters",
			input:   CreateInput{UserID: "u1", Username: "user@name"},
			wantErr: ErrInvalidUsername,
		},
		{
			name:    "bio too long",
			input:   CreateInput{UserID: "u1", Username: "valid_user", Bio: strings.Repeat("я", 501)},
			wantErr: ErrInvalidBio,
		},
		{
			name:    "insecure http avatar",
			input:   CreateInput{UserID: "u1", Username: "valid_user", AvatarURL: "http://example.com/a.png"},
			wantErr: ErrInvalidAvatarURL,
		},
		{
			name:    "invalid avatar scheme",
			input:   CreateInput{UserID: "u1", Username: "valid_user", AvatarURL: "ftp://example.com/a.png"},
			wantErr: ErrInvalidAvatarURL,
		},
		{
			name:    "invalid birth date format",
			input:   CreateInput{UserID: "u1", Username: "valid_user", BirthDate: "05.09.2007"},
			wantErr: ErrInvalidBirthDate,
		},
		{
			name:    "future birth date",
			input:   CreateInput{UserID: "u1", Username: "valid_user", BirthDate: "2099-01-01"},
			wantErr: ErrInvalidBirthDate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestServiceCreateConflicts(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	if _, err := svc.Create(ctx, CreateInput{UserID: "u1", Username: "ArcanaMaster"}); err != nil {
		t.Fatalf("first create: %v", err)
	}

	if _, err := svc.Create(ctx, CreateInput{UserID: "u1", Username: "another_name"}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	if _, err := svc.Create(ctx, CreateInput{UserID: "u2", Username: "arcanamaster"}); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestServiceUpdate(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	_, err := svc.Create(ctx, CreateInput{
		UserID:   "u1",
		Username: "first_user",
		Bio:      "старое био",
	})
	if err != nil {
		t.Fatalf("create u1: %v", err)
	}
	_, err = svc.Create(ctx, CreateInput{
		UserID:   "u2",
		Username: "second_user",
	})
	if err != nil {
		t.Fatalf("create u2: %v", err)
	}

	newBio := "новое био"
	newAvatar := "/media/avatars/u1.webp"
	updated, err := svc.Update(ctx, "u1", UpdateInput{
		Bio:       &newBio,
		AvatarURL: &newAvatar,
	})
	if err != nil {
		t.Fatalf("update u1: %v", err)
	}
	if updated.Username != "first_user" || updated.Bio != newBio || updated.AvatarURL != newAvatar {
		t.Fatalf("unexpected updated state: %+v", updated)
	}

	takenName := "SECOND_USER"
	if _, err := svc.Update(ctx, "u1", UpdateInput{Username: &takenName}); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}

	if _, err := svc.Update(ctx, "unknown", UpdateInput{Bio: &newBio}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
