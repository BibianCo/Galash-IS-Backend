package domain

import (
	"errors"
	"strings"
)

var ErrRoleNotFound = errors.New("rol no encontrado")
var ErrInvalidPermission = errors.New("permiso inválido")
var ErrUserNotFound = errors.New("usuario no encontrado")

// Role is an entry of the fixed catalog. ID is the value persisted in usuarios.rol;
// Name is the label the contract uses.
type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Permission struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Roles is the fixed catalog: the API never creates roles outside it.
var Roles = []Role{
	{ID: "visitante", Name: "Visitante"},
	{ID: "estudiante", Name: "Estudiante"},
	{ID: "profesor", Name: "Docente"},
	{ID: "admin", Name: "Admin"},
}

// FindRole accepts the persisted ID or the contract label, ignoring case,
// so "docente" resolves to the stored "profesor".
func FindRole(id string) (Role, bool) {
	id = strings.TrimSpace(id)
	for _, r := range Roles {
		if strings.EqualFold(r.ID, id) || strings.EqualFold(r.Name, id) {
			return r, true
		}
	}
	return Role{}, false
}

var ErrSelfPermissionChange = errors.New("no puede modificar los permisos de su propio rol")

// Dashboard is the access context the frontend uses to adapt the panel.
type Dashboard struct {
	User        User         `json:"user"`
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
}
