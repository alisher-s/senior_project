package repository

import (
	"context"

	"github.com/nu/student-event-ticketing-platform/notifications/model"
)

type NotificationRepository interface {
	// Enqueue persists notification request for later processing.
	Enqueue(ctx context.Context, n model.Notification) error

	// DequeueBatch atomically claims a batch of queued notifications for processing.
	DequeueBatch(ctx context.Context, limit int) ([]model.Notification, error)

	// UpdateStatus marks the notification processing result.
	UpdateStatus(ctx context.Context, id string, status model.NotificationStatus) error

	// RequeueAfterFailure sets status back to queued with an incremented retry_count (SMTP retry path).
	RequeueAfterFailure(ctx context.Context, id string, newRetryCount int) error

	// GetByUserID returns the most recent notifications for the given user (for in-app history).
	GetByUserID(ctx context.Context, userID string, limit int) ([]model.Notification, error)
}

type DeviceTokenRepository interface {
	// Upsert inserts or updates a device token for the user.
	Upsert(ctx context.Context, userID, token, platform string) error

	// GetByUserID returns all device tokens registered for the user.
	GetByUserID(ctx context.Context, userID string) ([]model.DeviceToken, error)

	// Delete removes a specific token (e.g. on logout or token rotation).
	Delete(ctx context.Context, userID, token string) error
}
