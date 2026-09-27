package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	application "github.com/galash-uptc/backend/internal/application/auth"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type registerRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
}

type Handler struct {
	service *application.Service
	db      ports.HealthChecker
}

func NewHandler(service *application.Service, db ports.HealthChecker) *Handler {
	return &Handler{service: service, db: db}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /auth/register", h.register)
	return mux
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	token, err := bearerToken(r.Header.Get("Authorization"))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var request registerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	result, err := h.service.Register(r.Context(), token, application.RegisterCommand{Name: request.Nombre, LastName: request.Apellido})
	if errors.Is(err, domain.ErrIncompleteIdentity) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, domain.ErrInvalidIdentity) {
		writeError(w, http.StatusUnauthorized, "token de Firebase inválido")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no fue posible guardar el usuario")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "base de datos no disponible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("se requiere Authorization: Bearer <token>")
	}
	return parts[1], nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
