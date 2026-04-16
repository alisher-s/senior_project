package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/auth/model"
)

type UserRepository interface {
	CreateUser(ctx context.Context, email string, passwordHash string, role model.Role) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error)
	// UpdateUserRole sets role and revokes refresh tokens so JWT/refresh reflect the new role.
	UpdateUserRole(ctx context.Context, id uuid.UUID, role model.Role) (model.User, error)
}

type RefreshTokenRepository interface {
	RevokeTokensByUserID(ctx context.Context, userID uuid.UUID) error
	InsertRefreshToken(ctx context.Context, jti uuid.UUID, userID uuid.UUID, expiresAt time.Time) error
	// RotateRefreshToken revokes all existing refresh tokens for the user and inserts a new one.
	// It uses a transaction + a row lock to serialize concurrent logins per user.
	RotateRefreshToken(ctx context.Context, userID uuid.UUID, jti uuid.UUID, expiresAt time.Time) error
	// ConsumeRefreshToken atomically validates and revokes refresh token (single-use).
	// Returns true only if the token existed, was not revoked and not expired.
	ConsumeRefreshToken(ctx context.Context, jti uuid.UUID, userID uuid.UUID, now time.Time) (bool, error)
}

