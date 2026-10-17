package httpadapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	application "github.com/galash-uptc/backend/internal/application/auth"
	seedbedapp "github.com/galash-uptc/backend/internal/application/seedbed"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type registerRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	TipoDNI  string `json:"tipo_dni"`
	DNI      string `json:"dni"`
	Email    string `json:"email"`
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
	service  *application.Service
	seedbeds *seedbedapp.Service
	db       ports.HealthChecker
}

func NewHandler(service *application.Service, db ports.HealthChecker, seedbeds ...*seedbedapp.Service) *Handler {
	var sb *seedbedapp.Service
	if len(seedbeds) > 0 {
		sb = seedbeds[0]
	}
	return &Handler{service: service, seedbeds: sb, db: db}
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
	mux.HandleFunc("GET /seedbeds", h.listSeedbeds)
	mux.HandleFunc("POST /seedbeds", h.createSeedbed)
	mux.HandleFunc("GET /seedbeds/{id}", h.getSeedbed)
	mux.HandleFunc("PATCH /seedbeds/{id}", h.updateSeedbed)
	mux.HandleFunc("PATCH /seedbeds/{id}/visibility", h.setSeedbedVisibility)
	mux.HandleFunc("GET /seedbeds/{id}/members", h.listSeedbedMembers)
	mux.HandleFunc("POST /seedbeds/{id}/members", h.addSeedbedMember)
	mux.HandleFunc("DELETE /seedbeds/{id}/members/{userId}", h.removeSeedbedMember)
	return localCORS(mux)
}

func localCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isLocalOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			if !isLocalOrigin(origin) {
				writeError(w, http.StatusForbidden, "origen no permitido")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLocalOrigin(origin string) bool {
	return origin == "http://localhost" || strings.HasPrefix(origin, "http://localhost:") ||
		origin == "http://127.0.0.1" || strings.HasPrefix(origin, "http://127.0.0.1:") ||
		origin == "http://[::1]" || strings.HasPrefix(origin, "http://[::1]:")
}

type seedbedRequest struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	ResearchLine         string `json:"researchLine"`
	Objective            string `json:"objective"`
	ResponsibleTeacherID string `json:"responsibleTeacherId"`
	ContactEmail         string `json:"contactEmail"`
}
type visibilityRequest struct {
	Status     string `json:"status"`
	Visibility string `json:"visibility"`
}
type memberRequest struct {
	UserID string `json:"userId"`
}

func (h *Handler) seedbedReady(w http.ResponseWriter) bool {
	if h.seedbeds == nil {
		writeError(w, 500, "módulo de semilleros no configurado")
		return false
	}
	return true
}
func (h *Handler) actor(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	token, err := sessionToken(r)
	if err != nil {
		writeError(w, 401, "sesión requerida")
		return domain.User{}, false
	}
	u, err := h.service.CurrentUser(r.Context(), token)
	if err != nil {
		invalidSession(w, r, "sesión inválida o expirada")
		return domain.User{}, false
	}
	return u, true
}
func seedbedError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, 403, err.Error())
	case errors.Is(err, domain.ErrSeedbedNotFound):
		writeError(w, 404, err.Error())
	case errors.Is(err, domain.ErrInvalidSeedbed), errors.Is(err, domain.ErrInvalidTeacher), errors.Is(err, domain.ErrInvalidSeedbedStatus):
		writeError(w, 400, err.Error())
	case errors.Is(err, domain.ErrDuplicateSeedbed):
		writeError(w, 409, err.Error())
	default:
		writeError(w, 500, "no fue posible procesar el semillero")
	}
}
func (h *Handler) listSeedbeds(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	private := false
	if r.URL.Query().Get("includeInactive") == "true" {
		a, ok := h.actor(w, r)
		if !ok {
			return
		}
		if !strings.EqualFold(a.Role, "admin") {
			writeError(w, 403, "se requiere rol admin")
			return
		}
		private = true
	}
	items, err := h.seedbeds.List(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("researchLine"), queryInt(r, "page", 1), queryInt(r, "pageSize", 20), private)
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"seedbeds": items})
}
func queryInt(r *http.Request, key string, def int) int {
	var n int
	if _, err := fmt.Sscanf(r.URL.Query().Get(key), "%d", &n); err != nil || n < 1 {
		return def
	}
	return n
}
func (h *Handler) createSeedbed(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	if !strings.EqualFold(a.Role, "admin") && !strings.EqualFold(a.Role, "profesor") && !strings.EqualFold(a.Role, "docente") {
		writeError(w, 403, "se requiere rol docente o admin")
		return
	}
	var q seedbedRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	s, err := h.seedbeds.Create(r.Context(), seedbedapp.CreateCommand{q.Name, q.Description, q.ResearchLine, q.Objective, q.ResponsibleTeacherID, q.ContactEmail})
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 201, s)
}
func (h *Handler) getSeedbed(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	s, err := h.seedbeds.Find(r.Context(), r.PathValue("id"))
	if err != nil {
		seedbedError(w, err)
		return
	}
	if s.Status != domain.SeedbedPublished || s.Visibility != domain.SeedbedPublished {
		a, ok := h.actor(w, r)
		if !ok {
			return
		}
		if !strings.EqualFold(a.Role, "admin") && a.ID != s.ResponsibleTeacherID {
			writeError(w, 404, "semillero no encontrado")
			return
		}
	}
	writeJSON(w, 200, s)
}
func (h *Handler) updateSeedbed(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	if h.seedbeds.CanManage(r.Context(), a.ID, r.PathValue("id")) != nil {
		writeError(w, 403, "no tiene permisos para esta operación")
		return
	}
	var q seedbedRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	s, err := h.seedbeds.Update(r.Context(), r.PathValue("id"), seedbedapp.UpdateCommand{q.Name, q.Description, q.ResearchLine, q.Objective, q.ResponsibleTeacherID, q.ContactEmail})
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 200, s)
}
func (h *Handler) setSeedbedVisibility(w http.ResponseWriter, r *http.Request) {
	a, ok := h.actor(w, r)
	if !ok || !h.seedbedReady(w) {
		return
	}
	var q visibilityRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	status := q.Status
	if status == "" {
		status = q.Visibility
	}
	s, err := h.seedbeds.Visibility(r.Context(), a.ID, r.PathValue("id"), status)
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 200, s)
}
func (h *Handler) listSeedbedMembers(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	history := r.URL.Query().Get("history") == "true"
	seedbed, err := h.seedbeds.Find(r.Context(), r.PathValue("id"))
	if err != nil {
		seedbedError(w, err)
		return
	}
	if !history && (seedbed.Status != domain.SeedbedPublished || seedbed.Visibility != domain.SeedbedPublished) {
		actor, authenticated := h.actor(w, r)
		if !authenticated || (!strings.EqualFold(actor.Role, "admin") && actor.ID != seedbed.ResponsibleTeacherID) {
			writeError(w, http.StatusNotFound, "semillero no encontrado")
			return
		}
	}
	if history {
		a, ok := h.actor(w, r)
		if !ok {
			return
		}
		if !strings.EqualFold(a.Role, "admin") && seedbed.ResponsibleTeacherID != a.ID {
			writeError(w, 403, "historial restringido")
			return
		}
	}
	m, err := h.seedbeds.Members(r.Context(), r.PathValue("id"), history)
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"members": m})
}
func (h *Handler) addSeedbedMember(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	var q memberRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	if err := h.seedbeds.AddMember(r.Context(), a.ID, r.PathValue("id"), q.UserID); err != nil {
		seedbedError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) removeSeedbedMember(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	if err := h.seedbeds.RemoveMember(r.Context(), a.ID, r.PathValue("id"), r.PathValue("userId")); err != nil {
		seedbedError(w, err)
		return
	}
	w.WriteHeader(204)
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
	if errors.Is(err, domain.ErrUserNotRegistered) {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"code":   "REGISTRATION_REQUIRED",
			"error":  "registro incompleto",
			"action": "complete_registration",
		})
		return
	}
	if errors.Is(err, domain.ErrInvalidCredentials) || errors.Is(err, domain.ErrInactiveUser) {
		writeError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no fue posible iniciar sesión")
		return
	}
	// Rotate: the browser's previous session stops working as soon as a new one is issued.
	if previous, err := r.Cookie(sessionCookieName); err == nil && previous.Value != "" && previous.Value != result.Token {
		_ = h.service.Logout(r.Context(), previous.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: result.Token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	writeJSON(w, http.StatusCreated, map[string]any{"user": result.User})
}

func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	tokens := sessionTokens(r)
	if len(tokens) == 0 {
		writeError(w, http.StatusUnauthorized, "sesión requerida")
		return
	}
	// Revoke every credential the request presents. A stale Authorization header
	// must not make logout revoke the wrong token and leave the cookie session alive.
	for _, token := range tokens {
		if err := h.service.Logout(r.Context(), token); err != nil {
			writeError(w, http.StatusUnauthorized, "sesión inválida")
			return
		}
	}
	clearSessionCookie(w)
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
		invalidSession(w, r, "sesión inválida o expirada")
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
		Email: request.Email,
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

const sessionCookieName = "galash_session"

func sessionToken(r *http.Request) (string, error) {
	// Keep the authentication source consistent with POST /auth/sessions:
	// when a client sends Authorization, it must not be shadowed by an older
	// session cookie attached by the browser.
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		return bearerToken(authorization)
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	return "", errors.New("se requiere una sesión")
}

// sessionTokens returns every well-formed session credential on the request.
func sessionTokens(r *http.Request) []string {
	var tokens []string
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		tokens = append(tokens, cookie.Value)
	}
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		if token, err := bearerToken(authorization); err == nil {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

// invalidSession answers 401 and expires the browser cookie when the rejected
// credential came from it, so stale cookies are not replayed on later requests.
func invalidSession(w http.ResponseWriter, r *http.Request, message string) {
	if strings.TrimSpace(r.Header.Get("Authorization")) == "" {
		clearSessionCookie(w)
	}
	writeError(w, http.StatusUnauthorized, message)
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
