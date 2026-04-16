package model

import (
	"time"

	"github.com/google/uuid"
)

type EventStatus string

const (
	EventStatusDraft      EventStatus = "draft"
	EventStatusPublished  EventStatus = "published"
	EventStatusCancelled  EventStatus = "cancelled"
)

type Event struct {
	ID                  uuid.UUID
	Title               string
	Description         string
	StartsAt            time.Time
	CapacityTotal      int
	CapacityAvailable  int
	Status              EventStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

