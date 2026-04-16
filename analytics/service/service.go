package service

import (
	"context"

	"github.com/nu/student-event-ticketing-platform/analytics/model"
)

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) EventStats(ctx context.Context, eventID *string) (model.EventStats, error) {
	_ = ctx
	_ = eventID
	return model.EventStats{}, ErrNotImplemented
}

