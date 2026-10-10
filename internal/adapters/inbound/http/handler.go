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
	Nombre         string `json:"nombre"`
	Apellido       string `json:"apellido"`
	TipoDNI        string `json:"tipo_dni"`
	DNI            string `json:"dni"`
	Email          string `json:"email"`
	Rol            string `json:"rol"`
}

type sessionRequest struct { Token string `json:"token"` }
type recoveryRequest struct { Email string `json:"email"` }
type resetRequest struct { Token string `json:"token"`; Password string `json:"password"` }

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
	mux.HandleFunc("POST /auth/sessions", h.createSession)
	mux.HandleFunc("DELETE /auth/sessions/current", h.deleteSession)
	mux.HandleFunc("GET /users/me", h.currentUser)
	return mux
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	var request sessionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil { writeError(w, http.StatusBadRequest, "cuerpo JSON inválido"); return }
	if request.Token == "" { request.Token = r.Header.Get("Authorization") }
	if strings.HasPrefix(strings.ToLower(request.Token), "bearer ") { request.Token, _ = bearerToken(request.Token) }
	result, err := h.service.Login(r.Context(), request.Token)
	if errors.Is(err, domain.ErrInvalidCredentials) || errors.Is(err, domain.ErrInactiveUser) { writeError(w, http.StatusUnauthorized, "credenciales inválidas"); return }
	if err != nil { writeError(w, http.StatusInternalServerError, "no fue posible iniciar sesión"); return }
	http.SetCookie(w, &http.Cookie{Name: "galash_session", Value: result.Token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	writeJSON(w, http.StatusCreated, map[string]any{"user": result.User})
}

func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r); if err != nil { writeError(w, http.StatusUnauthorized, "sesión requerida"); return }
	if err := h.service.Logout(r.Context(), token); err != nil { writeError(w, http.StatusUnauthorized, "sesión inválida"); return }
	http.SetCookie(w, &http.Cookie{Name: "galash_session", Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r); if err != nil { writeError(w, http.StatusUnauthorized, "sesión requerida"); return }
	user, err := h.service.CurrentUser(r.Context(), token)
	if err != nil { writeError(w, http.StatusUnauthorized, "sesión inválida o expirada"); return }
	writeJSON(w, http.StatusOK, user)
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
	result, err := h.service.Register(r.Context(), token, application.RegisterCommand{
		Name: request.Nombre, LastName: request.Apellido,
		DocumentType: request.TipoDNI, DocumentNumber: request.DNI,
		Email: request.Email, Role: request.Rol,
	})
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

func sessionToken(r *http.Request) (string, error) {
	if cookie, err := r.Cookie("galash_session"); err == nil && cookie.Value != "" { return cookie.Value, nil }
	return bearerToken(r.Header.Get("Authorization"))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
