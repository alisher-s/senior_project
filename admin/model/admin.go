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

