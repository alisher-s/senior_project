package handler

type InitiatePaymentRequestDTO struct {
	EventID  string `json:"event_id" validate:"required"`
	Amount   int64  `json:"amount" validate:"required,gt=0"`
	Currency string `json:"currency" validate:"required,min=3,max=3"`
}

type InitiatePaymentResponseDTO struct {
	PaymentID   string `json:"payment_id"`
	ProviderRef string `json:"provider_ref"`
	ProviderURL string `json:"provider_url"`
}

type PaymentWebhookRequestDTO struct {
	ProviderRef string `json:"provider_ref" validate:"required"`
	Status      string `json:"status" validate:"required"`
}

