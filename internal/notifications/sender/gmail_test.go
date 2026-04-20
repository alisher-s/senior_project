package sender

import (
	"errors"
	"testing"

	"github.com/nu/student-event-ticketing-platform/internal/config"
)

func TestNewSMTPSender_NotConfigured(t *testing.T) {
	t.Parallel()

	var cfg config.Config
	cfg.SMTP.Port = 587

	_, err := NewSMTPSender(cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestNewSMTPSender_UserFallbackToFrom(t *testing.T) {
	t.Parallel()

	var cfg config.Config
	cfg.SMTP.Host = "smtp.example.com"
	cfg.SMTP.Port = 587
	cfg.SMTP.From = "noreply@example.com"
	cfg.SMTP.Password = "pw"
	cfg.SMTP.User = ""

	s, err := NewSMTPSender(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.user != cfg.SMTP.From {
		t.Fatalf("expected user=%q, got %q", cfg.SMTP.From, s.user)
	}
}

func TestNewSMTPSender_SSLPort465(t *testing.T) {
	t.Parallel()

	var cfg config.Config
	cfg.SMTP.Host = "smtp.example.com"
	cfg.SMTP.Port = 465
	cfg.SMTP.From = "noreply@example.com"
	cfg.SMTP.Password = "pw"
	cfg.SMTP.User = "smtp-user"

	s, err := NewSMTPSender(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.dialer == nil || !s.dialer.SSL {
		t.Fatalf("expected dialer.SSL=true for port 465")
	}
}
