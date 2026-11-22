package hub

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	maxRoomHistory   = 50
	subscriberBuffer = 32
)

type Event struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Room       string    `json:"room,omitempty"`
	FromUserID string    `json:"from_user_id,omitempty"`
	ToUserID   string    `json:"to_user_id,omitempty"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
}

type Hub struct {
	mu      sync.RWMutex
	rooms   map[string]map[chan Event]struct{}
	users   map[string]map[chan Event]struct{}
	history map[string][]Event
}

func New() *Hub {
	return &Hub{
		rooms:   make(map[string]map[chan Event]struct{}),
		users:   make(map[string]map[chan Event]struct{}),
		history: make(map[string][]Event),
	}
}

func (h *Hub) SubscribeRoom(room string) (<-chan Event, func()) {
	ch := make(chan Event, subscriberBuffer)

	h.mu.Lock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[chan Event]struct{})
	}
	h.rooms[room][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if subs, ok := h.rooms[room]; ok {
			if _, exists := subs[ch]; exists {
				delete(subs, ch)
				close(ch)
			}
			if len(subs) == 0 {
				delete(h.rooms, room)
			}
		}
		h.mu.Unlock()
	}

	return ch, cancel
}

func (h *Hub) SubscribeUser(userID string) (<-chan Event, func()) {
	ch := make(chan Event, subscriberBuffer)

	h.mu.Lock()
	if h.users[userID] == nil {
		h.users[userID] = make(map[chan Event]struct{})
	}
	h.users[userID][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if subs, ok := h.users[userID]; ok {
			if _, exists := subs[ch]; exists {
				delete(subs, ch)
				close(ch)
			}
			if len(subs) == 0 {
				delete(h.users, userID)
			}
		}
		h.mu.Unlock()
	}

	return ch, cancel
}

func (h *Hub) PublishMessage(room, fromUserID, text string) Event {
	ev := Event{
		ID:         newID(),
		Type:       "message",
		Room:       room,
		FromUserID: fromUserID,
		Text:       text,
		CreatedAt:  time.Now().UTC(),
	}

	h.mu.Lock()
	hist := append(h.history[room], ev)
	if len(hist) > maxRoomHistory {
		hist = hist[len(hist)-maxRoomHistory:]
	}
	h.history[room] = hist

	for ch := range h.rooms[room] {
		select {
		case ch <- ev:
		default:
		}
	}
	h.mu.Unlock()

	return ev
}

func (h *Hub) NotifyUser(toUserID, fromUserID, text string) Event {
	ev := Event{
		ID:         newID(),
		Type:       "notification",
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Text:       text,
		CreatedAt:  time.Now().UTC(),
	}

	h.mu.RLock()
	for ch := range h.users[toUserID] {
		select {
		case ch <- ev:
		default:
		}
	}
	h.mu.RUnlock()

	return ev
}

func (h *Hub) RecentMessages(room string, limit int) []Event {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hist := h.history[room]
	if len(hist) == 0 {
		return []Event{}
	}
	if limit > 0 && len(hist) > limit {
		hist = hist[len(hist)-limit:]
	}

	res := make([]Event, len(hist))
	copy(res, hist)
	return res
}

func (h *Hub) RoomSubscribers(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room])
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
