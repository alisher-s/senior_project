package model

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleStudent   Role = "student"
	RoleOrganizer Role = "organizer"
	RoleAdmin     Role = "admin"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	// Role is the legacy column on users.role (kept in sync for backward compatibility).
	Role         Role
	ActiveRoles  []Role
	PendingRoles []Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

