package service

import "context"

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) ModerateEvent(ctx context.Context, eventID string, action string) error {
	_ = ctx
	_ = eventID
	_ = action
	return ErrNotImplemented
}

