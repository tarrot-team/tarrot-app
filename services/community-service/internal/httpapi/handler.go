package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tarrot-team/tarrot-app/services/community-service/internal/profile"
)

const maxBodyBytes = 8192

type createProfileRequest struct {
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
	BirthDate string `json:"birth_date"`
}

type updateProfileRequest struct {
	Username  *string `json:"username"`
	Bio       *string `json:"bio"`
	AvatarURL *string `json:"avatar_url"`
	BirthDate *string `json:"birth_date"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	profiles *profile.Service
	log      *slog.Logger
}

func NewHandler(profiles *profile.Service, log *slog.Logger) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	h := &handler{
		profiles: profiles,
		log:      log,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /profiles", h.handleCreateProfile)
	mux.HandleFunc("GET /profiles/me", h.handleGetMyProfile)
	mux.HandleFunc("GET /profiles/{id}", h.handleGetProfile)
	mux.HandleFunc("PATCH /profiles/{id}", h.handleUpdateProfile)
	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req createProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	p, err := h.profiles.Create(r.Context(), profile.CreateInput{
		UserID:    userID,
		Username:  req.Username,
		Bio:       req.Bio,
		AvatarURL: req.AvatarURL,
		BirthDate: req.BirthDate,
	})
	if err != nil {
		h.writeProfileError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

func (h *handler) handleGetMyProfile(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	p, err := h.profiles.GetByID(r.Context(), userID)
	if err != nil {
		h.writeProfileError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *handler) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.profiles.GetByID(r.Context(), id)
	if err != nil {
		h.writeProfileError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *handler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	callerID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if callerID == "" {
		writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	targetID := r.PathValue("id")
	if callerID != targetID {
		writeError(w, http.StatusForbidden, "cannot edit another user's profile")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	p, err := h.profiles.Update(r.Context(), targetID, profile.UpdateInput{
		Username:  req.Username,
		Bio:       req.Bio,
		AvatarURL: req.AvatarURL,
		BirthDate: req.BirthDate,
	})
	if err != nil {
		h.writeProfileError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *handler) writeProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profile.ErrInvalidUserID),
		errors.Is(err, profile.ErrInvalidUsername),
		errors.Is(err, profile.ErrInvalidBio),
		errors.Is(err, profile.ErrInvalidAvatarURL),
		errors.Is(err, profile.ErrInvalidBirthDate):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, profile.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, profile.ErrAlreadyExists),
		errors.Is(err, profile.ErrUsernameTaken):
		writeError(w, http.StatusConflict, err.Error())
	default:
		h.log.Error("unexpected profile error", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
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
