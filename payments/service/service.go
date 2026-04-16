package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	ticketingRepo "github.com/nu/student-event-ticketing-platform/ticketing/repository"
	"github.com/nu/student-event-ticketing-platform/payments/model"
	"github.com/nu/student-event-ticketing-platform/payments/repository"
)

type Service struct {
	repo    repository.PaymentRepository
	tickets ticketingRepo.TicketRepository
}

func New(repo repository.PaymentRepository, tickets ticketingRepo.TicketRepository) *Service {
	return &Service{repo: repo, tickets: tickets}
}

func (s *Service) Initiate(ctx context.Context, userID, eventID uuid.UUID, amount int64, currency string) (model.Payment, string /*provider_url*/, error) {
	providerRef := uuid.NewString()
	now := time.Now().UTC()

	pm := model.Payment{
		UserID:      userID,
		EventID:     eventID,
		Amount:      amount,
		Currency:    currency,
		Status:      model.PaymentStatusPending,
		ProviderName: "stub",
		ProviderRef:  providerRef,
	}

	payment, err := s.repo.CreateInitiation(ctx, pm)
	if err != nil {
		return model.Payment{}, "", err
	}

	// Stub provider URL. Real integration will redirect to the actual payment gateway.
	providerURL := fmt.Sprintf("https://payments.example/checkout?provider_ref=%s", payment.ProviderRef)
	_ = now // reserved for future provider-request fields

	return payment, providerURL, nil
}

func (s *Service) Webhook(ctx context.Context, providerRef string, status model.PaymentStatus) (model.Payment, error) {
	now := time.Now().UTC()

	payment, err := s.repo.UpdateStatusByProviderRef(ctx, providerRef, status, now)
	if err != nil {
		return model.Payment{}, err
	}

	// Successful payment: tickets are issued/reserved in ticketing layer already.
	if status == model.PaymentStatusSucceeded {
		return payment, nil
	}

	// Failed/canceled payment: release reserved seat by cancelling the user's ticket.
	if status == model.PaymentStatusFailed || status == model.PaymentStatusCanceled {
		ticket, err := s.tickets.GetByEventAndUser(ctx, payment.EventID, payment.UserID)
		if err != nil {
			if errors.Is(err, ticketingRepo.ErrTicketNotFound) {
				return payment, nil // nothing to cancel (idempotent)
			}
			return model.Payment{}, err
		}

		_, err = s.tickets.CancelTicket(ctx, payment.UserID, ticket.ID, now, true)
		if err != nil {
			if errors.Is(err, ticketingRepo.ErrTicketAlreadyCancelled) {
				return payment, nil // idempotent
			}
			return model.Payment{}, err
		}
	}

	return payment, nil
}

