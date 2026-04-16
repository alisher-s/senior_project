package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/events/model"
)

type EventRepository interface {
	Create(ctx context.Context, e model.Event) (model.Event, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Event, error)
	List(ctx context.Context, filter EventFilter) ([]model.Event, error)
	Update(ctx context.Context, id uuid.UUID, patch EventPatch) (model.Event, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type EventFilter struct {
	Query          string
	StartsAfter    *time.Time
	StartsBefore   *time.Time
	Limit          int
	Offset         int
}

type EventPatch struct {
	Title             *string
	Description       *string
	StartsAt          *time.Time
	CapacityTotal     *int
}

