package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nu/student-event-ticketing-platform/admin/model"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

// InsertModerationLog inserts a row into admin_moderation_logs. Pass eventID empty when the action is not tied to an event (e.g. role change).
func (p *Postgres) InsertModerationLog(ctx context.Context, adminUserID uuid.UUID, eventID, action, reason string) error {
	id := uuid.New()
	var evID any
	if strings.TrimSpace(eventID) != "" {
		eid, err := uuid.Parse(strings.TrimSpace(eventID))
		if err != nil {
			return fmt.Errorf("invalid event id: %w", err)
		}
		evID = eid
	} else {
		evID = nil
	}

	var reasonArg any
	if strings.TrimSpace(reason) != "" {
		reasonArg = strings.TrimSpace(reason)
	} else {
		reasonArg = nil
	}

	_, err := p.pool.Exec(ctx, `
		INSERT INTO admin_moderation_logs (id, admin_user_id, event_id, action, reason)
		VALUES ($1, $2, $3, $4, $5)
	`, id, adminUserID, evID, strings.TrimSpace(action), reasonArg)
	return err
}

// ModerationLogFilter selects rows from admin_moderation_logs.
type ModerationLogFilter struct {
	EventID *uuid.UUID
	AdminID *uuid.UUID
	Limit   int
	Offset  int
}

// ListModerationLogs returns moderation log rows newest first.
func (p *Postgres) ListModerationLogs(ctx context.Context, filter ModerationLogFilter) ([]model.ModerationLog, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	where := []string{"1=1"}
	args := []any{}
	argPos := 1

	if filter.EventID != nil {
		where = append(where, fmt.Sprintf("event_id = $%d", argPos))
		args = append(args, *filter.EventID)
		argPos++
	}
	if filter.AdminID != nil {
		where = append(where, fmt.Sprintf("admin_user_id = $%d", argPos))
		args = append(args, *filter.AdminID)
		argPos++
	}

	query := fmt.Sprintf(`
		SELECT id, admin_user_id, event_id, action, reason, created_at
		FROM admin_moderation_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(where, " AND "), argPos, argPos+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.ModerationLog
	for rows.Next() {
		var m model.ModerationLog
		var evID pgtype.UUID
		var reason pgtype.Text
		if err := rows.Scan(&m.ID, &m.AdminUserID, &evID, &m.Action, &reason, &m.CreatedAt); err != nil {
			return nil, err
		}
		if evID.Valid {
			uid, err := uuid.FromBytes(evID.Bytes[:])
			if err == nil {
				m.EventID = &uid
			}
		}
		if reason.Valid {
			s := reason.String
			m.Reason = &s
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
