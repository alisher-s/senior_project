package handler

import (
	"time"

	"github.com/nu/student-event-ticketing-platform/analytics/model"
)

// EventStatsResponseDTO documents the JSON shape for GET /analytics/events/stats.
// total_capacity, registered_count, and remaining_capacity satisfy the relationship described on model.EventStats.
type EventStatsResponseDTO struct {
	EventID *string `json:"event_id,omitempty"`

	TotalCapacity     int64 `json:"total_capacity"`
	RegisteredCount   int64 `json:"registered_count"`
	RemainingCapacity int64 `json:"remaining_capacity"`

	RegistrationTimeline []model.RegistrationHour `json:"registration_timeline"`

	AsOf time.Time `json:"as_of"`
}
