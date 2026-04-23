package core

import (
	"errors"
)

var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrNotFound        = errors.New("not found")
	ErrPayloadTooLarge = errors.New("payload too large")
)
