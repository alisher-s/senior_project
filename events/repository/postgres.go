package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nu/student-event-ticketing-platform/events/model"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func uuidPtrFromPG(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id, err := uuid.FromBytes(u.Bytes[:])
	if err != nil {
		return nil
	}
	return &id
}

func stringPtrFromPG(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func timePtrFromPG(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEventRow(row rowScanner) (model.Event, error) {
	var e model.Event
	var location pgtype.Text
	var endAt pgtype.Timestamptz
	var modBy pgtype.UUID
	var orgID pgtype.UUID
	err := row.Scan(
		&e.ID,
		&e.Title,
		&e.Description,
		&e.CoverImageURL,
		&e.StartsAt,
		&location,
		&endAt,
		&e.CapacityTotal,
		&e.CapacityAvailable,
		&e.Status,
		&e.ModerationStatus,
		&modBy,
		&orgID,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		return model.Event{}, err
	}
	e.Location = stringPtrFromPG(location)
	e.EndAt = timePtrFromPG(endAt)
	e.ModeratedBy = uuidPtrFromPG(modBy)
	e.OrganizerID = uuidPtrFromPG(orgID)
	return e, nil
}

func (p *Postgres) Create(ctx context.Context, e model.Event) (model.Event, error) {
	id := uuid.New()
	st := e.Status
	if st == "" {
		st = model.EventStatusPublished
	}
	var orgID any
	if e.OrganizerID != nil {
		orgID = *e.OrganizerID
	}
	// DB schema expects location to be NOT NULL (empty string when omitted).
	// Some existing local volumes may have NOT NULL without a DEFAULT, so avoid inserting NULL.
	location := ""
	if e.Location != nil {
		location = *e.Location
	}
	var endAt any
	if e.EndAt != nil {
		endAt = *e.EndAt
	}
	row := p.pool.QueryRow(ctx, `
		INSERT INTO events (id, title, description, cover_image_url, starts_at, location, end_at, capacity_total, capacity_available, status, organizer_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, title, description, cover_image_url, starts_at, location, end_at, capacity_total, capacity_available, status,
			moderation_status, moderated_by, organizer_id, created_at, updated_at
	`, id, e.Title, e.Description, e.CoverImageURL, e.StartsAt, location, endAt, e.CapacityTotal, e.CapacityAvailable, st, orgID)
	created, err := scanEventRow(row)
	if err != nil {
		return model.Event{}, err
	}
	return created, nil
}

func (p *Postgres) GetByID(ctx context.Context, id uuid.UUID) (model.Event, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id, title, description, cover_image_url, starts_at, location, end_at, capacity_total, capacity_available, status,
			moderation_status, moderated_by, organizer_id, created_at, updated_at
		FROM events
		WHERE id = $1
	`, id)
	e, err := scanEventRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Event{}, ErrNotFound
		}
		return model.Event{}, err
	}
	return e, nil
}

func (p *Postgres) List(ctx context.Context, filter EventFilter) ([]model.Event, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	where := []string{"1=1"}
	args := []any{}
	argPos := 1

	if filter.RequireApprovedModeration {
		where = append(where, "moderation_status = 'approved'")
	}
	if filter.OrganizerID != nil {
		where = append(where, fmt.Sprintf("organizer_id = $%d", argPos))
		args = append(args, *filter.OrganizerID)
		argPos++
	}
	if filter.ModerationStatus != nil && strings.TrimSpace(*filter.ModerationStatus) != "" {
		where = append(where, fmt.Sprintf("moderation_status = $%d", argPos))
		args = append(args, strings.TrimSpace(*filter.ModerationStatus))
		argPos++
	}
	if strings.TrimSpace(filter.Query) != "" {
		where = append(where, fmt.Sprintf("title ILIKE $%d", argPos))
		args = append(args, "%"+strings.TrimSpace(filter.Query)+"%")
		argPos++
	}
	if filter.StartsAfter != nil {
		where = append(where, fmt.Sprintf("starts_at >= $%d", argPos))
		args = append(args, *filter.StartsAfter)
		argPos++
	}
	if filter.StartsBefore != nil {
		where = append(where, fmt.Sprintf("starts_at <= $%d", argPos))
		args = append(args, *filter.StartsBefore)
		argPos++
	}

	query := fmt.Sprintf(`
		SELECT id, title, description, cover_image_url, starts_at, location, end_at, capacity_total, capacity_available, status,
			moderation_status, moderated_by, organizer_id, created_at, updated_at
		FROM events
		WHERE %s
		ORDER BY starts_at DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(where, " AND "), argPos, argPos+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Event
	for rows.Next() {
		e, err := scanEventRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (p *Postgres) Update(ctx context.Context, id uuid.UUID, patch EventPatch) (model.Event, error) {
	set := []string{}
	args := []any{}
	argPos := 1

	if patch.Title != nil {
		set = append(set, fmt.Sprintf("title = $%d", argPos))
		args = append(args, *patch.Title)
		argPos++
	}
	if patch.Description != nil {
		set = append(set, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *patch.Description)
		argPos++
	}
	if patch.CoverImageURL != nil {
		set = append(set, fmt.Sprintf("cover_image_url = $%d", argPos))
		args = append(args, *patch.CoverImageURL)
		argPos++
	}
	if patch.StartsAt != nil {
		set = append(set, fmt.Sprintf("starts_at = $%d", argPos))
		args = append(args, *patch.StartsAt)
		argPos++
	}
	if patch.Location != nil {
		set = append(set, fmt.Sprintf("location = $%d", argPos))
		args = append(args, *patch.Location)
		argPos++
	}
	if patch.EndAt != nil {
		set = append(set, fmt.Sprintf("end_at = $%d", argPos))
		args = append(args, *patch.EndAt)
		argPos++
	}
	if patch.CapacityTotal != nil {
		patchVal := *patch.CapacityTotal
		patchValPos := argPos
		set = append(set, fmt.Sprintf("capacity_total = $%d", patchValPos))
		args = append(args, patchVal)
		argPos++
		set = append(set, fmt.Sprintf(
			"capacity_available = GREATEST(0, $%d - ("+
				"SELECT COUNT(*) FROM tickets t "+
				"WHERE t.event_id = events.id AND t.status IN ('active','used')"+
				"))",
			patchValPos,
		))
	}
	if patch.Status != nil {
		set = append(set, fmt.Sprintf("status = $%d", argPos))
		args = append(args, string(*patch.Status))
		argPos++
	}

	if len(set) == 0 {
		return p.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE events
		SET %s, updated_at = NOW()
		WHERE id = $%d
		RETURNING id, title, description, cover_image_url, starts_at, location, end_at, capacity_total, capacity_available, status,
			moderation_status, moderated_by, organizer_id, created_at, updated_at
	`, strings.Join(set, ", "), argPos)

	row := p.pool.QueryRow(ctx, query, args...)
	e, err := scanEventRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Event{}, ErrNotFound
		}
		return model.Event{}, err
	}
	return e, nil
}

func (p *Postgres) UpdateCoverImage(ctx context.Context, id uuid.UUID, coverImageURL string) error {
	res, err := p.pool.Exec(ctx, `
		UPDATE events
		SET cover_image_url = $2, updated_at = NOW()
		WHERE id = $1
	`, id, coverImageURL)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) UpdateModeration(ctx context.Context, id uuid.UUID, st model.ModerationStatus, moderatedBy uuid.UUID) (model.Event, error) {
	row := p.pool.QueryRow(ctx, `
		UPDATE events
		SET moderation_status = $2, moderated_by = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, title, description, cover_image_url, starts_at, location, end_at, capacity_total, capacity_available, status,
			moderation_status, moderated_by, organizer_id, created_at, updated_at
	`, id, string(st), moderatedBy)
	e, err := scanEventRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Event{}, ErrNotFound
		}
		return model.Event{}, err
	}
	return e, nil
}

func (p *Postgres) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := p.pool.Exec(ctx, `DELETE FROM events WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
