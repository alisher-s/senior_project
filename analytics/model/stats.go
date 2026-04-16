package model

import "time"

type EventStats struct {
	EventID   string    `json:"event_id"`
	Tickets   int64     `json:"tickets"`
	Revenue    int64     `json:"revenue"`
	AsOf       time.Time `json:"as_of"`
}

