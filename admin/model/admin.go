package model

import (
	"time"

	"github.com/google/uuid"
)

type AdminAction string

const (
	AdminActionModerateEvent AdminAction = "moderate_event"
)

type AdminUserView struct {
	ID        uuid.UUID
	Email     string
	Role      string
	CreatedAt time.Time
}

// ModerationLog is a persisted admin audit row from admin_moderation_logs.
type ModerationLog struct {
	ID          uuid.UUID
	AdminUserID uuid.UUID
	EventID     *uuid.UUID
	Action      string
	Reason      *string
	CreatedAt   time.Time
}

