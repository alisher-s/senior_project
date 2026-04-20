package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nu/student-event-ticketing-platform/internal/infra/db"
	"github.com/nu/student-event-ticketing-platform/ticketing/model"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) RegisterTicket(ctx context.Context, userID uuid.UUID, eventID uuid.UUID, qrHashHex string, now time.Time) (model.Ticket, error) {
	// Single transaction: lock the event row (FOR UPDATE), enforce status/time/capacity, insert ticket, decrement capacity.
	// SERIALIZABLE is not required: one mutex row per event serializes registrants.
	var out model.Ticket
	err := db.WithTx(ctx, p.pool, func(tx pgx.Tx) error {
		var startsAt time.Time
		var evStatus string
		var capAvail int
		var modStatus string
		if err := tx.QueryRow(ctx, `
			SELECT starts_at, status, capacity_available, moderation_status
			FROM events
			WHERE id = $1
			FOR UPDATE
		`, eventID).Scan(&startsAt, &evStatus, &capAvail, &modStatus); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrEventNotFound
			}
			return err
		}

		switch evStatus {
		case "published":
		case "cancelled":
			return ErrEventCancelled
		default:
			return ErrEventNotPublished
		}

		if modStatus != "approved" {
			return ErrEventNotApproved
		}

		if !startsAt.After(now) {
			return ErrEventRegistrationClosed
		}

		if capAvail <= 0 {
			return ErrCapacityFull
		}

		// Insert ticket first so we only consume capacity on success.
		ticketID := uuid.New()
		if err := tx.QueryRow(ctx, `
			INSERT INTO tickets (id, event_id, user_id, status, qr_hash_hex, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (event_id, user_id) WHERE (status IN ('active', 'used')) DO NOTHING
			RETURNING id, event_id, user_id, status, qr_hash_hex, created_at
		`, ticketID, eventID, userID, model.TicketStatusActive, qrHashHex, now).Scan(
			&out.ID,
			&out.EventID,
			&out.UserID,
			&out.Status,
			&out.QRHashHex,
			&out.CreatedAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAlreadyRegistered
			}
			return err
		}

		// Decrement capacity while still holding the event row lock.
		ct, err := tx.Exec(ctx, `
			UPDATE events
			SET capacity_available = capacity_available - 1,
				updated_at = NOW()
			WHERE id = $1 AND capacity_available > 0
		`, eventID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			// Should be extremely rare since we already checked capAvail under FOR UPDATE,
			// but keep it for safety (and to handle unexpected triggers/constraints).
			return ErrCapacityFull
		}

		return nil
	})
	if err != nil {
		return model.Ticket{}, err
	}
	return out, nil
}

func (p *Postgres) GetByEventAndUser(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) (model.Ticket, error) {
	var t model.Ticket
	err := p.pool.QueryRow(ctx, `
		SELECT id, event_id, user_id, status, qr_hash_hex, created_at
		FROM tickets
		WHERE event_id = $1 AND user_id = $2 AND status IN ('active', 'used')
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

func (p *Postgres) GetUserTickets(ctx context.Context, userID uuid.UUID) ([]model.TicketWithEvent, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT t.id, t.event_id, t.user_id, t.status, t.qr_hash_hex, t.created_at,
		       e.title, e.starts_at, e.end_at, e.location
		FROM tickets t
		INNER JOIN events e ON e.id = t.event_id
		WHERE t.user_id = $1 AND t.status IN ('active', 'used')
		ORDER BY e.starts_at ASC, t.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.TicketWithEvent
	for rows.Next() {
		row, err := scanTicketWithEventRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []model.TicketWithEvent{}
	}
	return out, nil
}

func scanTicketWithEventRow(row interface {
	Scan(dest ...any) error
}) (model.TicketWithEvent, error) {
	var r model.TicketWithEvent
	var endAt pgtype.Timestamptz
	if err := row.Scan(
		&r.ID,
		&r.EventID,
		&r.UserID,
		&r.Status,
		&r.QRHashHex,
		&r.CreatedAt,
		&r.EventTitle,
		&r.EventStartsAt,
		&endAt,
		&r.EventLocation,
	); err != nil {
		return model.TicketWithEvent{}, err
	}
	if endAt.Valid {
		t := endAt.Time
		r.EventEndsAt = &t
	}
	return r, nil
}

func (p *Postgres) CancelTicket(ctx context.Context, userID uuid.UUID, ticketID uuid.UUID, now time.Time, allowAfterEventStart bool) (model.Ticket, error) {
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

	var evStarts time.Time
	if err := tx.QueryRow(ctx, `
		SELECT starts_at FROM events WHERE id = $1 FOR UPDATE
	`, t.EventID).Scan(&evStarts); err != nil {
		return model.Ticket{}, err
	}
	if !allowAfterEventStart && !evStarts.After(now) {
		return model.Ticket{}, ErrCancellationNotAllowed
	}

	// Mark ticket as cancelled (attendee flow: active tickets only; used passes are final).
	ct, err := tx.Exec(ctx, `
		UPDATE tickets
		SET status = $1
		WHERE id = $2 AND user_id = $3 AND status = 'active'
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

	var evStatus string
	var evStarts time.Time
	var evEnds pgtype.Timestamptz
	if err := tx.QueryRow(ctx, `
		SELECT status, starts_at, end_at FROM events WHERE id = $1 FOR UPDATE
	`, t.EventID).Scan(&evStatus, &evStarts, &evEnds); err != nil {
		return model.Ticket{}, err
	}
	if evStatus == "cancelled" {
		return model.Ticket{}, ErrTicketCannotBeUsed
	}
	if evStarts.After(now) {
		return model.Ticket{}, ErrCheckInNotOpenYet
	}
	var endPtr *time.Time
	if evEnds.Valid {
		t := evEnds.Time
		endPtr = &t
	}
	if now.After(model.EventEndInstant(evStarts, endPtr)) {
		return model.Ticket{}, ErrTicketExpired
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
	GetUserTickets(ctx context.Context, userID uuid.UUID) ([]model.TicketWithEvent, error)
	CancelTicket(ctx context.Context, userID uuid.UUID, ticketID uuid.UUID, now time.Time, allowAfterEventStart bool) (model.Ticket, error)
	UseTicketByQRHash(ctx context.Context, qrHashHex string, now time.Time) (model.Ticket, error)
} = (*Postgres)(nil)
