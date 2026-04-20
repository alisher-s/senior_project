package service

import "context"

// Sender delivers email notifications. Implementations can be swapped in tests (e.g. mocks).
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}
