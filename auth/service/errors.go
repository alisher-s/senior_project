package service

import "errors"

var (
	ErrEmailNotAllowed       = errors.New("email domain not allowed")
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrRefreshTokenInvalid  = errors.New("invalid refresh token")
	ErrRefreshTokenConsumed = errors.New("refresh token revoked or expired")
)

