package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"

	roleapp "github.com/galash-uptc/backend/internal/application/role"
	"github.com/galash-uptc/backend/internal/domain"
)

// WithRoles composes the roles and permissions module (Módulo 11).
func (h *Handler) WithRoles(roles *roleapp.Service) *Handler {
	h.roles = roles
	return h
}

func (h *Handler) roleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /roles", h.listRoles)
	mux.HandleFunc("GET /permissions", h.listPermissions)
	mux.HandleFunc("GET /roles/{roleId}/permissions", h.rolePermissions)
	mux.HandleFunc("PUT /roles/{roleId}/permissions", h.setRolePermissions)
	mux.HandleFunc("GET /users/{userId}/roles", h.userRoles)
	mux.HandleFunc("GET /users/me/dashboard", h.dashboard)
}

func (h *Handler) roleReady(w http.ResponseWriter) bool {
	if h.roles == nil {
		writeError(w, 500, "módulo de roles no configurado")
		return false
	}
	return true
}

func roleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrRoleNotFound), errors.Is(err, domain.ErrUserNotFound):
		writeError(w, 404, err.Error())
	case errors.Is(err, domain.ErrInvalidPermission):
		writeError(w, 400, err.Error())
	case errors.Is(err, domain.ErrSelfPermissionChange):
		writeError(w, 409, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, 403, "rol no autorizado")
	default:
		writeError(w, 500, "no fue posible procesar roles y permisos")
	}
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	if !h.roleReady(w) {
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"roles": h.roles.Roles()})
}

func (h *Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	if !h.roleReady(w) {
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	items, err := h.roles.Permissions(r.Context())
	if err != nil {
		roleError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"permissions": items})
}

func (h *Handler) rolePermissions(w http.ResponseWriter, r *http.Request) {
	if !h.roleReady(w) {
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	role, items, err := h.roles.RolePermissions(r.Context(), r.PathValue("roleId"))
	if err != nil {
		roleError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"role": role, "permissions": items})
}

func (h *Handler) setRolePermissions(w http.ResponseWriter, r *http.Request) {
	if !h.roleReady(w) {
		return
	}
	actor, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	// A missing "permissions" field is rejected instead of clearing the role.
	var q struct {
		Permissions *[]string `json:"permissions"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil || q.Permissions == nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	role, items, err := h.roles.SetRolePermissions(r.Context(), actor, r.PathValue("roleId"), *q.Permissions)
	if err != nil {
		roleError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"role": role, "permissions": items})
}

func (h *Handler) userRoles(w http.ResponseWriter, r *http.Request) {
	if !h.roleReady(w) {
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	roles, err := h.roles.UserRoles(r.Context(), r.PathValue("userId"))
	if err != nil {
		roleError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"userId": r.PathValue("userId"), "roles": roles})
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	if !h.roleReady(w) {
		return
	}
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	d, err := h.roles.Dashboard(r.Context(), actor)
	if err != nil {
		roleError(w, err)
		return
	}
	writeJSON(w, 200, d)
}
