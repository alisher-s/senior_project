package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nu/student-event-ticketing-platform/analytics/model"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

// remainingCapacityDerived returns seats left implied by total capacity minus registrations
// counted in this response, clamped at zero. Keeps total_capacity = registered_count +
// remaining_capacity for the same snapshot even if events.capacity_available drifted.
func remainingCapacityDerived(totalCap, registered int64) int64 {
	if registered >= totalCap {
		return 0
	}
	return totalCap - registered
}

// EventStatsParams scopes stats to the caller. When EventID is set, stats are for that
// event (organizer must own it unless IsAdmin). When nil, stats aggregate events in scope
// (organizer: own events; admin: all events).
type EventStatsParams struct {
	CallerID uuid.UUID
	IsAdmin  bool
	EventID  *uuid.UUID
}

func (p *Postgres) EventStats(ctx context.Context, params EventStatsParams) (model.EventStats, error) {
	if params.EventID != nil {
		return p.eventStatsForEvent(ctx, *params.EventID, params.CallerID, params.IsAdmin)
	}
	return p.eventStatsAggregate(ctx, params.CallerID, params.IsAdmin)
}

// statsQuerier is implemented by *pgxpool.Pool and pgx.Tx so timeline + counts can share one connection.
type statsQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (p *Postgres) eventStatsForEvent(ctx context.Context, eventID, callerID uuid.UUID, isAdmin bool) (model.EventStats, error) {
	// One REPEATABLE READ transaction so capacity, registration counts, and timeline
	// rows all read the same snapshot (default READ COMMITTED can differ per statement).
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return model.EventStats{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	now := time.Now().UTC()

	var capTotal int64
	var orgPG pgtype.UUID
	err = tx.QueryRow(ctx, `
		SELECT capacity_total::bigint, organizer_id
		FROM events
		WHERE id = $1
	`, eventID).Scan(&capTotal, &orgPG)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.EventStats{}, ErrNotFound
		}
		return model.EventStats{}, err
	}
	if !isAdmin {
		if !orgPG.Valid {
			return model.EventStats{}, ErrForbidden
		}
		oid, err := uuid.FromBytes(orgPG.Bytes[:])
		if err != nil || oid != callerID {
			return model.EventStats{}, ErrForbidden
		}
	}

	var regCount int64
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*)::bigint
		FROM tickets
		WHERE event_id = $1 AND status IN ('active', 'used')
	`, eventID).Scan(&regCount)
	if err != nil {
		return model.EventStats{}, err
	}

	sid := eventID.String()
	timeline, err := registrationTimeline(ctx, tx, timelineFilter{EventID: &eventID, CallerID: callerID, IsAdmin: isAdmin}, now)
	if err != nil {
		return model.EventStats{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.EventStats{}, err
	}
	committed = true

	return model.EventStats{
		EventID:               &sid,
		TotalCapacity:         capTotal,
		RegisteredCount:       regCount,
		RemainingCapacity:     remainingCapacityDerived(capTotal, regCount),
		RegistrationTimeline:  timeline,
		AsOf:                  now,
	}, nil
}

func (p *Postgres) eventStatsAggregate(ctx context.Context, callerID uuid.UUID, isAdmin bool) (model.EventStats, error) {
	// Same snapshot semantics as eventStatsForEvent (see comment there).
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return model.EventStats{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	now := time.Now().UTC()

	var where string
	var args []any
	if isAdmin {
		where = "1=1"
	} else {
		where = "organizer_id = $1"
		args = append(args, callerID)
	}

	var capTotal int64
	q := `SELECT COALESCE(SUM(capacity_total), 0)::bigint FROM events WHERE ` + where
	err = tx.QueryRow(ctx, q, args...).Scan(&capTotal)
	if err != nil {
		return model.EventStats{}, err
	}

	// Registered count is total registrations (active + used), not limited to the 7-day timeline window.
	var regQ string
	var regArgs []any
	if isAdmin {
		regQ = `SELECT COUNT(*)::bigint FROM tickets t WHERE t.status IN ('active', 'used')`
		regArgs = nil
	} else {
		regQ = `
			SELECT COUNT(*)::bigint
			FROM tickets t
			INNER JOIN events e ON e.id = t.event_id
			WHERE t.status IN ('active', 'used') AND e.organizer_id = $1
		`
		regArgs = []any{callerID}
	}

	var regCount int64
	err = tx.QueryRow(ctx, regQ, regArgs...).Scan(&regCount)
	if err != nil {
		return model.EventStats{}, err
	}

	timeline, err := registrationTimeline(ctx, tx, timelineFilter{EventID: nil, CallerID: callerID, IsAdmin: isAdmin}, now)
	if err != nil {
		return model.EventStats{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.EventStats{}, err
	}
	committed = true

	return model.EventStats{
		EventID:               nil,
		TotalCapacity:         capTotal,
		RegisteredCount:       regCount,
		RemainingCapacity:     remainingCapacityDerived(capTotal, regCount),
		RegistrationTimeline:  timeline,
		AsOf:                  now,
	}, nil
}

type timelineFilter struct {
	EventID  *uuid.UUID
	CallerID uuid.UUID
	IsAdmin  bool
}

func registrationTimeline(ctx context.Context, qry statsQuerier, f timelineFilter, now time.Time) ([]model.RegistrationHour, error) {
	since := now.Add(-7 * 24 * time.Hour)

	var q string
	var args []any
	if f.EventID != nil {
		q = `
			SELECT date_trunc('hour', t.created_at, 'UTC') AS bucket, COUNT(*)::bigint
			FROM tickets t
			WHERE t.event_id = $1
			  AND t.status IN ('active', 'used')
			  AND t.created_at >= $2
			GROUP BY 1
			ORDER BY 1 ASC
		`
		args = append(args, *f.EventID, since)
	} else if f.IsAdmin {
		q = `
			SELECT date_trunc('hour', t.created_at, 'UTC') AS bucket, COUNT(*)::bigint
			FROM tickets t
			WHERE t.status IN ('active', 'used')
			  AND t.created_at >= $1
			GROUP BY 1
			ORDER BY 1 ASC
		`
		args = append(args, since)
	} else {
		q = `
			SELECT date_trunc('hour', t.created_at, 'UTC') AS bucket, COUNT(*)::bigint
			FROM tickets t
			INNER JOIN events e ON e.id = t.event_id
			WHERE e.organizer_id = $1
			  AND t.status IN ('active', 'used')
			  AND t.created_at >= $2
			GROUP BY 1
			ORDER BY 1 ASC
		`
		args = append(args, f.CallerID, since)
	}

	rows, err := qry.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.RegistrationHour
	for rows.Next() {
		var bucket time.Time
		var cnt int64
		if err := rows.Scan(&bucket, &cnt); err != nil {
			return nil, err
		}
		out = append(out, model.RegistrationHour{Hour: bucket.UTC(), Count: cnt})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []model.RegistrationHour{}
	}
	return out, nil
}

var _ StatsRepository = (*Postgres)(nil)

