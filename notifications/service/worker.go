package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/nu/student-event-ticketing-platform/notifications/model"
	notificationsRepo "github.com/nu/student-event-ticketing-platform/notifications/repository"
)

// EmailWorker is a DB-backed worker that periodically dequeues and delivers notifications.
// It handles both email (via Sender) and push (via FCMSender + DeviceTokenRepository).
type EmailWorker struct {
	logger       *slog.Logger
	repo         notificationsRepo.NotificationRepository
	sender       Sender
	fcmSender    *FCMSender
	deviceTokens notificationsRepo.DeviceTokenRepository

	batchSize    int
	pollInterval time.Duration
}

func NewEmailWorker(
	logger *slog.Logger,
	repo notificationsRepo.NotificationRepository,
	sender Sender,
	fcmSender *FCMSender,
	deviceTokens notificationsRepo.DeviceTokenRepository,
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
		fcmSender:    fcmSender,
		deviceTokens: deviceTokens,
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
		var sendErr error
		switch n.Type {
		case model.NotificationTypeEmail:
			sendErr = w.sender.Send(ctx, n.To, n.Title, n.Body)
		case model.NotificationTypePush:
			sendErr = w.sendPush(ctx, n)
		default:
			w.logger.Warn("notifications_unknown_type", "type", n.Type, "notification_id", n.ID)
			_ = w.repo.UpdateStatus(ctx, n.ID, model.NotificationStatusFailed)
			continue
		}

		if sendErr != nil {
			w.logger.Error("notification_send_failed",
				"error", sendErr,
				"notification_id", n.ID,
				"type", n.Type,
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

// sendPush fans out a push notification to all device tokens registered for n.To (user UUID).
// If no tokens are found the notification is silently marked sent — user simply has no device registered.
func (w *EmailWorker) sendPush(ctx context.Context, n model.Notification) error {
	if w.fcmSender == nil || w.deviceTokens == nil {
		return nil
	}

	tokens, err := w.deviceTokens.GetByUserID(ctx, n.To)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}

	var lastErr error
	for _, dt := range tokens {
		if err := w.fcmSender.SendToToken(ctx, dt.Token, n.Title, n.Body); err != nil {
			w.logger.Warn("fcm_send_failed",
				"error", err,
				"token_prefix", dt.Token[:min(12, len(dt.Token))],
				"platform", dt.Platform,
			)
			lastErr = err
		}
	}
	return lastErr
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
