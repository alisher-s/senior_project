package service

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTicketConfirmationEmailHTML_IncludesFieldsAndInlineQR(t *testing.T) {
	t.Parallel()

	eventTitle := `My <Event> & Friends`
	startsAt := time.Date(2026, 4, 20, 10, 30, 0, 0, time.UTC)
	ticketID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	qr := "BASE64PNG=="

	html := TicketConfirmationEmailHTML(eventTitle, startsAt, ticketID, qr)

	if !strings.Contains(html, "Ticket confirmation") {
		t.Fatalf("expected header")
	}
	if strings.Contains(html, eventTitle) {
		t.Fatalf("expected event title to be escaped")
	}
	if !strings.Contains(html, "My &lt;Event&gt; &amp; Friends") {
		t.Fatalf("expected escaped event title")
	}
	if !strings.Contains(html, startsAt.Format(time.RFC3339)) {
		t.Fatalf("expected startsAt RFC3339")
	}
	if !strings.Contains(html, ticketID.String()) {
		t.Fatalf("expected ticket id")
	}
	if !strings.Contains(html, "data:image/png;base64,"+qr) {
		t.Fatalf("expected inline qr data url")
	}
}

func TestTicketConfirmationEmailHTML_NoQRDoesNotIncludeImage(t *testing.T) {
	t.Parallel()

	html := TicketConfirmationEmailHTML("Event", time.Now().UTC(), uuid.New(), "")
	if strings.Contains(html, "data:image/png;base64,") {
		t.Fatalf("did not expect inline qr")
	}
}
