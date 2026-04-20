package handler

import "time"

type CreateEventRequestDTO struct {
	Title         string    `json:"title" validate:"required,min=3,max=120"`
	Description   string    `json:"description" validate:"max=2000"`
	CoverImageURL string    `json:"cover_image_url,omitempty" validate:"omitempty,max=2048"`
	StartsAt      time.Time `json:"starts_at" validate:"required" example:"2026-01-01T10:00:00Z"`
	CapacityTotal int       `json:"capacity_total" validate:"required,min=1,max=100000"`
	// PriceAmount is in the smallest currency unit (cents for USD, tenge for KZT). 0 = free event.
	PriceAmount   int64  `json:"price_amount" validate:"min=0"`
	PriceCurrency string `json:"price_currency" validate:"omitempty,len=3" example:"KZT"`
}

type UpdateEventRequestDTO struct {
	Title         *string    `json:"title,omitempty" validate:"omitempty,min=3,max=120"`
	Description   *string    `json:"description,omitempty" validate:"omitempty,max=2000"`
	CoverImageURL *string    `json:"cover_image_url,omitempty" validate:"omitempty,max=2048"`
	StartsAt      *time.Time `json:"starts_at,omitempty" example:"2026-01-01T10:00:00Z"`
	CapacityTotal *int       `json:"capacity_total,omitempty" validate:"omitempty,min=1,max=100000"`
	Status        *string    `json:"status,omitempty" validate:"omitempty,oneof=draft published cancelled"`
	PriceAmount   *int64     `json:"price_amount,omitempty" validate:"omitempty,min=0"`
	PriceCurrency *string    `json:"price_currency,omitempty" validate:"omitempty,len=3"`
}

type EventDTO struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	CoverImageURL     string    `json:"cover_image_url,omitempty"`
	StartsAt          time.Time `json:"starts_at"`
	CapacityTotal     int       `json:"capacity_total"`
	CapacityAvailable int       `json:"capacity_available"`
	Status            string    `json:"status"`
	ModerationStatus  string    `json:"moderation_status"`
	PriceAmount       int64     `json:"price_amount"`
	PriceCurrency     string    `json:"price_currency"`
	IsFree            bool      `json:"is_free"`
}

type ListEventsResponseDTO struct {
	Items  []EventDTO `json:"items"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}
