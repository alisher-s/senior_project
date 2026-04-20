package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/nu/student-event-ticketing-platform/notifications/model"
	notificationsRepo "github.com/nu/student-event-ticketing-platform/notifications/repository"
)

// EmailWorker is a DB-backed worker that periodically dequeues queued notifications.
type EmailWorker struct {
	logger *slog.Logger
	repo   notificationsRepo.NotificationRepository
	sender Sender

	batchSize    int
	pollInterval time.Duration
}

func NewEmailWorker(
	logger *slog.Logger,
	repo notificationsRepo.NotificationRepository,
	sender Sender,
	batchSize int,
	pollInterval time.Duration,
) *EmailWorker {
	if batchSize <= 0 {
		batchSize = 20
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	return &EmailWorker{
		logger:       logger,
		repo:         repo,
		sender:       sender,
		batchSize:    batchSize,
		pollInterval: pollInterval,
	}
}

func (w *EmailWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *EmailWorker) processBatch(ctx context.Context) {
	batch, err := w.repo.DequeueBatch(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("notifications_dequeue_failed", "error", err)
		return
	}

	for _, n := range batch {
		// notifications_queue.body is treated as HTML (text/html).
		// Attachment bytes are not stored in the queue; pass nil for now.
		if err := w.sender.SendEmail(ctx, n.To, n.Title, n.Body, nil); err != nil {
			w.logger.Error("email_send_failed",
				"error", err,
				"notification_id", n.ID,
				"to", n.To,
				"retry_count", n.RetryCount,
			)
			if n.RetryCount < 3 {
				next := n.RetryCount + 1
				if err := w.repo.RequeueAfterFailure(ctx, n.ID, next); err != nil {
					w.logger.Error("notifications_requeue_failed", "notification_id", n.ID, "error", err)
				}
				continue
			}
			if err := w.repo.UpdateStatus(ctx, n.ID, model.NotificationStatusFailed); err != nil {
				w.logger.Error("notifications_mark_failed_failed", "notification_id", n.ID, "error", err)
			}
			continue
		}

		if err := w.repo.UpdateStatus(ctx, n.ID, model.NotificationStatusSent); err != nil {
			w.logger.Error("notifications_mark_sent_failed", "notification_id", n.ID, "error", err)
		}
	}
}

