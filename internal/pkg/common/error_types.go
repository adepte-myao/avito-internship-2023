package common

import "errors"

var (
	ErrJSONUnmarshalling = errors.New("error during JSON unmarshalling")
	ErrBindFailed        = errors.New("parameters bind failed")
	ErrValidationFailed  = errors.New("validation failed")
)
