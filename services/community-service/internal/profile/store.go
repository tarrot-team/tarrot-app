package profile

import (
	"context"
	"strings"
	"sync"
)

type MemoryStore struct {
	mu         sync.RWMutex
	byID       map[string]Profile
	byUsername map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byID:       make(map[string]Profile),
		byUsername: make(map[string]string),
	}
}

func (s *MemoryStore) Create(_ context.Context, p Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byID[p.UserID]; exists {
		return ErrAlreadyExists
	}

	key := strings.ToLower(p.Username)
	if _, taken := s.byUsername[key]; taken {
		return ErrUsernameTaken
	}

	s.byID[p.UserID] = p
	s.byUsername[key] = p.UserID
	return nil
}

func (s *MemoryStore) GetByID(_ context.Context, userID string) (Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.byID[userID]
	if !ok {
		return Profile{}, ErrNotFound
	}
	return p, nil
}

func (s *MemoryStore) Update(_ context.Context, p Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, exists := s.byID[p.UserID]
	if !exists {
		return ErrNotFound
	}

	oldKey := strings.ToLower(old.Username)
	newKey := strings.ToLower(p.Username)
	if ownerID, taken := s.byUsername[newKey]; taken && ownerID != p.UserID {
		return ErrUsernameTaken
	}

	if oldKey != newKey {
		delete(s.byUsername, oldKey)
	}
	s.byUsername[newKey] = p.UserID
	s.byID[p.UserID] = p
	return nil
}
