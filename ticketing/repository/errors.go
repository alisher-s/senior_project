package repository

import "errors"

var (
	ErrEventNotFound     = errors.New("event not found")
	ErrCapacityFull      = errors.New("event capacity full")
	ErrAlreadyRegistered = errors.New("already registered")
	ErrEventNotPublished = errors.New("event is not published")
	ErrEventNotApproved  = errors.New("event is not approved for listing")
	ErrEventCancelled    = errors.New("event is cancelled")
	// ErrEventRegistrationClosed is returned when registration is attempted at/after event start.
	ErrEventRegistrationClosed = errors.New("event registration is closed")
	// ErrCancellationNotAllowed blocks attendee self-cancel after the event has started (payment webhooks may bypass).
	ErrCancellationNotAllowed = errors.New("ticket cannot be cancelled after event start")
	// ErrCheckInNotOpenYet is returned when scanning a ticket before the event start time.
	ErrCheckInNotOpenYet = errors.New("check-in is not open yet")

	ErrTicketNotFound       = errors.New("ticket not found")
	ErrTicketAlreadyCancelled = errors.New("ticket already cancelled")
	ErrTicketAlreadyUsed     = errors.New("ticket already used")
	ErrTicketCannotBeUsed   = errors.New("ticket cannot be used")
	ErrTicketExpired        = errors.New("ticket expired")
)

