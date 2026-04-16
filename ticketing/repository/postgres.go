package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nu/student-event-ticketing-platform/ticketing/model"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) RegisterTicket(ctx context.Context, userID uuid.UUID, eventID uuid.UUID, qrHashHex string, now time.Time) (model.Ticket, error) {
	// Critical concurrency decision:
	// We use a single transaction + conditional UPDATE to prevent overbooking.
	// Why DB lock (vs Redis atomic decrement) for foundation:
	// - guarantees consistency between capacity decrement and ticket insert
	// - fewer moving parts (no Redis script coordination)
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.Ticket{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// 1) Atomically decrement capacity if available.
	ct, err := tx.Exec(ctx, `
		UPDATE events
		SET capacity_available = capacity_available - 1,
			updated_at = NOW()
		WHERE id = $1 AND capacity_available > 0
	`, eventID)
	if err != nil {
		return model.Ticket{}, err
	}
	if ct.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM events WHERE id = $1)`, eventID).Scan(&exists); err != nil {
			return model.Ticket{}, err
		}
		if !exists {
			return model.Ticket{}, ErrEventNotFound
		}
		return model.Ticket{}, ErrCapacityFull
	}

	// 2) Insert ticket. If user already has ticket for this event, rollback capacity decrement.
	ticketID := uuid.New()
	var t model.Ticket
	err = tx.QueryRow(ctx, `
		INSERT INTO tickets (id, event_id, user_id, status, qr_hash_hex, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_id, user_id) DO NOTHING
		RETURNING id, event_id, user_id, status, qr_hash_hex, created_at
	`, ticketID, eventID, userID, model.TicketStatusActive, qrHashHex, now).Scan(
		&t.ID,
		&t.EventID,
		&t.UserID,
		&t.Status,
		&t.QRHashHex,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Ticket{}, ErrAlreadyRegistered
		}
		return model.Ticket{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Ticket{}, err
	}
	committed = true
	return t, nil
}

func (p *Postgres) GetByEventAndUser(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) (model.Ticket, error) {
	var t model.Ticket
	err := p.pool.QueryRow(ctx, `
		SELECT id, event_id, user_id, status, qr_hash_hex, created_at
		FROM tickets
		WHERE event_id = $1 AND user_id = $2
	`, eventID, userID).Scan(
		&t.ID,
		&t.EventID,
		&t.UserID,
		&t.Status,
		&t.QRHashHex,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Ticket{}, ErrTicketNotFound
		}
		return model.Ticket{}, err
	}
	return t, nil
}

func (p *Postgres) CancelTicket(ctx context.Context, userID uuid.UUID, ticketID uuid.UUID, now time.Time) (model.Ticket, error) {
	// Transactional lifecycle update + deterministic capacity recomputation.
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.Ticket{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var t model.Ticket
	err = tx.QueryRow(ctx, `
		SELECT id, event_id, user_id, status, qr_hash_hex, created_at
		FROM tickets
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, ticketID, userID).Scan(
		&t.ID,
		&t.EventID,
		&t.UserID,
		&t.Status,
		&t.QRHashHex,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Ticket{}, ErrTicketNotFound
		}
		return model.Ticket{}, err
	}

	if t.Status == model.TicketStatusCancelled {
		return model.Ticket{}, ErrTicketAlreadyCancelled
	}

	// Mark ticket as cancelled (allowed only for active/used).
	ct, err := tx.Exec(ctx, `
		UPDATE tickets
		SET status = $1
		WHERE id = $2 AND user_id = $3 AND status IN ('active','used')
	`, model.TicketStatusCancelled, ticketID, userID)
	if err != nil {
		return model.Ticket{}, err
	}
	if ct.RowsAffected() == 0 {
		// Ticket might have been cancelled/modified concurrently.
		// We keep error mapping simple here.
		return model.Ticket{}, ErrTicketAlreadyCancelled
	}

	// Recompute availability as CapacityTotal - COUNT(active + used).
	_, err = tx.Exec(ctx, `
		UPDATE events e
		SET capacity_available = GREATEST(0, e.capacity_total - (
			SELECT COUNT(*) FROM tickets t
			WHERE t.event_id = e.id AND t.status IN ('active','used')
		)),
		updated_at = NOW()
		WHERE e.id = $1
	`, t.EventID)
	if err != nil {
		return model.Ticket{}, err
	}

	// Refresh updated ticket row for response.
	var updated model.Ticket
	err = tx.QueryRow(ctx, `
		SELECT id, event_id, user_id, status, qr_hash_hex, created_at
		FROM tickets
		WHERE id = $1
	`, ticketID).Scan(
		&updated.ID,
		&updated.EventID,
		&updated.UserID,
		&updated.Status,
		&updated.QRHashHex,
		&updated.CreatedAt,
	)
	if err != nil {
		return model.Ticket{}, err
	}

	_ = now // reserved if we later add updated_at on tickets
	if err := tx.Commit(ctx); err != nil {
		return model.Ticket{}, err
	}
	committed = true
	return updated, nil
}

func (p *Postgres) UseTicketByQRHash(ctx context.Context, qrHashHex string, now time.Time) (model.Ticket, error) {
	// Transactional lifecycle update + deterministic capacity recomputation.
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.Ticket{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var t model.Ticket
	err = tx.QueryRow(ctx, `
		SELECT id, event_id, user_id, status, qr_hash_hex, created_at
		FROM tickets
		WHERE qr_hash_hex = $1
		FOR UPDATE
	`, qrHashHex).Scan(
		&t.ID,
		&t.EventID,
		&t.UserID,
		&t.Status,
		&t.QRHashHex,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Ticket{}, ErrTicketNotFound
		}
		return model.Ticket{}, err
	}

	if t.Status == model.TicketStatusUsed {
		return model.Ticket{}, ErrTicketAlreadyUsed
	}
	if t.Status == model.TicketStatusCancelled {
		return model.Ticket{}, ErrTicketCannotBeUsed
	}

	// Mark ticket as used (active -> used).
	_, err = tx.Exec(ctx, `
		UPDATE tickets
		SET status = $1
		WHERE id = $2 AND status = 'active'
	`, model.TicketStatusUsed, t.ID)
	if err != nil {
		return model.Ticket{}, err
	}

	// Recompute availability (active + used always occupy capacity).
	_, err = tx.Exec(ctx, `
		UPDATE events e
		SET capacity_available = GREATEST(0, e.capacity_total - (
			SELECT COUNT(*) FROM tickets t
			WHERE t.event_id = e.id AND t.status IN ('active','used')
		)),
		updated_at = NOW()
		WHERE e.id = $1
	`, t.EventID)
	if err != nil {
		return model.Ticket{}, err
	}

	var updated model.Ticket
	err = tx.QueryRow(ctx, `
		SELECT id, event_id, user_id, status, qr_hash_hex, created_at
		FROM tickets
		WHERE id = $1
	`, t.ID).Scan(
		&updated.ID,
		&updated.EventID,
		&updated.UserID,
		&updated.Status,
		&updated.QRHashHex,
		&updated.CreatedAt,
	)
	if err != nil {
		return model.Ticket{}, err
	}

	_ = now // reserved if we later add updated_at on tickets
	if err := tx.Commit(ctx); err != nil {
		return model.Ticket{}, err
	}
	committed = true
	return updated, nil
}

// Compile-time guarantee that this repository satisfies the interface.
var _ interface {
	RegisterTicket(ctx context.Context, userID uuid.UUID, eventID uuid.UUID, qrHashHex string, now time.Time) (model.Ticket, error)
	GetByEventAndUser(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) (model.Ticket, error)
	CancelTicket(ctx context.Context, userID uuid.UUID, ticketID uuid.UUID, now time.Time) (model.Ticket, error)
	UseTicketByQRHash(ctx context.Context, qrHashHex string, now time.Time) (model.Ticket, error)
} = (*Postgres)(nil)

