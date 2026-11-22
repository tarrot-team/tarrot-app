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
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/spread"
)

const maxQuestionRunes = 500

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
	Mode           string        `json:"mode"`
	Question       string        `json:"question,omitempty"`
	Spread         spread.Spread `json:"spread"`
	Interpretation string        `json:"interpretation"`
	Source         string        `json:"source"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	interpreter interpreter.Interpreter
}

func NewHandler(interp interpreter.Interpreter) http.Handler {
	if interp == nil {
		interp = interpreter.FallbackInterpreter{}
	}
	h := &handler{interpreter: interp}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /arcana", handleAllArcana)
	mux.HandleFunc("GET /arcana/{number}", handleArcanaByNumber)
	mux.HandleFunc("GET /deck", handleDeck)
	mux.HandleFunc("GET /matrix", handleMatrix)
	mux.HandleFunc("POST /spread", h.handleSpread)
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

	writeJSON(w, http.StatusOK, spreadResponse{
		Mode:           mode,
		Question:       req.Question,
		Spread:         sp,
		Interpretation: res.Text,
		Source:         res.Source,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
