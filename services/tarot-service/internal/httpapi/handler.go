package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/arcana"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/deck"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/interpreter"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/matrix"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/reading"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/spread"
)

const (
	maxQuestionRunes = 500
	defaultListLimit = 20
	maxListLimit     = 100
)

type matrixResponse struct {
	BirthDate string                   `json:"birth_date"`
	Matrix    matrix.Matrix            `json:"matrix"`
	Arcana    map[string]arcana.Arcana `json:"arcana"`
}

type spreadRequest struct {
	Type     string                   `json:"type"`
	Question string                   `json:"question"`
	Cards    []spread.ManualCardInput `json:"cards,omitempty"`
}

type spreadResponse struct {
	ID             string        `json:"id,omitempty"`
	Mode           string        `json:"mode"`
	Question       string        `json:"question,omitempty"`
	Spread         spread.Spread `json:"spread"`
	Interpretation string        `json:"interpretation"`
	Source         string        `json:"source"`
	CreatedAt      *time.Time    `json:"created_at,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	interpreter interpreter.Interpreter
	readings    reading.Store
}

func NewHandler(interp interpreter.Interpreter, store reading.Store) http.Handler {
	if interp == nil {
		interp = interpreter.FallbackInterpreter{}
	}
	if store == nil {
		store = reading.NewMemoryStore()
	}
	h := &handler{
		interpreter: interp,
		readings:    store,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /arcana", handleAllArcana)
	mux.HandleFunc("GET /arcana/{number}", handleArcanaByNumber)
	mux.HandleFunc("GET /deck", handleDeck)
	mux.HandleFunc("GET /matrix", handleMatrix)
	mux.HandleFunc("POST /spread", h.handleSpread)
	mux.HandleFunc("GET /spreads", h.handleListSpreads)
	mux.HandleFunc("GET /spreads/{id}", h.handleGetSpread)
	mux.HandleFunc("DELETE /spreads/{id}", h.handleDeleteSpread)
	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleAllArcana(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, arcana.All())
}

func handleDeck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, deck.All())
}

func handleArcanaByNumber(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("number")
	num, err := strconv.Atoi(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "number must be an integer")
		return
	}

	item, err := arcana.Get(num)
	if err != nil {
		if errors.Is(err, arcana.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func handleMatrix(w http.ResponseWriter, r *http.Request) {
	rawDate := r.URL.Query().Get("birth_date")
	if rawDate == "" {
		writeError(w, http.StatusBadRequest, "birth_date query parameter is required (format: YYYY-MM-DD)")
		return
	}

	birthDate, err := time.Parse("2006-01-02", rawDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid birth_date format, expected YYYY-MM-DD")
		return
	}

	m, err := matrix.Calculate(birthDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	arcanaMap := make(map[string]arcana.Arcana)
	for _, n := range m.AllArcanaNumbers() {
		key := strconv.Itoa(n)
		if item, err := arcana.Get(n); err == nil {
			arcanaMap[key] = item
		}
	}

	writeJSON(w, http.StatusOK, matrixResponse{
		BirthDate: rawDate,
		Matrix:    m,
		Arcana:    arcanaMap,
	})
}

func (h *handler) handleSpread(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)

	var req spreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	req.Type = strings.TrimSpace(req.Type)
	req.Question = strings.TrimSpace(req.Question)
	if utf8.RuneCountInString(req.Question) > maxQuestionRunes {
		writeError(w, http.StatusBadRequest, "question is too long (max 500 characters)")
		return
	}

	var (
		sp   spread.Spread
		mode string
		err  error
	)

	if len(req.Cards) > 0 {
		mode = "manual"
		if req.Type == "" {
			req.Type = "custom"
		}
		sp, err = spread.BuildManual(req.Type, req.Cards)
	} else {
		mode = "auto"
		if req.Type == "" {
			req.Type = "three_cards"
		}
		sp, err = spread.Draw(req.Type)
	}

	if err != nil {
		if errors.Is(err, spread.ErrUnknownType) ||
			errors.Is(err, spread.ErrInvalidCardCount) ||
			errors.Is(err, spread.ErrDuplicateCard) ||
			errors.Is(err, deck.ErrCardNotFound) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to build spread")
		return
	}

	res, err := h.interpreter.InterpretSpread(r.Context(), interpreter.Input{
		Spread:   sp,
		Question: req.Question,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to interpret spread")
		return
	}

	resp := spreadResponse{
		Mode:           mode,
		Question:       req.Question,
		Spread:         sp,
		Interpretation: res.Text,
		Source:         res.Source,
	}

	if userID := strings.TrimSpace(r.Header.Get("X-User-ID")); userID != "" {
		now := time.Now().UTC()
		rec := reading.Reading{
			ID:             reading.NewID(),
			UserID:         userID,
			Mode:           mode,
			Question:       req.Question,
			Spread:         sp,
			Interpretation: res.Text,
			Source:         res.Source,
			CreatedAt:      now,
		}
		if err := h.readings.Save(r.Context(), rec); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save reading")
			return
		}
		resp.ID = rec.ID
		resp.CreatedAt = &now
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) handleListSpreads(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	limit := defaultListLimit
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > maxListLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}

	items, err := h.readings.ListByUser(r.Context(), userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list readings")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *handler) handleGetSpread(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	item, err := h.readings.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, reading.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get reading")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *handler) handleDeleteSpread(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if err := h.readings.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, reading.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, reading.ErrForbidden) {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete reading")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
