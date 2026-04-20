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
	// Single transaction: lock the event row (FOR UPDATE), enforce status/time/capacity, decrement, insert ticket.
	// SERIALIZABLE is not required: one mutex row per event serializes registrants.
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

	var startsAt time.Time
	var evStatus string
	var capAvail int
	var modStatus string
	err = tx.QueryRow(ctx, `
		SELECT starts_at, status, capacity_available, moderation_status
		FROM events
		WHERE id = $1
		FOR UPDATE
	`, eventID).Scan(&startsAt, &evStatus, &capAvail, &modStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Ticket{}, ErrEventNotFound
		}
		return model.Ticket{}, err
	}

	switch evStatus {
	case "published":
	case "cancelled":
		return model.Ticket{}, ErrEventCancelled
	default:
		return model.Ticket{}, ErrEventNotPublished
	}

	if modStatus != "approved" {
		return model.Ticket{}, ErrEventNotApproved
	}

	if !startsAt.After(now) {
		return model.Ticket{}, ErrEventRegistrationClosed
	}

	if capAvail <= 0 {
		return model.Ticket{}, ErrCapacityFull
	}

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
		return model.Ticket{}, ErrCapacityFull
	}

	// Insert ticket. Conflict only if another active/used row exists (see partial unique index).
	ticketID := uuid.New()
	var t model.Ticket
	err = tx.QueryRow(ctx, `
		INSERT INTO tickets (id, event_id, user_id, status, qr_hash_hex, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_id, user_id) WHERE (status IN ('active', 'used')) DO NOTHING
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
		       e.title, e.starts_at, e.location
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
		var row model.TicketWithEvent
		if err := rows.Scan(
			&row.ID,
			&row.EventID,
			&row.UserID,
			&row.Status,
			&row.QRHashHex,
			&row.CreatedAt,
			&row.EventTitle,
			&row.EventStartsAt,
			&row.EventLocation,
		); err != nil {
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
	if err := tx.QueryRow(ctx, `
		SELECT status, starts_at FROM events WHERE id = $1 FOR UPDATE
	`, t.EventID).Scan(&evStatus, &evStarts); err != nil {
		return model.Ticket{}, err
	}
	if evStatus == "cancelled" {
		return model.Ticket{}, ErrTicketCannotBeUsed
	}
	if evStarts.After(now) {
		return model.Ticket{}, ErrCheckInNotOpenYet
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

func (p *Postgres) GetActiveTicketHolderEmails(ctx context.Context, eventID uuid.UUID) ([]string, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT u.email
		FROM tickets t
		JOIN users u ON u.id = t.user_id
		WHERE t.event_id = $1 AND t.status IN ('active', 'used')
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	return emails, rows.Err()
}

// Compile-time guarantee that this repository satisfies the interface.
var _ interface {
	RegisterTicket(ctx context.Context, userID uuid.UUID, eventID uuid.UUID, qrHashHex string, now time.Time) (model.Ticket, error)
	GetByEventAndUser(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) (model.Ticket, error)
	GetUserTickets(ctx context.Context, userID uuid.UUID) ([]model.TicketWithEvent, error)
	CancelTicket(ctx context.Context, userID uuid.UUID, ticketID uuid.UUID, now time.Time, allowAfterEventStart bool) (model.Ticket, error)
	UseTicketByQRHash(ctx context.Context, qrHashHex string, now time.Time) (model.Ticket, error)
	GetActiveTicketHolderEmails(ctx context.Context, eventID uuid.UUID) ([]string, error)
} = (*Postgres)(nil)

