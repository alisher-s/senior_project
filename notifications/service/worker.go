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

	batchSize    int
	pollInterval time.Duration
}

func NewEmailWorker(
	logger *slog.Logger,
	repo notificationsRepo.NotificationRepository,
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
		// Stub async sender.
		w.logger.Info("email_send_stub",
			"to", n.To,
			"title", n.Title,
			"notification_id", n.ID,
		)
		time.Sleep(10 * time.Millisecond) // simulate IO

		if err := w.repo.UpdateStatus(ctx, n.ID, model.NotificationStatusSent); err != nil {
			w.logger.Error("notifications_mark_sent_failed", "notification_id", n.ID, "error", err)
		}
	}
}

