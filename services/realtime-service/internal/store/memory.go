package store

import (
	"context"
	"sync"

	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/hub"
)

const maxMemoryMessagesPerRoom = 200

type MemoryStore struct {
	mu    sync.RWMutex
	rooms map[string][]hub.Event
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rooms: make(map[string][]hub.Event),
	}
}

func (s *MemoryStore) SaveMessage(_ context.Context, ev hub.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgs := append(s.rooms[ev.Room], ev)
	if len(msgs) > maxMemoryMessagesPerRoom {
		msgs = msgs[len(msgs)-maxMemoryMessagesPerRoom:]
	}
	s.rooms[ev.Room] = msgs
	return nil
}

func (s *MemoryStore) ListMessages(_ context.Context, room string, limit int) ([]hub.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := s.rooms[room]
	if len(msgs) == 0 {
		return []hub.Event{}, nil
	}
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}

	res := make([]hub.Event, len(msgs))
	copy(res, msgs)
	return res, nil
}
