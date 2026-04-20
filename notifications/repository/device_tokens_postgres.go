package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nu/student-event-ticketing-platform/notifications/model"
)

type DeviceTokenPostgres struct {
	pool *pgxpool.Pool
}

func NewDeviceTokenPostgres(pool *pgxpool.Pool) *DeviceTokenPostgres {
	return &DeviceTokenPostgres{pool: pool}
}

func (r *DeviceTokenPostgres) Upsert(ctx context.Context, userID, token, platform string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO device_tokens (user_id, token, platform)
		VALUES ($1::uuid, $2, $3)
		ON CONFLICT (user_id, token) DO UPDATE SET platform = EXCLUDED.platform
	`, userID, token, platform)
	return err
}

func (r *DeviceTokenPostgres) GetByUserID(ctx context.Context, userID string) ([]model.DeviceToken, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, token, platform, created_at
		FROM device_tokens
		WHERE user_id = $1::uuid
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.DeviceToken
	for rows.Next() {
		var dt model.DeviceToken
		if err := rows.Scan(&dt.ID, &dt.UserID, &dt.Token, &dt.Platform, &dt.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, dt)
	}
	return out, rows.Err()
}

func (r *DeviceTokenPostgres) Delete(ctx context.Context, userID, token string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM device_tokens WHERE user_id = $1::uuid AND token = $2
	`, userID, token)
	return err
}
