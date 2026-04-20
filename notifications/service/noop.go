package service

import "context"

// NoopSender does not send mail; SendEmail always succeeds.
type NoopSender struct{}

func (NoopSender) SendEmail(ctx context.Context, to, subject, htmlBody string, qrPNG []byte) error {
	_ = ctx
	_ = to
	_ = subject
	_ = htmlBody
	_ = qrPNG
	return nil
}
