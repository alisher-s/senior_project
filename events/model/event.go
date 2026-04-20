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

// ModerationStatus is admin review state (distinct from EventStatus lifecycle).
type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "pending"
	ModerationApproved ModerationStatus = "approved"
	ModerationRejected ModerationStatus = "rejected"
)

type Event struct {
	ID                  uuid.UUID
	Title               string
	Description         string
	CoverImageURL       string
	StartsAt            time.Time
	CapacityTotal      int
	CapacityAvailable  int
	Status              EventStatus
	ModerationStatus    ModerationStatus
	ModeratedBy         *uuid.UUID
	OrganizerID         *uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

