package model

import (
	"time"

	"github.com/google/uuid"
)

type TicketStatus string

const (
	TicketStatusActive    TicketStatus = "active"
	TicketStatusUsed      TicketStatus = "used"
	TicketStatusCancelled TicketStatus = "cancelled"
	// TicketStatusExpired is returned in API responses when the event window has passed; it is not persisted in tickets.status.
	TicketStatusExpired TicketStatus = "expired"
)

type Ticket struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	UserID    uuid.UUID
	Status    TicketStatus
	QRHashHex string
	CreatedAt time.Time
}

// EventEndInstant returns the event end time for business rules: end_at when set, otherwise starts_at.
func EventEndInstant(startsAt time.Time, endAt *time.Time) time.Time {
	if endAt != nil {
		return *endAt
	}
	return startsAt
}

// TicketWithEvent is a ticket plus basic event fields from a tickets↔events JOIN.
type TicketWithEvent struct {
	Ticket
	EventTitle    string
	EventStartsAt time.Time
	// EventEndsAt is nil when the event has no end_at in the database; event end for expiry is computed as COALESCE(end_at, starts_at).
	EventEndsAt   *time.Time
	EventLocation string
}
