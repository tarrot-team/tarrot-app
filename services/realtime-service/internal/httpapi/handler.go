package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tarrot-team/tarrot-app/pkg/auth"
	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/hub"
	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/store"
)

const (
	maxBodyBytes    = 4096
	maxMessageRunes = 1000
	defaultLimit    = 20
	maxLimit        = 50
)

type sendMessageRequest struct {
	Text string `json:"text"`
}

type notifyRequest struct {
	ToUserID   string `json:"to_user_id"`
	FromUserID string `json:"from_user_id"`
	Text       string `json:"text"`
}

type roomMessagesResponse struct {
	Room     string      `json:"room"`
	Online   int         `json:"online"`
	Messages []hub.Event `json:"messages"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	hub   *hub.Hub
	store store.MessageStore
}

func NewHandler(h *hub.Hub, st store.MessageStore, jwtSecret []byte) http.Handler {
	if h == nil {
		h = hub.New()
	}
	if st == nil {
		st = store.NewMemoryStore()
	}
	api := &handler{hub: h, store: st}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /rooms/{room}/stream", api.handleRoomStream)
	mux.HandleFunc("GET /rooms/{room}/messages", api.handleRoomMessages)
	mux.HandleFunc("POST /rooms/{room}/messages", api.handlePostMessage)
	mux.HandleFunc("GET /notifications/stream", api.handleNotificationStream)
	mux.HandleFunc("POST /notifications", api.handleSendNotification)
	return auth.Middleware(jwtSecret)(mux)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	room := strings.TrimSpace(r.PathValue("room"))
	if room == "" {
		writeError(w, http.StatusBadRequest, "room is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	if utf8.RuneCountInString(text) > maxMessageRunes {
		writeError(w, http.StatusBadRequest, "text is too long (max 1000 characters)")
		return
	}

	ev := h.hub.PublishMessage(room, userID, text)
	if err := h.store.SaveMessage(r.Context(), ev); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save message")
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}

func (h *handler) handleRoomMessages(w http.ResponseWriter, r *http.Request) {
	room := strings.TrimSpace(r.PathValue("room"))
	if room == "" {
		writeError(w, http.StatusBadRequest, "room is required")
		return
	}

	limit := defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 50")
			return
		}
		limit = parsed
	}

	msgs, err := h.store.ListMessages(r.Context(), room, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}

	writeJSON(w, http.StatusOK, roomMessagesResponse{
		Room:     room,
		Online:   h.hub.RoomSubscribers(room),
		Messages: msgs,
	})
}

func (h *handler) handleRoomStream(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserID(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	room := strings.TrimSpace(r.PathValue("room"))
	if room == "" {
		writeError(w, http.StatusBadRequest, "room is required")
		return
	}

	ch, cancel := h.hub.SubscribeRoom(room)
	defer cancel()

	streamSSE(w, r, ch)
}

func (h *handler) handleNotificationStream(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ch, cancel := h.hub.SubscribeUser(userID)
	defer cancel()

	streamSSE(w, r, ch)
}

func (h *handler) handleSendNotification(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	toUserID := strings.TrimSpace(req.ToUserID)
	text := strings.TrimSpace(req.Text)
	if toUserID == "" || text == "" {
		writeError(w, http.StatusBadRequest, "to_user_id and text are required")
		return
	}
	if utf8.RuneCountInString(text) > maxMessageRunes {
		writeError(w, http.StatusBadRequest, "text is too long (max 1000 characters)")
		return
	}

	ev := h.hub.NotifyUser(toUserID, strings.TrimSpace(req.FromUserID), text)
	writeJSON(w, http.StatusCreated, ev)
}

func streamSSE(w http.ResponseWriter, r *http.Request, ch <-chan hub.Event) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			_, _ = fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
