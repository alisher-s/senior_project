package service

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented") // kept for backward compatibility

	ErrPaymentNotFound = errors.New("payment not found")
)

