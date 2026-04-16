package repository

import (
	"context"
	"time"

	"github.com/nu/student-event-ticketing-platform/payments/model"
)

type PaymentRepository interface {
	CreateInitiation(ctx context.Context, p model.Payment) (model.Payment, error)
	GetByProviderRef(ctx context.Context, providerRef string) (model.Payment, error)
	// UpdateStatus must be idempotent by providerRef (provider_ref unique).
	UpdateStatusByProviderRef(ctx context.Context, providerRef string, status model.PaymentStatus, now time.Time) (model.Payment, error)
}

