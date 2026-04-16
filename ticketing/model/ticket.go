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

