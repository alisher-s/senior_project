package repository

import (
	"context"
	"errors"
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

	var u model.User
	// Note: role is validated in service; repository keeps a simple insert returning values.
	err := p.pool.QueryRow(ctx, `
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
	return u, nil
}

func (p *Postgres) UpdateUserRole(ctx context.Context, id uuid.UUID, role model.Role) (model.User, error) {
	if err := p.RevokeTokensByUserID(ctx, id); err != nil {
		return model.User{}, err
	}

	var u model.User
	err := p.pool.QueryRow(ctx, `
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
	return u, nil
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
	// Serialize concurrent logins for the same user:
	// 1) Lock user's row with FOR UPDATE
	// 2) Revoke existing refresh tokens
	// 3) Insert new refresh token
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

	// Lock the user row to ensure concurrent login requests do not overlap.
	var lockedID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&lockedID); err != nil {
		return err
	}
	_ = lockedID

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
	// Single-use token consumption:
	// - revoked_at must be NULL
	// - token must not be expired
	// The update is atomic; exactly one call succeeds.
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

