package reading

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/spread"
)

type Reading struct {
	ID             string        `json:"id"`
	UserID         string        `json:"user_id"`
	Mode           string        `json:"mode"`
	Question       string        `json:"question,omitempty"`
	Spread         spread.Spread `json:"spread"`
	Interpretation string        `json:"interpretation"`
	Source         string        `json:"source"`
	CreatedAt      time.Time     `json:"created_at"`
}

type Store interface {
	Save(ctx context.Context, r Reading) error
	GetByID(ctx context.Context, id string) (Reading, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]Reading, error)
	Delete(ctx context.Context, id, userID string) error
}

var (
	ErrNotFound  = errors.New("reading not found")
	ErrForbidden = errors.New("reading belongs to another user")
)

func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
