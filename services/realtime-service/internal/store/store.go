package store

import (
	"context"

	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/hub"
)

type MessageStore interface {
	SaveMessage(ctx context.Context, ev hub.Event) error
	ListMessages(ctx context.Context, room string, limit int) ([]hub.Event, error)
}
