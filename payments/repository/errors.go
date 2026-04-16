package repository

import "errors"

var (
	ErrProviderRefExists = errors.New("provider_ref already exists")
	ErrPaymentNotFound   = errors.New("payment not found")
)

