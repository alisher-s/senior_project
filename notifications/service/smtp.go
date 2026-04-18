package service

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/nu/student-event-ticketing-platform/internal/config"
)

// SMTPSender sends mail via net/smtp (PLAIN auth + optional STARTTLS from SendMail).
type SMTPSender struct {
	host     string
	port     int
	from     string
	password string
}

func NewSMTPSender(cfg config.Config) *SMTPSender {
	return &SMTPSender{
		host:     cfg.SMTP.Host,
		port:     cfg.SMTP.Port,
		from:     cfg.SMTP.From,
		password: cfg.SMTP.Password,
	}
}

func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.host == "" {
		return fmt.Errorf("smtp: host not configured")
	}
	if s.from == "" {
		return fmt.Errorf("smtp: from address not configured")
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.from, s.password, s.host)

	var msg strings.Builder
	fmt.Fprintf(&msg, "From: %s\r\n", s.from)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg.String()))
}
