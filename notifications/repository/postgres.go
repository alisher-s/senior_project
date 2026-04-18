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

	_, err := p.pool.Exec(ctx, `
		INSERT INTO notifications_queue (id, type, recipient, title, body, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'queued', $6, NOW())
	`, n.ID, n.Type, n.To, n.Title, n.Body, n.CreatedAt)
	return err
}

func (p *Postgres) DequeueBatch(ctx context.Context, limit int) ([]model.Notification, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := p.pool.Query(ctx, `
		WITH claimed AS (
			SELECT id, type, recipient, title, body, retry_count, created_at
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
		RETURNING nq.id, nq.type, nq.recipient, nq.title, nq.body, nq.retry_count, nq.created_at
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.To, &n.Title, &n.Body, &n.RetryCount, &n.CreatedAt); err != nil {
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
		// Worker will treat missing notifications as idempotent no-op.
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

