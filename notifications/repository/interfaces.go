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
}

