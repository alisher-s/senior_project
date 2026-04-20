package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nu/student-event-ticketing-platform/payments/model"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) CreateInitiation(ctx context.Context, pm model.Payment) (model.Payment, error) {
	now := time.Now().UTC()

	if pm.ProviderRef == "" {
		return model.Payment{}, errors.New("missing provider_ref")
	}

	id := uuid.New()
	var created model.Payment
	err := p.pool.QueryRow(ctx, `
		INSERT INTO payments (
			id, user_id, event_id, amount, currency, status, provider_name, provider_ref, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, user_id, event_id, amount, currency, status, provider_name, provider_ref, created_at, updated_at
	`, id, pm.UserID, pm.EventID, pm.Amount, pm.Currency, pm.Status, pm.ProviderName, pm.ProviderRef, now, now).Scan(
		&created.ID,
		&created.UserID,
		&created.EventID,
		&created.Amount,
		&created.Currency,
		&created.Status,
		&created.ProviderName,
		&created.ProviderRef,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.Payment{}, ErrProviderRefExists
		}
		return model.Payment{}, err
	}
	return created, nil
}

func (p *Postgres) GetByProviderRef(ctx context.Context, providerRef string) (model.Payment, error) {
	var out model.Payment
	err := p.pool.QueryRow(ctx, `
		SELECT id, user_id, event_id, amount, currency, status, provider_name, provider_ref, created_at, updated_at
		FROM payments
		WHERE provider_ref = $1
	`, providerRef).Scan(
		&out.ID,
		&out.UserID,
		&out.EventID,
		&out.Amount,
		&out.Currency,
		&out.Status,
		&out.ProviderName,
		&out.ProviderRef,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Payment{}, ErrPaymentNotFound
		}
		return model.Payment{}, err
	}
	return out, nil
}

func (p *Postgres) UpdateStatusByProviderRef(ctx context.Context, providerRef string, status model.PaymentStatus, now time.Time) (model.Payment, error) {
	var out model.Payment
	err := p.pool.QueryRow(ctx, `
		UPDATE payments
		SET status = $2,
			updated_at = $3
		WHERE provider_ref = $1
		RETURNING id, user_id, event_id, amount, currency, status, provider_name, provider_ref, created_at, updated_at
	`, providerRef, status, now).Scan(
		&out.ID,
		&out.UserID,
		&out.EventID,
		&out.Amount,
		&out.Currency,
		&out.Status,
		&out.ProviderName,
		&out.ProviderRef,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Payment{}, ErrPaymentNotFound
		}
		return model.Payment{}, err
	}
	return out, nil
}
