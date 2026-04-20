package service

import "errors"

var (
	ErrEventNotFound = errors.New("event not found")
	ErrForbidden     = errors.New("forbidden")
)
