package handler

import "time"

// EventStatsResponseDTO documents the JSON shape returned when analytics is implemented (currently the handler returns 501).
type EventStatsResponseDTO struct {
	EventID string    `json:"event_id"`
	Tickets int64     `json:"tickets"`
	Revenue int64     `json:"revenue"`
	AsOf    time.Time `json:"as_of"`
}
