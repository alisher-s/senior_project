package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/auth/model"
)

type UserRepository interface {
	UpdateUserRole(ctx context.Context, id uuid.UUID, role model.Role) (model.User, error)
}

type Service struct {
	users UserRepository
}

func New(users UserRepository) *Service {
	return &Service{users: users}
}

func (s *Service) ModerateEvent(ctx context.Context, eventID string, action string) error {
	_ = ctx
	_ = eventID
	_ = action
	return ErrNotImplemented
}

// SetUserRole updates the target user's role and revokes their refresh tokens (callers must re-login).
func (s *Service) SetUserRole(ctx context.Context, userID uuid.UUID, role string) (model.User, error) {
	r, err := normalizeRole(role)
	if err != nil {
		return model.User{}, err
	}
	u, err := s.users.UpdateUserRole(ctx, userID, r)
	if err != nil {
		return model.User{}, err
	}
	return u, nil
}

func normalizeRole(s string) (model.Role, error) {
	r := model.Role(strings.ToLower(strings.TrimSpace(s)))
	switch r {
	case model.RoleStudent, model.RoleOrganizer, model.RoleAdmin:
		return r, nil
	default:
		return "", ErrInvalidRole
	}
}
