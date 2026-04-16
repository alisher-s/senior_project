package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/auth/model"
	eventsmodel "github.com/nu/student-event-ticketing-platform/events/model"
	eventsrepo "github.com/nu/student-event-ticketing-platform/events/repository"
	notificationsmodel "github.com/nu/student-event-ticketing-platform/notifications/model"
)

type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error)
	UpdateUserRole(ctx context.Context, id uuid.UUID, role model.Role) (model.User, error)
}

type EventModerationRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (eventsmodel.Event, error)
	UpdateModeration(ctx context.Context, id uuid.UUID, st eventsmodel.ModerationStatus, moderatedBy uuid.UUID) (eventsmodel.Event, error)
}

type NotificationSender interface {
	Send(ctx context.Context, n notificationsmodel.Notification) error
}

type Service struct {
	users  UserRepository
	events EventModerationRepository
	notify NotificationSender
}

func New(users UserRepository, events EventModerationRepository, notify NotificationSender) *Service {
	return &Service{users: users, events: events, notify: notify}
}

// ModerateEvent updates moderation status and, on reject, notifies the organizer when known.
func (s *Service) ModerateEvent(ctx context.Context, adminID uuid.UUID, eventIDStr, action, reason string, logger *slog.Logger) (eventsmodel.ModerationStatus, error) {
	if logger == nil {
		logger = slog.Default()
	}
	id, err := uuid.Parse(strings.TrimSpace(eventIDStr))
	if err != nil {
		return "", ErrInvalidEventID
	}
	act := strings.ToLower(strings.TrimSpace(action))
	var st eventsmodel.ModerationStatus
	switch act {
	case "approve":
		st = eventsmodel.ModerationApproved
	case "reject":
		st = eventsmodel.ModerationRejected
	default:
		return "", ErrInvalidAction
	}

	before, err := s.events.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, eventsrepo.ErrNotFound) {
			return "", ErrEventNotFound
		}
		return "", err
	}

	updated, err := s.events.UpdateModeration(ctx, id, st, adminID)
	if err != nil {
		if errors.Is(err, eventsrepo.ErrNotFound) {
			return "", ErrEventNotFound
		}
		return "", err
	}

	logger.Info("event_moderation_audit",
		"admin_id", adminID.String(),
		"event_id", id.String(),
		"action", act,
		"result", "success",
		"moderation_status", string(updated.ModerationStatus),
	)

	if st == eventsmodel.ModerationRejected && before.OrganizerID != nil {
		u, err := s.users.GetUserByID(ctx, *before.OrganizerID)
		if err == nil {
			body := fmt.Sprintf("Your event %q was rejected.", before.Title)
			if r := strings.TrimSpace(reason); r != "" {
				body += " Reason: " + r
			}
			if err := s.notify.Send(ctx, notificationsmodel.Notification{
				Type:  notificationsmodel.NotificationTypeEmail,
				To:    u.Email,
				Title: "Event rejected",
				Body:  body,
			}); err != nil {
				logger.Error("event_reject_notification_failed", "event_id", id.String(), "error", err)
			}
		}
	}

	return updated.ModerationStatus, nil
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
