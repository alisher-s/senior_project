package model

import "time"

type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypePush  NotificationType = "push"
)

type NotificationStatus string

const (
	NotificationStatusQueued     NotificationStatus = "queued"
	NotificationStatusProcessing NotificationStatus = "processing"
	NotificationStatusSent       NotificationStatus = "sent"
	NotificationStatusFailed     NotificationStatus = "failed"
)

type Notification struct {
	ID         string
	Type       NotificationType
	To         string
	UserID     string // optional: UUID of the owning user; used for push dispatch and in-app history
	Title      string
	Body       string
	RetryCount int
	CreatedAt  time.Time
}
