package model

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID                  uuid.UUID
	Title               string
	Description         string
	StartsAt            time.Time
	CapacityTotal      int
	CapacityAvailable  int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

