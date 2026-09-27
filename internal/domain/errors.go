package domain

import "errors"

var ErrIncompleteIdentity = errors.New("el token debe incluir nombre y correo electrónico")
var ErrInvalidIdentity = errors.New("token de Firebase inválido")
