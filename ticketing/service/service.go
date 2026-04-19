package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/google/uuid"

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

// ListMyTickets returns all tickets for the user with basic event info (empty slice if none).
func (s *Service) ListMyTickets(ctx context.Context, userID uuid.UUID) ([]model.TicketWithEvent, error) {
	return s.repo.GetUserTickets(ctx, userID)
}

