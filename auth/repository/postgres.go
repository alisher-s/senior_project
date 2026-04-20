package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nu/student-event-ticketing-platform/auth/model"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) CreateUser(ctx context.Context, email string, passwordHash string, role model.Role) (model.User, error) {
	id := uuid.New()

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.User{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var u model.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, role, created_at, updated_at
	`, id, email, passwordHash, string(role)).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.User{}, ErrEmailAlreadyExists
		}
		return model.User{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role, status)
		VALUES ($1, $2, 'active')
	`, u.ID, string(role))
	if err != nil {
		return model.User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.User{}, err
	}
	committed = true

	if err := p.mergeRoleRows(ctx, &u); err != nil {
		return model.User{}, err
	}
	return u, nil
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := p.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, err
	}
	if err := p.mergeRoleRows(ctx, &u); err != nil {
		return model.User{}, err
	}
	return u, nil
}

func (p *Postgres) GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	var u model.User
	err := p.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, err
	}
	if err := p.mergeRoleRows(ctx, &u); err != nil {
		return model.User{}, err
	}
	return u, nil
}

func (p *Postgres) ListUsers(ctx context.Context, q string, limit, offset int) ([]model.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	where := []string{"1=1"}
	args := []any{}
	argPos := 1

	if strings.TrimSpace(q) != "" {
		where = append(where, fmt.Sprintf("email ILIKE $%d", argPos))
		args = append(args, "%"+strings.TrimSpace(q)+"%")
		argPos++
	}

	query := fmt.Sprintf(`
		SELECT id, email, role, created_at, updated_at
		FROM users
		WHERE %s
		ORDER BY email ASC
		LIMIT $%d OFFSET $%d
	`, strings.Join(where, " AND "), argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.User, 0, limit)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		// Intentionally omit password_hash for list endpoints.
		u.PasswordHash = ""
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (p *Postgres) mergeRoleRows(ctx context.Context, u *model.User) error {
	rows, err := p.pool.Query(ctx, `
		SELECT role, status FROM user_roles WHERE user_id = $1
	`, u.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var active, pending []model.Role
	for rows.Next() {
		var roleStr, status string
		if err := rows.Scan(&roleStr, &status); err != nil {
			return err
		}
		r := model.Role(roleStr)
		switch status {
		case "active":
			active = append(active, r)
		case "pending":
			pending = append(pending, r)
		default:
			continue
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(active) == 0 {
		// No active user_roles row (legacy row missing or only pending rows): fall back to users.role.
		u.ActiveRoles = []model.Role{u.Role}
	} else {
		u.ActiveRoles = active
	}
	u.PendingRoles = pending
	return nil
}

func (p *Postgres) UpdateUserRole(ctx context.Context, id uuid.UUID, role model.Role) (model.User, error) {
	if err := p.RevokeTokensByUserID(ctx, id); err != nil {
		return model.User{}, err
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.User{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var u model.User
	err = tx.QueryRow(ctx, `
		UPDATE users
		SET role = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, password_hash, role, created_at, updated_at
	`, id, string(role)).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, err
	}

	if err := syncUserRolesInTx(ctx, tx, id, role); err != nil {
		return model.User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.User{}, err
	}
	committed = true

	if err := p.mergeRoleRows(ctx, &u); err != nil {
		return model.User{}, err
	}
	return u, nil
}

func syncUserRolesInTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, role model.Role) error {
	switch role {
	case model.RoleStudent:
		if _, err := tx.Exec(ctx, `
			DELETE FROM user_roles WHERE user_id = $1 AND role IN ('organizer', 'admin')
		`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role, status) VALUES ($1, 'student', 'active')
			ON CONFLICT (user_id, role) DO UPDATE SET status = 'active'
		`, userID)
		return err

	case model.RoleOrganizer:
		if _, err := tx.Exec(ctx, `
			DELETE FROM user_roles WHERE user_id = $1 AND role = 'admin'
		`, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role, status) VALUES ($1, 'student', 'active')
			ON CONFLICT (user_id, role) DO UPDATE SET status = 'active'
		`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role, status) VALUES ($1, 'organizer', 'active')
			ON CONFLICT (user_id, role) DO UPDATE SET status = 'active'
		`, userID)
		return err

	case model.RoleAdmin:
		if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role, status) VALUES ($1, 'admin', 'active')
		`, userID)
		return err

	default:
		return errors.New("unsupported role")
	}
}

func (p *Postgres) EnsureOrganizerRolePending(ctx context.Context, userID uuid.UUID) error {
	tag, err := p.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role, status) VALUES ($1, 'organizer', 'pending')
		ON CONFLICT (user_id, role) DO UPDATE SET
			status = CASE WHEN user_roles.status = 'active' THEN user_roles.status ELSE 'pending' END
	`, userID)
	if err != nil {
		return err
	}
	_ = tag
	return nil
}

func (p *Postgres) RevokeTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	return err
}

func (p *Postgres) InsertRefreshToken(ctx context.Context, jti uuid.UUID, userID uuid.UUID, expiresAt time.Time) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (jti, user_id, expires_at, revoked_at)
		VALUES ($1, $2, $3, NULL)
	`, jti, userID, expiresAt)
	return err
}

func (p *Postgres) RotateRefreshToken(ctx context.Context, userID uuid.UUID, jti uuid.UUID, expiresAt time.Time) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var lockedID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&lockedID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO refresh_tokens (jti, user_id, expires_at, revoked_at)
		VALUES ($1, $2, $3, NULL)
	`, jti, userID, expiresAt); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (p *Postgres) ConsumeRefreshToken(ctx context.Context, jti uuid.UUID, userID uuid.UUID, now time.Time) (bool, error) {
	cmdTag := p.pool.QueryRow(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $4
		WHERE jti = $1
		  AND user_id = $2
		  AND revoked_at IS NULL
		  AND expires_at > $3
		RETURNING 1
	`, jti, userID, now, now)

	var one int
	err := cmdTag.Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return one == 1, nil
}
