package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/events/model"
	"github.com/nu/student-event-ticketing-platform/events/repository"
)

type Service struct {
	repo repository.EventRepository
}

func New(repo repository.EventRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, title, description, coverImageURL string, startsAt time.Time, capacityTotal int, priceAmount int64, priceCurrency string, organizerID uuid.UUID) (model.Event, error) {
	org := organizerID
	if priceCurrency == "" {
		priceCurrency = "KZT"
	}
	e := model.Event{
		Title:             title,
		Description:       description,
		CoverImageURL:     coverImageURL,
		StartsAt:          startsAt,
		CapacityTotal:     capacityTotal,
		CapacityAvailable: capacityTotal,
		Status:            model.EventStatusPublished,
		OrganizerID:       &org,
		PriceAmount:       priceAmount,
		PriceCurrency:     strings.ToUpper(priceCurrency),
	}
	return s.repo.Create(ctx, e)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (model.Event, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, filter repository.EventFilter) ([]model.Event, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, title *string, description *string, coverImageURL *string, startsAt *time.Time, capacityTotal *int, priceAmount *int64, priceCurrency *string, status *model.EventStatus) (model.Event, error) {
	patch := repository.EventPatch{
		Title:         title,
		Description:   description,
		CoverImageURL: coverImageURL,
		StartsAt:      startsAt,
		CapacityTotal: capacityTotal,
		Status:        status,
		PriceAmount:   priceAmount,
		PriceCurrency: priceCurrency,
	}
	return s.repo.Update(ctx, id, patch)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
