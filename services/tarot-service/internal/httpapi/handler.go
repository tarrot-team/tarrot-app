package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/arcana"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/matrix"
)

type matrixResponse struct {
	BirthDate string                   `json:"birth_date"`
	Matrix    matrix.Matrix            `json:"matrix"`
	Arcana    map[string]arcana.Arcana `json:"arcana"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /arcana", handleAllArcana)
	mux.HandleFunc("GET /arcana/{number}", handleArcanaByNumber)
	mux.HandleFunc("GET /matrix", handleMatrix)
	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleAllArcana(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, arcana.All())
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

	nums := []int{m.A, m.B, m.C, m.D, m.E, m.F, m.G, m.H, m.I, m.J, m.K, m.L, m.M}
	arcanaMap := make(map[string]arcana.Arcana)
	for _, n := range nums {
		key := strconv.Itoa(n)
		if _, exists := arcanaMap[key]; exists {
			continue
		}
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

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
