package httpadapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	application "github.com/galash-uptc/backend/internal/application/auth"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type registerRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	TipoDNI  string `json:"tipo_dni"`
	DNI      string `json:"dni"`
	Email    string `json:"email"`
	Rol      string `json:"rol"`
}

type sessionRequest struct {
	Token string `json:"token"`
}
type recoveryRequest struct {
	Email string `json:"email"`
}
type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
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
	mux.HandleFunc("POST /auth/sessions", h.createSession)
	mux.HandleFunc("DELETE /auth/sessions/current", h.deleteSession)
	mux.HandleFunc("GET /users/me", h.currentUser)
	mux.HandleFunc("GET /members", h.publicMembers)
	mux.HandleFunc("GET /find/user", h.adminMembers)
	mux.HandleFunc("GET /admin/members", h.adminMembers)
	mux.HandleFunc("PUT /users/roles", h.updateRole)
	mux.HandleFunc("PUT /users/{userId}/roles", h.updateRole)
	return mux
}

type roleRequest struct {
	Role string `json:"role"`
	Rol  string `json:"rol"`
}

func (h *Handler) publicMembers(w http.ResponseWriter, r *http.Request) {
	query := searchQuery(r)
	users, err := h.service.SearchUsers(r.Context(), query, requestedLimit(r))
	if errors.Is(err, domain.ErrIncompleteIdentity) {
		writeError(w, http.StatusBadRequest, "se requiere una búsqueda por nombre o correo")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no fue posible consultar los integrantes")
		return
	}
	profiles := make([]domain.PublicProfile, 0, len(users))
	for _, user := range users {
		profiles = append(profiles, domain.PublicProfile{ID: user.ID, Name: user.Name, LastName: user.LastName, Role: user.Role})
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": profiles})
}

func (h *Handler) adminMembers(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sesión requerida")
		return
	}
	users, err := h.service.SearchAdminUsers(r.Context(), token, searchQuery(r), requestedLimit(r))
	if errors.Is(err, domain.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sesión inválida o expirada")
		return
	}
	if errors.Is(err, domain.ErrForbidden) {
		writeError(w, http.StatusForbidden, "se requiere rol admin")
		return
	}
	if errors.Is(err, domain.ErrIncompleteIdentity) {
		writeError(w, http.StatusBadRequest, "se requiere una búsqueda por nombre o correo")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no fue posible consultar los usuarios")
		return
	}
	profiles := make([]domain.AdminProfile, 0, len(users))
	for _, user := range users {
		profiles = append(profiles, domain.AdminProfile{ID: user.ID, FirebaseUID: user.FirebaseUID, Name: user.Name, LastName: user.LastName, Email: user.Email, DocumentType: user.DocumentType, DocumentNumber: user.DocumentNumber, Role: user.Role, Status: user.Status})
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": profiles})
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sesión requerida")
		return
	}
	targetUID := strings.TrimSpace(r.Header.Get("X-User-UID"))
	pathUID := strings.TrimSpace(r.PathValue("userId"))
	if targetUID != "" && pathUID != "" && targetUID != pathUID {
		writeError(w, http.StatusBadRequest, "el UID del header no coincide con el recurso")
		return
	}
	if targetUID == "" {
		targetUID = pathUID
	}
	var request roleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	role := request.Role
	if role == "" {
		role = request.Rol
	}
	err = h.service.UpdateUserRole(r.Context(), token, targetUID, role)
	switch {
	case errors.Is(err, domain.ErrInvalidSession):
		writeError(w, http.StatusUnauthorized, "sesión inválida o expirada")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "se requiere rol admin")
	case errors.Is(err, domain.ErrSelfRoleChange):
		writeError(w, http.StatusConflict, "no puede cambiar su propio rol")
	case errors.Is(err, domain.ErrInvalidRole), errors.Is(err, domain.ErrIncompleteIdentity):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		writeError(w, http.StatusNotFound, "usuario no encontrado")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func requestedLimit(r *http.Request) int {
	limit := 5
	if value := r.URL.Query().Get("limit"); value != "" {
		if _, err := fmt.Sscanf(value, "%d", &limit); err != nil {
			limit = 5
		}
	}
	return limit
}

func searchQuery(r *http.Request) string {
	for _, key := range []string{"q", "nombre", "name", "correo", "email"} {
		if value := r.URL.Query().Get(key); value != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	var request sessionRequest
	// The Authorization header is authoritative. Do not decode the body first:
	// clients using the header are allowed to send no body at all.
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		var err error
		request.Token, err = bearerToken(authorization)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			if err == io.EOF {
				writeError(w, http.StatusUnauthorized, "se requiere un token de sesión")
				return
			}
			writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
			return
		}
	}
	result, err := h.service.Login(r.Context(), request.Token)
	if errors.Is(err, domain.ErrInvalidCredentials) || errors.Is(err, domain.ErrInactiveUser) || errors.Is(err, domain.ErrUserNotRegistered) {
		writeError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no fue posible iniciar sesión")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "galash_session", Value: result.Token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	writeJSON(w, http.StatusCreated, map[string]any{"user": result.User})
}

func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sesión requerida")
		return
	}
	if err := h.service.Logout(r.Context(), token); err != nil {
		writeError(w, http.StatusUnauthorized, "sesión inválida")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "galash_session", Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sesión requerida")
		return
	}
	user, err := h.service.CurrentUser(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sesión inválida o expirada")
		return
	}
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
	if cookie, err := r.Cookie("galash_session"); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
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
