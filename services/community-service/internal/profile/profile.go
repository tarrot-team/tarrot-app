package profile

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MinUsernameRunes  = 3
	MaxUsernameRunes  = 32
	MaxBioRunes       = 500
	MaxAvatarURLRunes = 500
)

var (
	ErrNotFound         = errors.New("profile not found")
	ErrAlreadyExists    = errors.New("profile already exists")
	ErrUsernameTaken    = errors.New("username already taken")
	ErrInvalidUserID    = errors.New("user_id is required")
	ErrInvalidUsername  = errors.New("username must be 3-32 characters and contain only letters, digits, '_' or '-'")
	ErrInvalidBio       = errors.New("bio is too long (max 500 characters)")
	ErrInvalidAvatarURL = errors.New("invalid avatar_url")
	ErrInvalidBirthDate = errors.New("invalid birth_date format, expected YYYY-MM-DD in the past")
)

type Profile struct {
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	Bio        string    `json:"bio"`
	AvatarURL  string    `json:"avatar_url"`
	BirthDate  string    `json:"birth_date,omitempty"`
	Level      int       `json:"level"`
	Reputation int       `json:"reputation"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateInput struct {
	UserID    string
	Username  string
	Bio       string
	AvatarURL string
	BirthDate string
}

type UpdateInput struct {
	Username  *string
	Bio       *string
	AvatarURL *string
	BirthDate *string
}

type Store interface {
	Create(ctx context.Context, p Profile) error
	GetByID(ctx context.Context, userID string) (Profile, error)
	Update(ctx context.Context, p Profile) error
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Profile, error) {
	userID := strings.TrimSpace(in.UserID)
	if userID == "" {
		return Profile{}, ErrInvalidUserID
	}

	username, err := validateUsername(in.Username)
	if err != nil {
		return Profile{}, err
	}

	bio, err := validateBio(in.Bio)
	if err != nil {
		return Profile{}, err
	}

	avatarURL, err := validateAvatarURL(in.AvatarURL)
	if err != nil {
		return Profile{}, err
	}

	birthDate, err := validateBirthDate(in.BirthDate, s.now())
	if err != nil {
		return Profile{}, err
	}

	now := s.now()
	p := Profile{
		UserID:     userID,
		Username:   username,
		Bio:        bio,
		AvatarURL:  avatarURL,
		BirthDate:  birthDate,
		Level:      1,
		Reputation: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, p); err != nil {
		return Profile{}, err
	}

	return p, nil
}

func (s *Service) GetByID(ctx context.Context, userID string) (Profile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Profile{}, ErrInvalidUserID
	}
	return s.store.GetByID(ctx, userID)
}

func (s *Service) Update(ctx context.Context, userID string, in UpdateInput) (Profile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Profile{}, ErrInvalidUserID
	}

	p, err := s.store.GetByID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}

	if in.Username != nil {
		username, err := validateUsername(*in.Username)
		if err != nil {
			return Profile{}, err
		}
		p.Username = username
	}

	if in.Bio != nil {
		bio, err := validateBio(*in.Bio)
		if err != nil {
			return Profile{}, err
		}
		p.Bio = bio
	}

	if in.AvatarURL != nil {
		avatarURL, err := validateAvatarURL(*in.AvatarURL)
		if err != nil {
			return Profile{}, err
		}
		p.AvatarURL = avatarURL
	}

	if in.BirthDate != nil {
		birthDate, err := validateBirthDate(*in.BirthDate, s.now())
		if err != nil {
			return Profile{}, err
		}
		p.BirthDate = birthDate
	}

	p.UpdatedAt = s.now()
	if err := s.store.Update(ctx, p); err != nil {
		return Profile{}, err
	}

	return p, nil
}

func validateUsername(raw string) (string, error) {
	username := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(username)
	if n < MinUsernameRunes || n > MaxUsernameRunes {
		return "", ErrInvalidUsername
	}
	for _, r := range username {
		isASCII := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !isASCII && r != '_' && r != '-' {
			return "", ErrInvalidUsername
		}
	}
	return username, nil
}

func validateBio(raw string) (string, error) {
	bio := strings.TrimSpace(raw)
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return "", ErrInvalidBio
	}
	return bio, nil
}

func validateAvatarURL(raw string) (string, error) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", nil
	}
	if utf8.RuneCountInString(u) > MaxAvatarURLRunes {
		return "", ErrInvalidAvatarURL
	}
	if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "/") {
		return "", ErrInvalidAvatarURL
	}
	return u, nil
}

func validateBirthDate(raw string, now time.Time) (string, error) {
	d := strings.TrimSpace(raw)
	if d == "" {
		return "", nil
	}
	parsed, err := time.Parse("2006-01-02", d)
	if err != nil || parsed.After(now) {
		return "", ErrInvalidBirthDate
	}
	return d, nil
}
