package authx

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
	UserID string   `json:"user_id"`
	Role   Role     `json:"role"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID string   `json:"user_id"`
	Role   Role     `json:"role"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

type ctxKey string

const (
	ctxUserIDKey ctxKey = "jwt_user_id"
	ctxRoleKey   ctxKey = "jwt_role"
	ctxRolesKey  ctxKey = "jwt_roles"
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

// GenerateAccessToken issues an access JWT. legacyRole mirrors users.role for clients that still read the singular claim.
func (j JWT) GenerateAccessToken(userID uuid.UUID, roles []Role, legacyRole Role) (string, error) {
	now := time.Now().UTC()
	userIDStr := userID.String()
	roleStrs := rolesToStrings(roles)
	claims := AccessClaims{
		UserID: userIDStr,
		Role:   legacyRole,
		Roles:  roleStrs,
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

func (j JWT) GenerateRefreshToken(userID uuid.UUID, roles []Role, legacyRole Role) (string, uuid.UUID, time.Time, error) {
	now := time.Now().UTC()
	userIDStr := userID.String()
	jti := uuid.New()
	expiresAt := now.Add(j.refreshTTL)

	claims := RefreshClaims{
		UserID: userIDStr,
		Role:   legacyRole,
		Roles:  rolesToStrings(roles),
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
	if _, err := effectiveRolesFromClaims(claims.Roles, claims.Role); err != nil {
		return nil, err
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
	if _, err := effectiveRolesFromClaims(claims.Roles, claims.Role); err != nil {
		return nil, err
	}
	return claims, nil
}

// EffectiveAccessRoles returns normalized roles from claims (prefers "roles" array; falls back to legacy "role").
func EffectiveAccessRoles(c *AccessClaims) ([]Role, error) {
	return effectiveRolesFromClaims(c.Roles, c.Role)
}

// EffectiveRefreshRoles returns normalized roles from refresh claims.
func EffectiveRefreshRoles(c *RefreshClaims) ([]Role, error) {
	return effectiveRolesFromClaims(c.Roles, c.Role)
}

func effectiveRolesFromClaims(roles []string, legacy Role) ([]Role, error) {
	var out []Role
	if len(roles) > 0 {
		for _, s := range roles {
			r := Role(s)
			switch r {
			case RoleStudent, RoleOrganizer, RoleAdmin:
				out = append(out, r)
			default:
				return nil, fmt.Errorf("invalid role in token: %q", s)
			}
		}
	} else if legacy != "" {
		switch legacy {
		case RoleStudent, RoleOrganizer, RoleAdmin:
			out = []Role{legacy}
		default:
			return nil, fmt.Errorf("invalid role in token: %q", legacy)
		}
	}
	out = dedupeRolesSorted(out)
	if len(out) == 0 {
		return nil, errors.New("missing roles in token")
	}
	return out, nil
}

func rolesToStrings(roles []Role) []string {
	if len(roles) == 0 {
		return nil
	}
	out := dedupeRolesSorted(slices.Clone(roles))
	s := make([]string, len(out))
	for i, r := range out {
		s[i] = string(r)
	}
	return s
}

func dedupeRolesSorted(roles []Role) []Role {
	if len(roles) <= 1 {
		return roles
	}
	seen := make(map[Role]struct{}, len(roles))
	var uniq []Role
	for _, r := range roles {
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		uniq = append(uniq, r)
	}
	slices.SortFunc(uniq, func(a, b Role) int {
		return cmpString(string(a), string(b))
	})
	return uniq
}

func cmpString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// PrimaryRole picks admin > organizer > student when multiple roles are present (legacy single-role checks).
func PrimaryRole(roles []Role) Role {
	set := make(map[Role]struct{}, len(roles))
	for _, r := range roles {
		set[r] = struct{}{}
	}
	for _, pick := range []Role{RoleAdmin, RoleOrganizer, RoleStudent} {
		if _, ok := set[pick]; ok {
			return pick
		}
	}
	return ""
}

// RolesEqual compares two role slices as sets (order-insensitive).
func RolesEqual(a, b []Role) bool {
	if len(a) != len(b) {
		return false
	}
	ma := make(map[Role]int, len(a))
	for _, r := range a {
		ma[r]++
	}
	for _, r := range b {
		ma[r]--
		if ma[r] < 0 {
			return false
		}
	}
	for _, n := range ma {
		if n != 0 {
			return false
		}
	}
	return true
}

func WithAccessClaims(ctx context.Context, userID uuid.UUID, roles []Role) context.Context {
	ctx = context.WithValue(ctx, ctxUserIDKey, userID)
	ctx = context.WithValue(ctx, ctxRolesKey, roles)
	ctx = context.WithValue(ctx, ctxRoleKey, PrimaryRole(roles))
	return ctx
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(ctxUserIDKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}

// RoleFromContext returns PrimaryRole for backward compatibility with code that expects a single role.
func RoleFromContext(ctx context.Context) (Role, bool) {
	roles, ok := RolesFromContext(ctx)
	if !ok || len(roles) == 0 {
		return "", false
	}
	return PrimaryRole(roles), true
}

func RolesFromContext(ctx context.Context) ([]Role, bool) {
	v := ctx.Value(ctxRolesKey)
	roles, ok := v.([]Role)
	return roles, ok
}

// HasRole reports whether the JWT carries the given role.
func HasRole(ctx context.Context, want Role) bool {
	roles, ok := RolesFromContext(ctx)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}
