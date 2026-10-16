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
var ErrInvalidSeedbed = errors.New("datos del semillero inválidos")
var ErrSeedbedNotFound = errors.New("semillero no encontrado")
var ErrDuplicateSeedbed = errors.New("ya existe un semillero con ese nombre")
var ErrInvalidTeacher = errors.New("el responsable debe ser un docente activo")
var ErrInvalidSeedbedStatus = errors.New("estado o visibilidad inválidos")
var ErrAlreadyMember = errors.New("el usuario ya pertenece al semillero")
var ErrMemberNotFound = errors.New("integrante no encontrado")
