package model

import (
	"time"

	"github.com/google/uuid"
)

type TicketStatus string

const (
	TicketStatusActive   TicketStatus = "active"
	TicketStatusUsed     TicketStatus = "used"
	TicketStatusCancelled TicketStatus = "cancelled"
)

type Ticket struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	UserID    uuid.UUID
	Status    TicketStatus
	QRHashHex string
	CreatedAt time.Time
}

// TicketWithEvent is a ticket plus basic event fields from a tickets↔events JOIN.
type TicketWithEvent struct {
	Ticket
	EventTitle    string
	EventStartsAt time.Time
	EventLocation string
}

