package domain

import "errors"

var ErrIncompleteIdentity = errors.New("el token debe incluir nombre y correo electrónico")
var ErrInvalidIdentity = errors.New("token de Firebase inválido")
var ErrInvalidCredentials = errors.New("credenciales inválidas")
var ErrInactiveUser = errors.New("la cuenta está inactiva")
var ErrInvalidSession = errors.New("sesión inválida o expirada")
var ErrInvalidRecoveryToken = errors.New("token de recuperación inválido o expirado")
