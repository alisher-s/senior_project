package model

import "time"

type DeviceToken struct {
	ID        string
	UserID    string
	Token     string
	Platform  string
	CreatedAt time.Time
}
