package service

import "context"

// NoopSender does not send mail; Send always succeeds. Used when SMTP_HOST is unset.
type NoopSender struct{}

func (NoopSender) Send(ctx context.Context, to, subject, body string) error {
	_ = ctx
	_ = to
	_ = subject
	_ = body
	return nil
}
