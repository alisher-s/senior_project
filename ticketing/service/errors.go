package service

import "errors"

var (
	ErrEventNotFound     = errors.New("event not found")
	ErrCapacityFull      = errors.New("capacity full")
	ErrAlreadyRegistered = errors.New("already registered")
	ErrEventNotPublished = errors.New("event is not published")
	ErrEventNotApproved  = errors.New("event is not approved for registration")
	ErrEventCancelled    = errors.New("event is cancelled")
	ErrEventRegistrationClosed = errors.New("event registration is closed")
	ErrCancellationNotAllowed = errors.New("ticket cannot be cancelled after event start")
	ErrCheckInNotOpenYet = errors.New("check-in is not open yet")

	ErrEventRequiresPayment  = errors.New("event requires payment")

	ErrTicketNotFound        = errors.New("ticket not found")
	ErrTicketAlreadyCancelled = errors.New("ticket already cancelled")
	ErrTicketAlreadyUsed    = errors.New("ticket already used")
	ErrTicketCannotBeUsed   = errors.New("ticket cannot be used")
)

