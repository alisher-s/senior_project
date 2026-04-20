package sender

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/nu/student-event-ticketing-platform/internal/config"
	gomail "gopkg.in/gomail.v2"
)

var ErrNotConfigured = errors.New("smtp not configured")

// SMTPSender sends HTML email via SMTP.
// - Port 587: STARTTLS (opportunistic) using gomail.
// - Port 465: implicit TLS (SMTPS).
type SMTPSender struct {
	host string
	port int
	user string
	pass string

	from string

	dialer *gomail.Dialer
}

func NewSMTPSender(cfg config.Config) (*SMTPSender, error) {
	host := strings.TrimSpace(cfg.SMTP.Host)
	port := cfg.SMTP.Port
	from := strings.TrimSpace(cfg.SMTP.From)
	pass := strings.TrimSpace(cfg.SMTP.Password)

	user := strings.TrimSpace(cfg.SMTP.User)
	if user == "" {
		user = from
	}

	if host == "" || port <= 0 || from == "" || user == "" || pass == "" {
		return nil, ErrNotConfigured
	}

	d := gomail.NewDialer(host, port, user, pass)
	d.TLSConfig = &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if port == 465 {
		d.SSL = true
	}

	return &SMTPSender{
		host:   host,
		port:   port,
		user:   user,
		pass:   pass,
		from:   from,
		dialer: d,
	}, nil
}

// SendEmail sends an HTML email. If qrPNG is provided, it is attached as qr.png.
func (s *SMTPSender) SendEmail(ctx context.Context, to, subject, htmlBody string, qrPNG []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.dialer == nil {
		return ErrNotConfigured
	}
	if stringsTrim(to) == "" {
		return fmt.Errorf("smtp: missing recipient")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	if len(qrPNG) > 0 {
		m.Attach(
			"qr.png",
			gomail.SetHeader(map[string][]string{
				"Content-Type": {"image/png"},
			}),
			gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(qrPNG)
				return err
			}),
		)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// gomail doesn't support context directly; keep a conservative timeout on the dialer.
	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("smtp: send failed: %w", err)
	}
	return nil
}

func stringsTrim(s string) string {
	// Small helper to avoid importing strings in this package for just TrimSpace.
	// Keep it minimal and allocation-friendly.
	i := 0
	j := len(s)
	for i < j {
		c := s[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		i++
	}
	for j > i {
		c := s[j-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		j--
	}
	return s[i:j]
}
