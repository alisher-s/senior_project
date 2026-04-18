package handler

type RegisterTicketRequestDTO struct {
	EventID string `json:"event_id" validate:"required"`
}

type RegisterTicketResponseDTO struct {
	TicketID string `json:"ticket_id"`
	EventID  string `json:"event_id"`
	UserID   string `json:"user_id"`
	Status   string `json:"status"`
	// QRPNGBase64 is raw PNG bytes encoded as standard Base64 (no data: URL prefix).
	QRPNGBase64 string `json:"qr_png_base64" example:"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="`
	// QRHashHex is the check-in digest sent to POST /tickets/use.
	QRHashHex string `json:"qr_hash_hex" example:"a1b2c3d4e5f6789012345678901234567890abcd"`
}

type CancelTicketResponseDTO struct {
	TicketID string `json:"ticket_id"`
	EventID  string `json:"event_id"`
	UserID   string `json:"user_id"`
	Status   string `json:"status"`
}

type UseTicketRequestDTO struct {
	QRHashHex string `json:"qr_hash_hex" validate:"required"`
}

type UseTicketResponseDTO struct {
	TicketID string `json:"ticket_id"`
	EventID  string `json:"event_id"`
	UserID   string `json:"user_id"`
	Status   string `json:"status"`
}
