package reading

import (
	"context"
	"sort"
	"sync"
)

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Reading
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items: make(map[string]Reading),
	}
}

func (s *MemoryStore) Save(_ context.Context, r Reading) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[r.ID] = r
	return nil
}

func (s *MemoryStore) GetByID(_ context.Context, id string) (Reading, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.items[id]
	if !ok {
		return Reading{}, ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) ListByUser(_ context.Context, userID string, limit int) ([]Reading, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]Reading, 0)
	for _, r := range s.items {
		if r.UserID == userID {
			res = append(res, r)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.After(res[j].CreatedAt)
	})

	if limit > 0 && len(res) > limit {
		res = res[:limit]
	}

	return res, nil
}

func (s *MemoryStore) Delete(_ context.Context, id, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.items[id]
	if !ok {
		return ErrNotFound
	}
	if r.UserID != userID {
		return ErrForbidden
	}

	delete(s.items, id)
	return nil
}
