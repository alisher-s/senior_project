package repository

import (
	"context"

	"github.com/nu/student-event-ticketing-platform/analytics/model"
)

// StatsRepository loads analytics aggregates from Postgres.
type StatsRepository interface {
	EventStats(ctx context.Context, params EventStatsParams) (model.EventStats, error)
}
