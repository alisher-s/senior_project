package handler

type InitiatePaymentRequestDTO struct {
	EventID string `json:"event_id" validate:"required"`
}

type InitiatePaymentResponseDTO struct {
	PaymentID   string `json:"payment_id"`
	ProviderRef string `json:"provider_ref"`
	ProviderURL string `json:"provider_url"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
}

type PaymentWebhookRequestDTO struct {
	ProviderRef string `json:"provider_ref" validate:"required"`
	Status      string `json:"status" validate:"required"`
}
