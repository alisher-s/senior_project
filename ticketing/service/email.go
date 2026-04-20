package service

import (
	"fmt"
	"html"
	"time"

	"github.com/google/uuid"
)

func TicketConfirmationEmailHTML(eventTitle string, startsAt time.Time, ticketID uuid.UUID, qrPNGBase64 string) string {
	titleEsc := html.EscapeString(eventTitle)
	ticketIDEsc := html.EscapeString(ticketID.String())
	startsAtStr := startsAt.UTC().Format(time.RFC3339)

	// QR is rendered inline as a data URL so the worker can stay attachment-free.
	qrImg := ""
	if qrPNGBase64 != "" {
		qrImg = fmt.Sprintf(
			`<p><img alt="Ticket QR" src="data:image/png;base64,%s" style="max-width:256px;height:auto"/></p>`,
			qrPNGBase64,
		)
	}

	return fmt.Sprintf(`<!doctype html>
<html>
  <body style="font-family: Arial, sans-serif; line-height: 1.4;">
    <h2>Ticket confirmation</h2>
    <p><b>Event:</b> %s</p>
    <p><b>Date/time (UTC):</b> %s</p>
    <p><b>Ticket ID:</b> %s</p>
    %s
    <p>You can also access your QR code in the app.</p>
  </body>
</html>`, titleEsc, startsAtStr, ticketIDEsc, qrImg)
}
