package repository

import "errors"

var (
	ErrProviderRefExists = errors.New("provider_ref already exists")
	ErrPaymentNotFound   = errors.New("payment not found")
	// ErrPaymentsDisabled is returned by the stub repository until Postgres-backed payments are enabled.
	ErrPaymentsDisabled = errors.New("payments not enabled")
)

