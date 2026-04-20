package sender

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"

	gomail "gopkg.in/gomail.v2"
)

// GmailSender sends emails through Gmail SMTP (STARTTLS on 587).
// Credentials are currently hardcoded for local testing.
type GmailSender struct {
	host string
	port int
	user string
	pass string

	from string

	dialer *gomail.Dialer
}

func NewGmailSender() *GmailSender {
	host := "smtp.gmail.com"
	port := 587
	user := "alisher.aitkazin03@gmail.com"
	pass := "xwgt ikvk gtig mzrn"

	d := gomail.NewDialer(host, port, user, pass)
	d.TLSConfig = &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}

	return &GmailSender{
		host:   host,
		port:   port,
		user:   user,
		pass:   pass,
		from:   user,
		dialer: d,
	}
}

// SendEmail sends an HTML email. If qrPNG is provided, it is attached as qr.png.
func (s *GmailSender) SendEmail(ctx context.Context, to, subject, htmlBody string, qrPNG []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if stringsTrim(to) == "" {
		return fmt.Errorf("gmail: missing recipient")
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
		return fmt.Errorf("gmail: send failed: %w", err)
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

