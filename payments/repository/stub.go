package repository

import (
	"context"
	"time"

	"github.com/nu/student-event-ticketing-platform/payments/model"
)

// Stub is a no-op payment store for development when real payments are not wired yet.
type Stub struct{}

func NewStub() *Stub {
	return &Stub{}
}

func (s *Stub) CreateInitiation(ctx context.Context, p model.Payment) (model.Payment, error) {
	_ = ctx
	_ = p
	return model.Payment{}, ErrPaymentsDisabled
}

func (s *Stub) GetByProviderRef(ctx context.Context, providerRef string) (model.Payment, error) {
	_ = ctx
	_ = providerRef
	return model.Payment{}, ErrPaymentsDisabled
}

func (s *Stub) UpdateStatusByProviderRef(ctx context.Context, providerRef string, status model.PaymentStatus, now time.Time) (model.Payment, error) {
	_ = ctx
	_ = providerRef
	_ = status
	_ = now
	return model.Payment{}, ErrPaymentsDisabled
}
