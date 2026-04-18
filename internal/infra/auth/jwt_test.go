package authx_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	"github.com/nu/student-event-ticketing-platform/internal/config"
)

func testJWT(t *testing.T) authx.JWT {
	t.Helper()
	cfg := config.Config{}
	cfg.JWT.AccessSecret = "test_access_secret_ci_ci_ci_ci"
	cfg.JWT.RefreshSecret = "test_refresh_secret_ci_ci_ci_ci"
	cfg.JWT.AccessTTL = time.Minute
	cfg.JWT.RefreshTTL = time.Hour
	cfg.JWT.Issuer = "nu-ticketing"
	cfg.JWT.Audience = "nu-ticketing-client"
	return authx.NewJWT(cfg)
}

func TestGenerateParseAccessToken_MultiRole(t *testing.T) {
	j := testJWT(t)
	uid := uuid.New()

	s, err := j.GenerateAccessToken(uid, []authx.Role{authx.RoleStudent, authx.RoleOrganizer}, authx.RoleOrganizer)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := j.ParseAccessToken(s)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != uid.String() {
		t.Fatalf("user_id: got %q want %q", claims.UserID, uid)
	}
	eff, err := authx.EffectiveAccessRoles(claims)
	if err != nil {
		t.Fatal(err)
	}
	if len(eff) != 2 {
		t.Fatalf("roles len: got %d want 2: %v", len(eff), eff)
	}
	if authx.PrimaryRole(eff) != authx.RoleOrganizer {
		t.Fatalf("PrimaryRole: got %v want organizer", authx.PrimaryRole(eff))
	}
}

func TestParseAccessToken_LegacySingleRoleClaim(t *testing.T) {
	j := testJWT(t)
	uid := uuid.New()
	now := time.Now().UTC()

	claims := authx.AccessClaims{
		UserID: uid.String(),
		Role:   authx.RoleStudent,
		Roles:  nil,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "nu-ticketing",
			Subject:   uid.String(),
			Audience:  []string{"nu-ticketing-client"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	s, err := tok.SignedString([]byte("test_access_secret_ci_ci_ci_ci"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := j.ParseAccessToken(s)
	if err != nil {
		t.Fatal(err)
	}
	eff, err := authx.EffectiveAccessRoles(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if len(eff) != 1 || eff[0] != authx.RoleStudent {
		t.Fatalf("effective roles: got %v want [student]", eff)
	}
}

func TestRolesEqual(t *testing.T) {
	a := []authx.Role{authx.RoleStudent, authx.RoleOrganizer}
	b := []authx.Role{authx.RoleOrganizer, authx.RoleStudent}
	if !authx.RolesEqual(a, b) {
		t.Fatal("expected equal")
	}
	if authx.RolesEqual(a, []authx.Role{authx.RoleStudent}) {
		t.Fatal("expected not equal")
	}
}

func TestWithAccessClaims_HasRole(t *testing.T) {
	uid := uuid.New()
	ctx := authx.WithAccessClaims(context.Background(), uid, []authx.Role{authx.RoleOrganizer, authx.RoleStudent})
	if !authx.HasRole(ctx, authx.RoleStudent) || !authx.HasRole(ctx, authx.RoleOrganizer) {
		t.Fatal("expected both roles")
	}
	if authx.HasRole(ctx, authx.RoleAdmin) {
		t.Fatal("unexpected admin")
	}
	r, ok := authx.RoleFromContext(ctx)
	if !ok || r != authx.RoleOrganizer {
		t.Fatalf("RoleFromContext (PrimaryRole): got %v ok=%v", r, ok)
	}
}

func TestParseAccessToken_InvalidRoleString(t *testing.T) {
	j := testJWT(t)
	uid := uuid.New()
	now := time.Now().UTC()

	claims := authx.AccessClaims{
		UserID: uid.String(),
		Role:   authx.RoleStudent,
		Roles:  []string{"student", "superuser"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "nu-ticketing",
			Subject:   uid.String(),
			Audience:  []string{"nu-ticketing-client"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	s, err := tok.SignedString([]byte("test_access_secret_ci_ci_ci_ci"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.ParseAccessToken(s); err == nil {
		t.Fatal("expected error for invalid role in array")
	}
}
