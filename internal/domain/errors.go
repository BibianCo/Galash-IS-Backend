package domain

import "errors"

var ErrIncompleteIdentity = errors.New("el token debe incluir nombre y correo electrónico")
var ErrInvalidIdentity = errors.New("token de Firebase inválido")
var ErrInvalidCredentials = errors.New("credenciales inválidas")
var ErrUserNotRegistered = errors.New("usuario no registrado")
var ErrInactiveUser = errors.New("la cuenta está inactiva")
var ErrInvalidSession = errors.New("sesión inválida o expirada")
var ErrInvalidRecoveryToken = errors.New("token de recuperación inválido o expirado")
var ErrForbidden = errors.New("no tiene permisos para esta operación")
var ErrInvalidRole = errors.New("rol inválido")
var ErrSelfRoleChange = errors.New("no puede cambiar su propio rol")
