package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/auth/model"
	"github.com/nu/student-event-ticketing-platform/auth/service"
	"github.com/nu/student-event-ticketing-platform/internal/config"
	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
)

type stubUserRepo struct {
	byID        model.User
	getErr      error
	ensureCalls int
	ensureUID   uuid.UUID
}

func (s *stubUserRepo) CreateUser(ctx context.Context, email string, passwordHash string, role model.Role) (model.User, error) {
	panic("not used")
}

func (s *stubUserRepo) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	panic("not used")
}

func (s *stubUserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	if s.getErr != nil {
		return model.User{}, s.getErr
	}
	u := s.byID
	u.ID = id
	return u, nil
}

func (s *stubUserRepo) UpdateUserRole(ctx context.Context, id uuid.UUID, role model.Role) (model.User, error) {
	panic("not used")
}

func (s *stubUserRepo) ListUsers(ctx context.Context, q string, limit, offset int) ([]model.User, error) {
	panic("not used")
}

func (s *stubUserRepo) EnsureOrganizerRolePending(ctx context.Context, userID uuid.UUID) error {
	s.ensureCalls++
	s.ensureUID = userID
	return nil
}

type noopRefresh struct{}

func (noopRefresh) RevokeTokensByUserID(ctx context.Context, userID uuid.UUID) error { return nil }

func (noopRefresh) InsertRefreshToken(ctx context.Context, jti uuid.UUID, userID uuid.UUID, expiresAt time.Time) error {
	return nil
}

func (noopRefresh) RotateRefreshToken(ctx context.Context, userID uuid.UUID, jti uuid.UUID, expiresAt time.Time) error {
	return nil
}

func (noopRefresh) ConsumeRefreshToken(ctx context.Context, jti uuid.UUID, userID uuid.UUID, now time.Time) (bool, error) {
	return false, nil
}

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

func TestRequestOrganizerRole_InvalidBody(t *testing.T) {
	stub := &stubUserRepo{byID: model.User{ActiveRoles: []model.Role{model.RoleStudent}}}
	svc := service.New(config.Config{}, stub, noopRefresh{}, testJWT(t))
	uid := uuid.New()

	if err := svc.RequestOrganizerRole(context.Background(), uid, []string{"admin"}); !errors.Is(err, service.ErrOrganizerRequestInvalidBody) {
		t.Fatalf("got %v want ErrOrganizerRequestInvalidBody", err)
	}
	if stub.ensureCalls != 0 {
		t.Fatalf("EnsureOrganizerRolePending calls: %d", stub.ensureCalls)
	}
}

func TestRequestOrganizerRole_OnlyStudentsMayRequest(t *testing.T) {
	stub := &stubUserRepo{byID: model.User{ActiveRoles: []model.Role{model.RoleOrganizer}}}
	svc := service.New(config.Config{}, stub, noopRefresh{}, testJWT(t))
	uid := uuid.New()

	if err := svc.RequestOrganizerRole(context.Background(), uid, []string{"organizer"}); !errors.Is(err, service.ErrOrganizerRequestNotAllowed) {
		t.Fatalf("got %v want ErrOrganizerRequestNotAllowed", err)
	}
	if stub.ensureCalls != 0 {
		t.Fatalf("EnsureOrganizerRolePending calls: %d", stub.ensureCalls)
	}
}

func TestRequestOrganizerRole_AlreadyActive(t *testing.T) {
	stub := &stubUserRepo{byID: model.User{
		ActiveRoles: []model.Role{model.RoleStudent, model.RoleOrganizer},
	}}
	svc := service.New(config.Config{}, stub, noopRefresh{}, testJWT(t))
	uid := uuid.New()

	if err := svc.RequestOrganizerRole(context.Background(), uid, []string{"organizer"}); !errors.Is(err, service.ErrOrganizerAlreadyActive) {
		t.Fatalf("got %v want ErrOrganizerAlreadyActive", err)
	}
	if stub.ensureCalls != 0 {
		t.Fatalf("EnsureOrganizerRolePending calls: %d", stub.ensureCalls)
	}
}

func TestRequestOrganizerRole_IdempotentWhenPending(t *testing.T) {
	stub := &stubUserRepo{byID: model.User{
		ActiveRoles:  []model.Role{model.RoleStudent},
		PendingRoles: []model.Role{model.RoleOrganizer},
	}}
	svc := service.New(config.Config{}, stub, noopRefresh{}, testJWT(t))
	uid := uuid.New()

	if err := svc.RequestOrganizerRole(context.Background(), uid, []string{"organizer"}); err != nil {
		t.Fatal(err)
	}
	if stub.ensureCalls != 0 {
		t.Fatalf("expected no DB write when already pending, got %d calls", stub.ensureCalls)
	}
}

func TestRequestOrganizerRole_InsertsPending(t *testing.T) {
	stub := &stubUserRepo{byID: model.User{
		ActiveRoles: []model.Role{model.RoleStudent},
	}}
	svc := service.New(config.Config{}, stub, noopRefresh{}, testJWT(t))
	uid := uuid.New()

	if err := svc.RequestOrganizerRole(context.Background(), uid, []string{"organizer"}); err != nil {
		t.Fatal(err)
	}
	if stub.ensureCalls != 1 || stub.ensureUID != uid {
		t.Fatalf("EnsureOrganizerRolePending: calls=%d uid=%v want uid=%v", stub.ensureCalls, stub.ensureUID, uid)
	}
}
