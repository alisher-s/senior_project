package repository

import (
	"context"

	"time"

	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/ticketing/model"
)

type TicketRepository interface {
	RegisterTicket(ctx context.Context, userID uuid.UUID, eventID uuid.UUID, qrHashHex string, now time.Time) (model.Ticket, error)
	GetByEventAndUser(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) (model.Ticket, error)

	CancelTicket(ctx context.Context, userID uuid.UUID, ticketID uuid.UUID, now time.Time) (model.Ticket, error)
	UseTicketByQRHash(ctx context.Context, qrHashHex string, now time.Time) (model.Ticket, error)
}

