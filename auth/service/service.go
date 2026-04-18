package service

import (
	"context"
	"errors"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"

	"github.com/nu/student-event-ticketing-platform/auth/model"
	"github.com/nu/student-event-ticketing-platform/auth/repository"
	"github.com/nu/student-event-ticketing-platform/internal/config"
)

type Service struct {
	cfg            config.Config
	users          repository.UserRepository
	refreshTokens  repository.RefreshTokenRepository
	jwt            authx.JWT
}

func New(
	cfg config.Config,
	users repository.UserRepository,
	refreshTokens repository.RefreshTokenRepository,
	jwt authx.JWT,
) *Service {
	return &Service{
		cfg:           cfg,
		users:         users,
		refreshTokens: refreshTokens,
		jwt:           jwt,
	}
}

func activeRolesAuthx(u model.User) []authx.Role {
	out := make([]authx.Role, len(u.ActiveRoles))
	for i, r := range u.ActiveRoles {
		out[i] = authx.Role(r)
	}
	return out
}

func (s *Service) Register(ctx context.Context, email, password string) (model.User, string, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if err := validateEmail(email); err != nil {
		return model.User{}, "", "", ErrEmailNotAllowed
	}
	if !strings.HasSuffix(email, "@"+strings.ToLower(s.cfg.Auth.NuEmailDomain)) {
		return model.User{}, "", "", ErrEmailNotAllowed
	}
	if len(password) < 8 {
		return model.User{}, "", "", errors.New("password too short")
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.Auth.BcryptCost)
	if err != nil {
		return model.User{}, "", "", err
	}

	u, err := s.users.CreateUser(ctx, email, string(pwHash), model.RoleStudent)
	if err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			return model.User{}, "", "", ErrEmailAlreadyExists
		}
		return model.User{}, "", "", err
	}

	ar := activeRolesAuthx(u)
	access, err := s.jwt.GenerateAccessToken(u.ID, ar, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}

	refresh, jti, expiresAt, err := s.jwt.GenerateRefreshToken(u.ID, ar, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}
	if err := s.refreshTokens.InsertRefreshToken(ctx, jti, u.ID, expiresAt); err != nil {
		return model.User{}, "", "", err
	}

	return u, access, refresh, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (model.User, string, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	u, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.User{}, "", "", ErrInvalidCredentials
		}
		return model.User{}, "", "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return model.User{}, "", "", ErrInvalidCredentials
	}

	ar := activeRolesAuthx(u)
	access, err := s.jwt.GenerateAccessToken(u.ID, ar, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}

	refresh, jti, expiresAt, err := s.jwt.GenerateRefreshToken(u.ID, ar, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}

	if err := s.refreshTokens.RotateRefreshToken(ctx, u.ID, jti, expiresAt); err != nil {
		return model.User{}, "", "", err
	}

	return u, access, refresh, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (model.User, string, string, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	claims, err := s.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}

	tokRoles, err := authx.EffectiveRefreshRoles(claims)
	if err != nil {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}

	if !authx.RolesEqual(tokRoles, activeRolesAuthx(u)) {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}

	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}

	now := time.Now().UTC()
	ok, err := s.refreshTokens.ConsumeRefreshToken(ctx, jti, u.ID, now)
	if err != nil {
		return model.User{}, "", "", err
	}
	if !ok {
		return model.User{}, "", "", ErrRefreshTokenConsumed
	}

	ar := activeRolesAuthx(u)
	access, err := s.jwt.GenerateAccessToken(u.ID, ar, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}

	refresh, newJti, expiresAt, err := s.jwt.GenerateRefreshToken(u.ID, ar, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}
	if err := s.refreshTokens.InsertRefreshToken(ctx, newJti, u.ID, expiresAt); err != nil {
		return model.User{}, "", "", err
	}

	return u, access, refresh, nil
}

// RequestOrganizerRole records a pending organizer role for the caller (student self-service).
func (s *Service) RequestOrganizerRole(ctx context.Context, userID uuid.UUID, requested []string) error {
	if len(requested) != 1 || requested[0] != string(model.RoleOrganizer) {
		return ErrOrganizerRequestInvalidBody
	}

	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if !slices.Contains(u.ActiveRoles, model.RoleStudent) {
		return ErrOrganizerRequestNotAllowed
	}
	if slices.Contains(u.ActiveRoles, model.RoleOrganizer) {
		return ErrOrganizerAlreadyActive
	}
	if slices.Contains(u.PendingRoles, model.RoleOrganizer) {
		return nil
	}

	return s.users.EnsureOrganizerRolePending(ctx, userID)
}

// UserByID returns the user with active and pending roles (for /auth/me responses).
func (s *Service) UserByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	return s.users.GetUserByID(ctx, id)
}

func validateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	return err
}
