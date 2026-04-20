package service

import (
	"context"

	"github.com/nu/student-event-ticketing-platform/notifications/model"
	"github.com/nu/student-event-ticketing-platform/notifications/repository"
)

type Service struct {
	repo repository.NotificationRepository
}

func New(repo repository.NotificationRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Send(ctx context.Context, n model.Notification) error {
	// Enqueue into persistent notifications_queue; async worker will deliver.
	return s.repo.Enqueue(ctx, n)
}

func (s *Service) GetByUserID(ctx context.Context, userID string, limit int) ([]model.Notification, error) {
	return s.repo.GetByUserID(ctx, userID, limit)
}

