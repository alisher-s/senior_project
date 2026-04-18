package model

import "time"

// RegistrationHour is one bucket in the registration timeline (UTC hour start).
type RegistrationHour struct {
	Hour  time.Time `json:"hour"`
	Count int64     `json:"count"`
}

// EventStats aggregates ticketing metrics. When EventID is nil, totals span all events
// in the caller’s scope (organizer: own events; admin: entire platform).
//
// Capacity fields are aligned within the same snapshot: registered_count is tickets with
// status active or used in scope; remaining_capacity is max(0, total_capacity − registered_count).
// When registered_count ≤ total_capacity, total_capacity = registered_count + remaining_capacity.
type EventStats struct {
	EventID *string `json:"event_id,omitempty"`

	TotalCapacity     int64 `json:"total_capacity"`      // Sum or single event capacity_total
	RegisteredCount   int64 `json:"registered_count"`    // Non-cancelled tickets (active + used) in scope
	RemainingCapacity int64 `json:"remaining_capacity"` // Derived from total − registered (see package comment)

	RegistrationTimeline []RegistrationHour `json:"registration_timeline"`

	AsOf time.Time `json:"as_of"`
}
