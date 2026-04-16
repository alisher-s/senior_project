package service

import "errors"

var (
	ErrInvalidRole     = errors.New("invalid role")
	ErrInvalidEventID  = errors.New("invalid event id")
	ErrInvalidAction   = errors.New("invalid moderation action")
	ErrEventNotFound   = errors.New("event not found")
)

