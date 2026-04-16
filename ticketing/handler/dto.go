package handler

type RegisterTicketRequestDTO struct {
	EventID string `json:"event_id" validate:"required"`
}

type RegisterTicketResponseDTO struct {
	TicketID   string `json:"ticket_id"`
	EventID    string `json:"event_id"`
	UserID     string `json:"user_id"`
	Status     string `json:"status"`
	QRPNGBase64 string `json:"qr_png_base64"`
	QRHashHex  string `json:"qr_hash_hex"`
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

