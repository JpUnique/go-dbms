package utils

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNotFound           = errors.New("resource not found")
	ErrAlreadyExists      = errors.New("resource already exists")
	ErrInternal           = errors.New("internal server error")
	ErrInvalidInput       = errors.New("invalid input")
)
