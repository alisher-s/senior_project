package service

import "context"

// Sender delivers email notifications. Implementations can be swapped in tests (e.g. mocks).
type Sender interface {
	// SendEmail sends an email where body is HTML. If qrPNG is provided, it is attached as qr.png.
	SendEmail(ctx context.Context, to, subject, htmlBody string, qrPNG []byte) error
}
