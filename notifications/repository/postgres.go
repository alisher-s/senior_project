package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nu/student-event-ticketing-platform/notifications/model"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (p *Postgres) Enqueue(ctx context.Context, n model.Notification) error {
	now := time.Now().UTC()
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = now
	}

	// user_id is optional: store NULL when empty string.
	var userID *string
	if n.UserID != "" {
		userID = &n.UserID
	}

	_, err := p.pool.Exec(ctx, `
		INSERT INTO notifications_queue (id, type, recipient, title, body, status, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'queued', $6::uuid, $7, NOW())
	`, n.ID, n.Type, n.To, n.Title, n.Body, userID, n.CreatedAt)
	return err
}

func (p *Postgres) DequeueBatch(ctx context.Context, limit int) ([]model.Notification, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := p.pool.Query(ctx, `
		WITH claimed AS (
			SELECT id, type, recipient, title, body, retry_count, created_at,
			       COALESCE(user_id::text, '') AS user_id
			FROM notifications_queue
			WHERE status = 'queued'
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE notifications_queue nq
		SET status = 'processing', updated_at = NOW()
		FROM claimed
		WHERE nq.id = claimed.id
		RETURNING nq.id, nq.type, nq.recipient, nq.title, nq.body, nq.retry_count, nq.created_at,
		          COALESCE(nq.user_id::text, '') AS user_id
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.To, &n.Title, &n.Body, &n.RetryCount, &n.CreatedAt, &n.UserID); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (p *Postgres) UpdateStatus(ctx context.Context, id string, status model.NotificationStatus) error {
	res, err := p.pool.Exec(ctx, `
		UPDATE notifications_queue
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, id, status)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (p *Postgres) RequeueAfterFailure(ctx context.Context, id string, newRetryCount int) error {
	res, err := p.pool.Exec(ctx, `
		UPDATE notifications_queue
		SET status = 'queued', retry_count = $2, updated_at = NOW()
		WHERE id = $1
	`, id, newRetryCount)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (p *Postgres) GetByUserID(ctx context.Context, userID string, limit int) ([]model.Notification, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := p.pool.Query(ctx, `
		SELECT id, type, recipient, title, body, retry_count, created_at,
		       COALESCE(user_id::text, '') AS user_id
		FROM notifications_queue
		WHERE user_id = $1::uuid
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.To, &n.Title, &n.Body, &n.RetryCount, &n.CreatedAt, &n.UserID); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
