package handler

import "time"

type CreateEventRequestDTO struct {
	Title          string `json:"title" validate:"required,min=3,max=120"`
	Description    string `json:"description" validate:"max=2000"`
	StartsAt       time.Time `json:"starts_at" validate:"required"`
	CapacityTotal  int    `json:"capacity_total" validate:"required,min=1,max=100000"`
}

type UpdateEventRequestDTO struct {
	Title         *string   `json:"title,omitempty" validate:"omitempty,min=3,max=120"`
	Description   *string   `json:"description,omitempty" validate:"omitempty,max=2000"`
	StartsAt      *time.Time `json:"starts_at,omitempty"`
	CapacityTotal *int      `json:"capacity_total,omitempty" validate:"omitempty,min=1,max=100000"`
	Status        *string   `json:"status,omitempty" validate:"omitempty,oneof=draft published cancelled"`
}

type EventDTO struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	StartsAt           time.Time `json:"starts_at"`
	CapacityTotal      int       `json:"capacity_total"`
	CapacityAvailable  int       `json:"capacity_available"`
	Status             string    `json:"status"`
	ModerationStatus   string    `json:"moderation_status"`
}

type ListEventsResponseDTO struct {
	Items []EventDTO `json:"items"`
	Limit int         `json:"limit"`
	Offset int        `json:"offset"`
}

