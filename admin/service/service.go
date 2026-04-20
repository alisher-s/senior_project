package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/admin/model"
	"github.com/nu/student-event-ticketing-platform/admin/repository"
	authmodel "github.com/nu/student-event-ticketing-platform/auth/model"
	eventsmodel "github.com/nu/student-event-ticketing-platform/events/model"
	eventsrepo "github.com/nu/student-event-ticketing-platform/events/repository"
	notificationsmodel "github.com/nu/student-event-ticketing-platform/notifications/model"
)

type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (authmodel.User, error)
	UpdateUserRole(ctx context.Context, id uuid.UUID, role authmodel.Role) (authmodel.User, error)
}

type ModerationLogRepository interface {
	InsertModerationLog(ctx context.Context, adminUserID uuid.UUID, eventID, action, reason string) error
	ListModerationLogs(ctx context.Context, filter repository.ModerationLogFilter) ([]model.ModerationLog, error)
}

type EventModerationRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (eventsmodel.Event, error)
	UpdateModeration(ctx context.Context, id uuid.UUID, st eventsmodel.ModerationStatus, moderatedBy uuid.UUID) (eventsmodel.Event, error)
}

type NotificationSender interface {
	Send(ctx context.Context, n notificationsmodel.Notification) error
}

type Service struct {
	users     UserRepository
	events    EventModerationRepository
	moderation ModerationLogRepository
	notify    NotificationSender
}

func New(users UserRepository, events EventModerationRepository, moderation ModerationLogRepository, notify NotificationSender) *Service {
	return &Service{users: users, events: events, moderation: moderation, notify: notify}
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

	if err := s.moderation.InsertModerationLog(ctx, adminID, id.String(), act, strings.TrimSpace(reason)); err != nil {
		logger.Error("moderation_log_insert_failed", "admin_id", adminID.String(), "event_id", id.String(), "error", err)
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
func (s *Service) SetUserRole(ctx context.Context, adminID, targetUserID uuid.UUID, role string, logger *slog.Logger) (authmodel.User, error) {
	if logger == nil {
		logger = slog.Default()
	}
	r, err := normalizeRole(role)
	if err != nil {
		return authmodel.User{}, err
	}
	u, err := s.users.UpdateUserRole(ctx, targetUserID, r)
	if err != nil {
		return authmodel.User{}, err
	}

	reason := fmt.Sprintf("target_user_id=%s;new_role=%s", targetUserID.String(), string(r))
	if err := s.moderation.InsertModerationLog(ctx, adminID, "", "patch_user_role", reason); err != nil {
		logger.Error("moderation_log_insert_failed", "admin_id", adminID.String(), "target_user_id", targetUserID.String(), "error", err)
	}

	return u, nil
}

// ListModerationLogs returns paginated moderation audit rows.
func (s *Service) ListModerationLogs(ctx context.Context, filter repository.ModerationLogFilter) ([]model.ModerationLog, error) {
	return s.moderation.ListModerationLogs(ctx, filter)
}

func normalizeRole(s string) (authmodel.Role, error) {
	r := authmodel.Role(strings.ToLower(strings.TrimSpace(s)))
	switch r {
	case authmodel.RoleStudent, authmodel.RoleOrganizer, authmodel.RoleAdmin:
		return r, nil
	default:
		return "", ErrInvalidRole
	}
}
