package repository

import "errors"

var (
	ErrEventNotFound     = errors.New("event not found")
	ErrCapacityFull      = errors.New("event capacity full")
	ErrAlreadyRegistered = errors.New("already registered")

	ErrTicketNotFound       = errors.New("ticket not found")
	ErrTicketAlreadyCancelled = errors.New("ticket already cancelled")
	ErrTicketAlreadyUsed     = errors.New("ticket already used")
	ErrTicketCannotBeUsed   = errors.New("ticket cannot be used")
)

