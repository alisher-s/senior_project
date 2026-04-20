package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	ticketingRepo "github.com/nu/student-event-ticketing-platform/ticketing/repository"
	"github.com/nu/student-event-ticketing-platform/payments/model"
	"github.com/nu/student-event-ticketing-platform/payments/repository"
)

type Service struct {
	repo         repository.PaymentRepository
	tickets      ticketingRepo.TicketRepository
	stripeClient *StripeClient
}

func New(repo repository.PaymentRepository, tickets ticketingRepo.TicketRepository, stripeClient *StripeClient) *Service {
	return &Service{repo: repo, tickets: tickets, stripeClient: stripeClient}
}

func (s *Service) Initiate(ctx context.Context, userID, eventID uuid.UUID, amount int64, currency string) (model.Payment, string /*provider_url*/, error) {
	now := time.Now().UTC()

	pm := model.Payment{
		UserID:   userID,
		EventID:  eventID,
		Amount:   amount,
		Currency: currency,
		Status:   model.PaymentStatusPending,
	}

	if s.stripeClient != nil {
		sessionID, sessionURL, err := s.stripeClient.CreateCheckoutSession(
			ctx, amount, currency,
			fmt.Sprintf("Event Ticket — %s", eventID.String()),
		)
		if err != nil {
			return model.Payment{}, "", err
		}
		pm.ProviderName = "stripe"
		pm.ProviderRef = sessionID

		payment, err := s.repo.CreateInitiation(ctx, pm)
		if err != nil {
			return model.Payment{}, "", err
		}
		_ = now
		return payment, sessionURL, nil
	}

	// Stub: payments not configured.
	pm.ProviderName = "stub"
	pm.ProviderRef = uuid.NewString()

	payment, err := s.repo.CreateInitiation(ctx, pm)
	if err != nil {
		return model.Payment{}, "", err
	}
	_ = now
	return payment, fmt.Sprintf("https://payments.example/checkout?provider_ref=%s", payment.ProviderRef), nil
}

func (s *Service) Webhook(ctx context.Context, providerRef string, status model.PaymentStatus) (model.Payment, error) {
	now := time.Now().UTC()

	payment, err := s.repo.UpdateStatusByProviderRef(ctx, providerRef, status, now)
	if err != nil {
		return model.Payment{}, err
	}

	if status == model.PaymentStatusSucceeded {
		// Auto-create ticket. Idempotent: if ticket already exists, ignore the error.
		b := make([]byte, 32)
		if _, randErr := rand.Read(b); randErr == nil {
			payload := base64.RawURLEncoding.EncodeToString(b)
			sum := sha256.Sum256([]byte(payload))
			qrHash := hex.EncodeToString(sum[:])
			_, regErr := s.tickets.RegisterTicket(ctx, payment.UserID, payment.EventID, qrHash, now)
			if regErr != nil && !errors.Is(regErr, ticketingRepo.ErrAlreadyRegistered) {
				return model.Payment{}, regErr
			}
		}
		return payment, nil
	}

	if status == model.PaymentStatusFailed || status == model.PaymentStatusCanceled {
		ticket, err := s.tickets.GetByEventAndUser(ctx, payment.EventID, payment.UserID)
		if err != nil {
			if errors.Is(err, ticketingRepo.ErrTicketNotFound) {
				return payment, nil
			}
			return model.Payment{}, err
		}

		_, err = s.tickets.CancelTicket(ctx, payment.UserID, ticket.ID, now, true)
		if err != nil {
			if errors.Is(err, ticketingRepo.ErrTicketAlreadyCancelled) {
				return payment, nil
			}
			return model.Payment{}, err
		}
	}

	return payment, nil
}
