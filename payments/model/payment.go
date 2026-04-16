package model

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCanceled  PaymentStatus = "canceled"
)

type Payment struct {
	ID           uuid.UUID
	UserID      uuid.UUID
	EventID     uuid.UUID
	Amount       int64
	Currency     string
	Status       PaymentStatus
	ProviderName string
	ProviderRef  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

