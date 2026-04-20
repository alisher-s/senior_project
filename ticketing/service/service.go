package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"

	"github.com/nu/student-event-ticketing-platform/ticketing/model"
	"github.com/nu/student-event-ticketing-platform/ticketing/repository"
)

type Service struct {
	repo repository.TicketRepository
}

func New(repo repository.TicketRepository) *Service {
	return &Service{repo: repo}
}

func GenerateRandomQRPayload(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Base64URL => no unpredictable characters in QR.
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func qrHashHex(payload string) string {
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func renderQRPNGBase64(payload string) (string, error) {
	png, err := qrcode.Encode(payload, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(png), nil
}

func (s *Service) RegisterTicket(ctx context.Context, userID, eventID uuid.UUID) (model.Ticket, string, error) {
	// Cryptographically safe payload: prevents predictable/guessable QR values.
	// Only the SHA-256 hash of the payload is persisted; raw payload is not stored in DB.
	payload, err := GenerateRandomQRPayload(32)
	if err != nil {
		return model.Ticket{}, "", err
	}
	hashHex := qrHashHex(payload)

	now := time.Now().UTC()
	ticket, err := s.repo.RegisterTicket(ctx, userID, eventID, hashHex, now)
	if err != nil {
		return model.Ticket{}, "", err
	}

	qrPNGBase64, err := renderQRPNGBase64(payload)
	if err != nil {
		// Ticket is already created; returning error here is safer than leaking payload / changing DB state.
		return model.Ticket{}, "", err
	}
	return ticket, qrPNGBase64, nil
}

func (s *Service) CancelTicket(ctx context.Context, userID, ticketID uuid.UUID) (model.Ticket, error) {
	now := time.Now().UTC()
	return s.repo.CancelTicket(ctx, userID, ticketID, now, false)
}

func (s *Service) UseTicketByQRHash(ctx context.Context, qrHashHex string) (model.Ticket, error) {
	now := time.Now().UTC()
	return s.repo.UseTicketByQRHash(ctx, qrHashHex, now)
}

func overlayTicketExpiry(t model.TicketWithEvent, now time.Time) model.TicketWithEvent {
	if t.Status != model.TicketStatusActive {
		return t
	}
	if now.After(model.EventEndInstant(t.EventStartsAt, t.EventEndsAt)) {
		t.Status = model.TicketStatusExpired
	}
	return t
}

// GetUserTickets returns tickets for the user with event metadata. Active tickets are returned with status `expired` when the event end time has passed (computed from events.end_at or events.starts_at).
func (s *Service) GetUserTickets(ctx context.Context, userID uuid.UUID) ([]model.TicketWithEvent, error) {
	rows, err := s.repo.GetUserTickets(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for i := range rows {
		rows[i] = overlayTicketExpiry(rows[i], now)
	}
	return rows, nil
}

// ListMyTickets is an alias for GetUserTickets.
func (s *Service) ListMyTickets(ctx context.Context, userID uuid.UUID) ([]model.TicketWithEvent, error) {
	return s.GetUserTickets(ctx, userID)
}
