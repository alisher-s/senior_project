package authx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/internal/config"
)

type Role string

const (
	RoleStudent   Role = "student"
	RoleOrganizer Role = "organizer"
	RoleAdmin     Role = "admin"
)

type AccessClaims struct {
	UserID string `json:"user_id"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID string `json:"user_id"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}

type ctxKey string

const (
	ctxUserIDKey ctxKey = "jwt_user_id"
	ctxRoleKey   ctxKey = "jwt_role"
)

func NewJWT(cfg config.Config) JWT {
	return JWT{
		accessSecret:  []byte(cfg.JWT.AccessSecret),
		refreshSecret: []byte(cfg.JWT.RefreshSecret),
		accessTTL:     cfg.JWT.AccessTTL,
		refreshTTL:    cfg.JWT.RefreshTTL,
		issuer:        cfg.JWT.Issuer,
		audience:      cfg.JWT.Audience,
	}
}

type JWT struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
	audience      string
}

func (j JWT) GenerateAccessToken(userID uuid.UUID, role Role) (string, error) {
	now := time.Now().UTC()
	userIDStr := userID.String()
	claims := AccessClaims{
		UserID: userIDStr,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userIDStr,
			Audience:  []string{j.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTTL)),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(j.accessSecret)
}

func (j JWT) GenerateRefreshToken(userID uuid.UUID, role Role) (string, uuid.UUID, time.Time, error) {
	now := time.Now().UTC()
	userIDStr := userID.String()
	jti := uuid.New()
	expiresAt := now.Add(j.refreshTTL)

	claims := RefreshClaims{
		UserID: userIDStr,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userIDStr,
			Audience:  []string{j.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti.String(),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(j.refreshSecret)
	return s, jti, expiresAt, err
}

func (j JWT) ParseAccessToken(tokenStr string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected jwt signing method: %T", t.Method)
		}
		return j.accessSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims.UserID == "" {
		return nil, errors.New("missing user_id in access token")
	}
	if claims.Issuer != j.issuer {
		return nil, fmt.Errorf("invalid issuer in access token: %q", claims.Issuer)
	}
	audOK := false
	for _, a := range claims.Audience {
		if a == j.audience {
			audOK = true
			break
		}
	}
	if !audOK {
		return nil, fmt.Errorf("invalid audience in access token")
	}
	switch claims.Role {
	case RoleStudent, RoleOrganizer, RoleAdmin:
	default:
		return nil, fmt.Errorf("invalid role in access token: %q", claims.Role)
	}
	return claims, nil
}

func (j JWT) ParseRefreshToken(tokenStr string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected jwt signing method: %T", t.Method)
		}
		return j.refreshSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims.UserID == "" {
		return nil, errors.New("missing user_id in refresh token")
	}
	if claims.ID == "" {
		return nil, errors.New("missing refresh jti")
	}
	if claims.Issuer != j.issuer {
		return nil, fmt.Errorf("invalid issuer in refresh token: %q", claims.Issuer)
	}
	audOK := false
	for _, a := range claims.Audience {
		if a == j.audience {
			audOK = true
			break
		}
	}
	if !audOK {
		return nil, fmt.Errorf("invalid audience in refresh token")
	}
	switch claims.Role {
	case RoleStudent, RoleOrganizer, RoleAdmin:
	default:
		return nil, fmt.Errorf("invalid role in refresh token: %q", claims.Role)
	}
	return claims, nil
}

func WithAccessClaims(ctx context.Context, userID uuid.UUID, role Role) context.Context {
	ctx = context.WithValue(ctx, ctxUserIDKey, userID)
	ctx = context.WithValue(ctx, ctxRoleKey, role)
	return ctx
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(ctxUserIDKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}

func RoleFromContext(ctx context.Context) (Role, bool) {
	v := ctx.Value(ctxRoleKey)
	role, ok := v.(Role)
	return role, ok
}

